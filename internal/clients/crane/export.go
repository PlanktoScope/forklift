// Package crane provides a wrapper around go-containerregistry's functionality
package crane

import (
	"context"
	"io"

	"github.com/containerd/platforms"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
)

type Platform = v1.Platform

func ExportOCIImage(
	ctx context.Context, imageName string, w io.Writer, platform string,
) error {
	ref, err := name.ParseReference(imageName, name.StrictValidation)
	if err != nil {
		return errors.Wrapf(err, "couldn't parse image name: %s", imageName)
	}
	imageName = ref.Name()

	parsedPlatform, err := v1.ParsePlatform(platform)
	if err != nil {
		return errors.Wrapf(err, "couldn't parse platform: %s", platform)
	}
	desc, err := crane.Get(imageName, crane.WithContext(ctx), crane.WithPlatform(parsedPlatform))
	if err != nil {
		return errors.Wrapf(err, "couldn't pull image %s", imageName)
	}
	var image v1.Image
	if desc.MediaType.IsSchema1() {
		if image, err = desc.Schema1(); err != nil {
			return errors.Wrapf(err, "couldn't pull schema 1 image %s", imageName)
		}
	} else {
		if image, err = desc.Image(); err != nil {
			return errors.Wrapf(err, "couldn't pull image %s", imageName)
		}
	}
	return crane.Export(image, w)
}

func DetectPlatform() Platform {
	detectedPlatform := platforms.Normalize(platforms.DefaultSpec())
	return Platform{
		Architecture: detectedPlatform.Architecture,
		OS:           detectedPlatform.OS,
		Variant:      detectedPlatform.Variant,
	}
}
