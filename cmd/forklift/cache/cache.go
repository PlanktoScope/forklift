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
	"you first need to cache the repos specified by your pallet with " +
		"`forklift plt cache-repo`",
)

func getPalletCache(wpath string, ensureWorkspace bool) (*forklift.FSPalletCache, error) {
	if ensureWorkspace {
		if !forklift.Exists(wpath) {
			fmt.Printf("Making a new workspace at %s...", wpath)
		}
		if err := forklift.EnsureExists(wpath); err != nil {
			return nil, errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
		}
	}
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetPalletCache()
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func getRepoCache(wpath string, ensureWorkspace bool) (*forklift.FSRepoCache, error) {
	if ensureWorkspace {
		if !forklift.Exists(wpath) {
			fmt.Printf("Making a new workspace at %s...", wpath)
		}
		if err := forklift.EnsureExists(wpath); err != nil {
			return nil, errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
		}
	}
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

// rm-all

func rmAllAction(c *cli.Context) error {
	if err := rmGitRepoAction("pallet", getPalletCache)(c); err != nil {
		return errors.Wrap(err, "couldn't remove cached pallets")
	}
	if err := rmGitRepoAction("repo", getRepoCache)(c); err != nil {
		return errors.Wrap(err, "couldn't remove cached repositories")
	}

	if err := rmImgAction(c); err != nil {
		return errors.Wrap(err, "couldn't remove unused Docker container images")
	}
	return nil
}

// rm-img

func rmImgAction(_ *cli.Context) error {
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
