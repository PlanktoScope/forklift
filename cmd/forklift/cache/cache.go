package cache

import (
	"context"
	"fmt"
	"os"
	"sort"

	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/internal/clients/docker"
)

var errMissingCache = errors.New(
	"you first need to cache the pallets specified by your pallet with " +
		"`forklift plt cache-pallet`",
)

func getMirrorCache(wpath string, ensureWorkspace bool) (*forklift.FSMirrorCache, error) {
	if ensureWorkspace {
		if !forklift.DirExists(wpath) {
			fmt.Fprintf(os.Stderr, "Making a new workspace at %s...", wpath)
		}
		if err := forklift.EnsureExists(wpath); err != nil {
			return nil, errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
		}
	}
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetMirrorCache()
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func getPalletCache(wpath string, ensureWorkspace bool) (*forklift.FSPalletCache, error) {
	if ensureWorkspace {
		if !forklift.DirExists(wpath) {
			fmt.Fprintf(os.Stderr, "Making a new workspace at %s...", wpath)
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

// del-all

func delAllAction(c *cli.Context) error {
	if err := delGitRepoAction("mirror", getMirrorCache)(c); err != nil {
		return errors.Wrap(err, "couldn't remove cached mirrors")
	}
	if err := delGitRepoAction("pallet", getPalletCache)(c); err != nil {
		return errors.Wrap(err, "couldn't remove cached pallets")
	}

	if err := delImgAction(c); err != nil {
		return errors.Wrap(err, "couldn't remove unused Docker container images")
	}
	return nil
}

// del-img

func delImgAction(_ *cli.Context) error {
	fmt.Fprintln(os.Stderr, "Removing unused Docker container images...")
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
			fmt.Fprintf(os.Stderr, "Untagged %s\n", deleted.Untagged)
		}
		if deleted.Deleted != "" {
			fmt.Fprintf(os.Stderr, "Deleted %s\n", deleted.Deleted)
		}
	}
	fmt.Fprintf(
		os.Stderr, "Total reclaimed space: %s\n", units.HumanSize(float64(report.SpaceReclaimed)),
	)
	return nil
}
