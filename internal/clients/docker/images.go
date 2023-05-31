package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/docker/cli/cli/command/inspect"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/distribution/reference"
	dt "github.com/docker/docker/api/types"
	dtf "github.com/docker/docker/api/types/filters"
	dtr "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pkg/errors"
)

type Image struct {
	Repository string
	ID         string
	Inspect    dt.ImageInspect
}

func (c *Client) ListImages(ctx context.Context) ([]Image, error) {
	imageSummaries, err := c.Client.ImageList(ctx, dt.ImageListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't list Docker images")
	}
	imageNames := make([]string, 0, len(imageSummaries))
	images := make(map[string]Image)
	for _, summary := range imageSummaries {
		image := Image{
			ID: strings.TrimPrefix(summary.ID, "sha256:")[:12],
		}
		if len(summary.RepoDigests) > 0 {
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
		return c.Client.ImageInspectWithRaw(ctx, imageHash)
	}
	if err := inspect.Inspect(buffer, []string{imageHash}, "", getRefFunc); err != nil {
		return Image{}, errors.Wrapf(
			err, "couldn't get more detailed information about image %s", imageHash,
		)
	}
	inspect := []dt.ImageInspect{}
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

func (c *Client) PruneUnusedImages(ctx context.Context) (dt.ImagesPruneReport, error) {
	return c.Client.ImagesPrune(ctx, dtf.Args{})
}

func CompareDeletedImages(i, j dt.ImageDeleteResponseItem) int {
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
	ctx context.Context, taggedName string, outStream *streams.Out,
) (trust.ImageRefAndAuth, error) {
	// This function is adapted from the github.com/docker/cli/cli/command/image
	// package's RunPull function, which is licensed under Apache-2.0. This function was changed to
	// assume that the name is already tagged and normalized and that no auth or content trust image
	// verification is needed.
	distributionRef, err := reference.ParseNormalizedNamed(taggedName)
	switch {
	case err != nil:
		return trust.ImageRefAndAuth{}, err
	case reference.IsNameOnly(distributionRef):
		return trust.ImageRefAndAuth{}, errors.Errorf(
			"image %s must be specified with a tag", taggedName,
		)
	}

	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, authResolver, taggedName)
	if err != nil {
		return trust.ImageRefAndAuth{}, errors.Wrapf(
			err, "couldn't look up ref of image %s", taggedName,
		)
	}

	if err = c.pullImage(ctx, imgRefAndAuth, outStream); err != nil {
		return trust.ImageRefAndAuth{}, err
	}

	return imgRefAndAuth, nil
}

func NewOutStream(out io.Writer) *streams.Out {
	return streams.NewOut(out)
}

func authResolver(ctx context.Context, index *dtr.IndexInfo) dtr.AuthConfig {
	return dtr.AuthConfig{}
}

func (c *Client) pullImage(
	ctx context.Context, imgRefAndAuth trust.ImageRefAndAuth, out *streams.Out,
) (err error) {
	// This function is adapted from the github.com/docker/cli/cli/command/image
	// package's imagePullPrivileged function, which is licensed under Apache-2.0. This function was
	// changed so that it doesn't use Docker's CLI and gives up immediately if the operation is
	// unauthorized.
	encodedAuth, err := dtr.EncodeAuthConfig(*imgRefAndAuth.AuthConfig())
	if err != nil {
		return err
	}
	responseBody, err := c.Client.ImagePull(
		ctx, reference.FamiliarString(imgRefAndAuth.Reference()), dt.ImagePullOptions{
			RegistryAuth: encodedAuth,
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

	return jsonmessage.DisplayJSONMessagesToStream(responseBody, out, nil)
}
