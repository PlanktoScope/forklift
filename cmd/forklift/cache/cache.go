package cache

import (
	"context"
	"fmt"
	"sort"

	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
)

var errMissingCache = errors.Errorf(
	"you first need to cache the repos specified by your environment with " +
		"`forklift env cache-repo`",
)

func getCache(wpath string) (*forklift.FSRepoCache, error) {
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetRepoCache()
	if err != nil {
		return nil, err
	}
	return cache, nil
}

// rm

func rmAction(c *cli.Context) error {
	cache, err := getCache(c.String("workspace"))
	if err != nil {
		return err
	}

	fmt.Println("Clearing cache...")
	if err = cache.Remove(); err != nil {
		return errors.Wrap(err, "couldn't clear cache")
	}

	fmt.Println("Removing unused Docker container images...")
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}
	report, err := client.PruneUnusedImages(context.Background())
	if err != nil {
		return errors.Wrap(err, "couldn't prune unused Docker container images")
	}
	sort.Slice(report.ImagesDeleted, func(i, j int) bool {
		return docker.CompareDeletedImages(report.ImagesDeleted[i], report.ImagesDeleted[j]) < 0
	})
	for _, deleted := range report.ImagesDeleted {
		if deleted.Untagged != "" {
			fmt.Printf("Untagged %s\n", deleted.Untagged)
		}
		if deleted.Deleted != "" {
			fmt.Printf("Deleted %s\n", deleted.Deleted)
		}
	}
	fmt.Printf("Total reclaimed space: %s\n", units.HumanSize(float64(report.SpaceReclaimed)))
	return nil
}
