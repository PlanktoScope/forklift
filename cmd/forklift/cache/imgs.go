package cache

import (
	"context"
	"fmt"
	"sort"

	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
)

// ls-img

func lsImgAction(c *cli.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	imgs, err := client.ListImages(context.Background(), c.Args().First())
	if err != nil {
		return errors.Wrapf(err, "couldn't list local Docker images")
	}
	sort.Slice(imgs, func(i, j int) bool {
		return imgs[i].Repository < imgs[j].Repository
	})
	for _, img := range imgs {
		fmt.Printf("%s: %s", img.ID, img.Repository)
		if img.Tag != "" {
			fmt.Printf(":%s", img.Tag)
		}
		fmt.Println()
	}
	return nil
}

// show-img

func showImgAction(c *cli.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	imageHash := c.Args().First()
	image, err := client.InspectImage(context.Background(), imageHash)
	if err != nil {
		return errors.Wrapf(err, "couldn't inspect image %s", imageHash)
	}
	printImg(0, image)
	return nil
}

func printImg(indent int, img docker.Image) {
	fcli.IndentedPrintf(indent, "Docker container image: %s\n", img.ID)
	indent++

	fcli.IndentedPrint(indent, "Provided by container image repository: ")
	if img.Repository == "" {
		fmt.Print("(none)")
	} else {
		fmt.Print(img.Repository)
	}
	fmt.Println()

	printImgRepoTags(indent+1, img.Inspect.RepoTags)
	printImgRepoDigests(indent+1, img.Inspect.RepoDigests)

	fcli.IndentedPrintf(indent, "Created: %s\n", img.Inspect.Created)
	fcli.IndentedPrintf(indent, "Size: %s\n", units.HumanSize(float64(img.Inspect.Size)))
}

func printImgRepoTags(indent int, tags []string) {
	fcli.IndentedPrint(indent, "Repo tags:")
	if len(tags) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, tag := range tags {
		fcli.BulletedPrintf(indent, "%s\n", tag)
	}
}

func printImgRepoDigests(indent int, digests []string) {
	fcli.IndentedPrint(indent, "Repo digests:")
	if len(digests) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, digest := range digests {
		fcli.BulletedPrintf(indent, "%s\n", digest)
	}
}
