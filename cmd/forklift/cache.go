package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift/cache"
	"github.com/PlanktoScope/forklift/internal/app/forklift/env"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

// ls-repo

func cacheLsRepoAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Printf("The cache is empty.")
		return nil
	}
	repos, err := cache.ListRepos(workspace.CacheFS(wpath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	for _, repo := range repos {
		fmt.Printf(
			"%s locked at %s\n", repo.Config.Path, repo.Commit,
		)
	}
	return nil
}

// up

func downloadRepo(palletsPath string, repo env.Repo) (downloaded bool, err error) {
	if !repo.Lock.IsCommitLocked() {
		return false, errors.Errorf("pallet repository %s isn't locked at a commit!", repo.Path())
	}
	vcsRepoVersion, err := repo.VCSRepoVersion()
	if err != nil {
		return false, errors.Wrapf(
			err, "couldn't determine version-locked github repo path for %s", repo.VCSRepoPath,
		)
	}
	path := filepath.Join(palletsPath, vcsRepoVersion)
	if workspace.Exists(path) {
		// TODO: perform a disk checksum
		return false, nil
	}

	fmt.Printf("Downloading %s...\n", vcsRepoVersion)
	gitRepo, err := git.Clone(repo.VCSRepoPath, path)
	if err != nil {
		return false, errors.Wrapf(
			err, "couldn't clone repo %s to %s", repo.VCSRepoPath, path,
		)
	}
	commitTime, err := git.GetCommitTime(gitRepo, repo.Lock.Commit)
	if err != nil {
		if cerr := os.RemoveAll(filepath.Join(path)); cerr != nil {
			fmt.Printf("Error: couldn't delete %s! You will need to delete it yourself.\n", path)
		}
		return false, errors.Wrapf(err, "couldn't check time of commit %s", repo.Lock.Commit)
	}
	commitTimestamp := env.ToTimestamp(commitTime)
	if commitTimestamp != repo.Lock.Timestamp {
		if cerr := os.RemoveAll(filepath.Join(path)); cerr != nil {
			fmt.Printf("Error: couldn't delete %s! You will need to delete it yourself.\n", path)
		}
		return false, errors.Errorf(
			"commit %s was made at %s, while the Pallet repository lock file expects it to have been "+
				"made at %s",
			env.ShortCommit(repo.Lock.Commit), commitTimestamp, repo.Lock.Timestamp,
		)
	}
	if _, err = git.Checkout(gitRepo, repo.Lock.Commit); err != nil {
		if cerr := os.RemoveAll(filepath.Join(path)); cerr != nil {
			fmt.Printf("Error: couldn't delete %s! You will need to delete it yourself.\n", path)
		}
		return false, errors.Wrapf(
			err, "couldn't check out commit %s", repo.Lock.Commit,
		)
	}
	if err = os.RemoveAll(filepath.Join(path, ".git")); err != nil {
		return false, errors.Wrap(err, "couldn't detach from git")
	}
	return true, nil
}

func cacheUpdateAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		return errMissingEnv
	}
	fmt.Printf("Downloading Pallet repositories...\n")
	repos, err := env.ListRepos(workspace.LocalEnvFS(wpath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	cachePath := workspace.CachePath(wpath)
	changed := false
	for _, repo := range repos {
		downloaded, err := downloadRepo(cachePath, repo)
		changed = changed || downloaded
		if err != nil {
			return errors.Wrapf(
				err, "couldn't download %s at commit %s", repo.Path(), env.ShortCommit(repo.Lock.Commit),
			)
		}
	}
	if !changed {
		fmt.Printf("Done! No further actions are needed at this time.\n")
		return nil
	}
	fmt.Printf("Done! Next, you'll probably want to run `forklift depl apply`.\n")
	return nil
}

func cacheRmAction(c *cli.Context) error {
	wpath := c.String("workspace")
	fmt.Printf("Removing cache from workspace %s...\n", wpath)
	return errors.Wrap(workspace.RemoveCache(wpath), "couldn't remove cache")
}
