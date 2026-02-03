package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/docker/cli/cli/streams"
	dti "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type Image struct {
	Repository string
	Tag        string
	ID         string
	Inspect    dti.InspectResponse
}

// docker image ls

func (c *Client) ListImages(ctx context.Context, matchName string) ([]Image, error) {
	listOptions := client.ImageListOptions{}
	if matchName != "" {
		listOptions.Filters = make(client.Filters).Add("reference", matchName)
	}

	result, err := c.Client.ImageList(ctx, listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't list Docker images")
	}
	imageNames := make([]string, 0, len(result.Items))
	images := make(map[string]Image)
	for _, summary := range result.Items {
		image := Image{
			ID: strings.TrimPrefix(summary.ID, "sha256:")[:12],
		}
		if len(summary.RepoTags) > 0 {
			parts := strings.Split(summary.RepoTags[0], "@")
			if len(parts) > 1 {
				image.Tag = parts[1]
			}
			if len(parts) > 0 {
				image.Repository = parts[0]
			}
		} else if len(summary.RepoDigests) > 0 {
			parts := strings.Split(summary.RepoDigests[0], "@")
			if len(parts) > 0 {
				image.Repository = parts[0]
			}
		}
		name := image.ID
		imageNames = append(imageNames, name)
		images[name] = image
	}

	orderedImages := make([]Image, 0, len(imageNames))
	for _, name := range imageNames {
		orderedImages = append(orderedImages, images[name])
	}
	return orderedImages, nil
}

func (c *Client) InspectImage(ctx context.Context, imageHash string) (Image, error) {
	buffer := &bytes.Buffer{}
	getRefFunc := func(imageHash string) (interface{}, []byte, error) {
		raw := bytes.Buffer{}
		response, err := c.Client.ImageInspect(ctx, imageHash, client.ImageInspectWithRawResponse(&raw))
		return response, raw.Bytes(), err
	}
	if err := inspect.Inspect(buffer, []string{imageHash}, "", getRefFunc); err != nil {
		return Image{}, errors.Wrapf(
			err, "couldn't get more detailed information about image %s", imageHash,
		)
	}
	inspect := []dti.InspectResponse{}
	if err := json.Unmarshal(buffer.Bytes(), &inspect); err != nil {
		return Image{}, errors.Wrapf(
			err, "couldn't parse detailed information about image %s", imageHash,
		)
	}
	if len(inspect) != 1 {
		return Image{}, errors.Errorf("inspection response has unexpected length %d", len(inspect))
	}
	image := Image{
		ID:      inspect[0].ID,
		Inspect: inspect[0],
	}
	if len(image.Inspect.RepoDigests) > 0 {
		parts := strings.Split(image.Inspect.RepoDigests[0], "@")
		if len(parts) > 0 {
			image.Repository = parts[0]
		}
	}
	return image, nil
}

func (c *Client) PruneUnusedImages(ctx context.Context) (client.ImagePruneResult, error) {
	return c.Client.ImagePrune(ctx, client.ImagePruneOptions{
		Filters: make(client.Filters).Add("dangling", "false"),
	})
}

func CompareDeletedImages(i, j dti.DeleteResponse) int {
	switch {
	default:
		return 0
	case i.Untagged != "" && j.Untagged == "":
		return -1
	case i.Untagged == "" && j.Untagged != "":
		return 1
	case i.Untagged < j.Untagged:
		return -1
	case i.Untagged > j.Untagged:
		return 1
	case i.Deleted < j.Deleted:
		return -1
	case i.Deleted > j.Deleted:
		return 1
	}
}

func (c *Client) PullImage(
	ctx context.Context, taggedName, platform string, outStream *streams.Out,
) error {
	// This function is adapted from the github.com/docker/cli/cli/command/image
	// package's runPull function, which is licensed under Apache-2.0. This function was changed to
	// assume that the name is already tagged and normalized and that no auth or content trust image
	// verification is needed.
	distributionRef, err := reference.ParseNormalizedNamed(taggedName)
	switch {
	case err != nil:
		return err
	case reference.IsNameOnly(distributionRef):
		return errors.Errorf("image %s must be specified with a tag", taggedName)
	}

	parsedPlatform, err := platforms.Parse(platform)
	if err != nil {
		return err
	}

	responseBody, err := c.Client.ImagePull(
		ctx, reference.FamiliarString(distributionRef), client.ImagePullOptions{
			Platforms: []ocispec.Platform{parsedPlatform},
		},
	)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := responseBody.Close(); cerr != nil {
			if err == nil {
				err = cerr
			}
		}
	}()

	return jsonmessage.DisplayJSONMessagesToStream(responseBody, outStream, nil)
}

func NewOutStream(out io.Writer) *streams.Out {
	return streams.NewOut(out)
}
