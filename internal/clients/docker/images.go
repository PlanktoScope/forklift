package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/docker/cli/cli/command/inspect"
	dt "github.com/docker/docker/api/types"
	dtf "github.com/docker/docker/api/types/filters"
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
