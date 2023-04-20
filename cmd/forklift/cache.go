package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
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
	repos, err := forklift.ListCachedRepos(workspace.CacheFS(wpath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	for _, repo := range repos {
		fmt.Printf("%s@%s\n", repo.Config.Path, repo.Version)
	}
	return nil
}

// up

func validateCommit(versionedRepo forklift.VersionedRepo, gitRepo *git.Repo) error {
	// Check commit time
	commitTime, err := gitRepo.GetCommitTime(versionedRepo.Lock.Commit)
	if err != nil {
		return errors.Wrapf(err, "couldn't check time of commit %s", versionedRepo.Lock.ShortCommit())
	}
	commitTimestamp := forklift.ToTimestamp(commitTime)
	versionedTimestamp := versionedRepo.Lock.Timestamp
	if commitTimestamp != versionedTimestamp {
		return errors.Errorf(
			"commit %s was made at %s, while the Pallet repository lock file expects it to have been "+
				"made at %s",
			versionedRepo.Lock.ShortCommit(), commitTimestamp, versionedTimestamp,
		)
	}

	// TODO: implement remaining checks specified in https://go.dev/ref/mod#pseudo-versions
	// (if base version is specified, there must be a corresponding semantic version tag that is an
	// ancestor of the revision described by the pseudo-version; and the revision must be an ancestor
	// of one of the module repository's branches or tags)
	return nil
}

func downloadRepo(palletsPath string, repo forklift.VersionedRepo) (downloaded bool, err error) {
	if !repo.Lock.IsCommitLocked() {
		return false, errors.Errorf(
			"the local environment's version lock for repository %s has no commit lock", repo.Path(),
		)
	}
	vcsRepoPath := repo.VCSRepoPath
	vcsRepoVersion, err := repo.VCSRepoVersion()
	if err != nil {
		return false, errors.Wrapf(
			err, "couldn't determine version-locked github repo path for %s", vcsRepoPath,
		)
	}
	path := filepath.Join(palletsPath, vcsRepoVersion)
	if workspace.Exists(path) {
		// TODO: perform a disk checksum
		return false, nil
	}

	fmt.Printf("Downloading %s...\n", vcsRepoVersion)
	gitRepo, err := git.Clone(vcsRepoPath, path)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't clone repo %s to %s", vcsRepoPath, path)
	}

	// Validate commit
	shortCommit := repo.Lock.ShortCommit()
	if err = validateCommit(repo, gitRepo); err != nil {
		if cerr := os.RemoveAll(path); cerr != nil {
			fmt.Printf(
				"Error: couldn't clean up %s after failed validation! You'll need to delete it yourself.\n",
				path,
			)
		}
		return false, errors.Wrapf(
			err, "commit %s for github repo %s failed repo lock validation", shortCommit, vcsRepoPath,
		)
	}

	// Checkout commit
	if err = gitRepo.Checkout(repo.Lock.Commit); err != nil {
		if cerr := os.RemoveAll(path); cerr != nil {
			fmt.Printf("Error: couldn't clean up %s! You will need to delete it yourself.\n", path)
		}
		return false, errors.Wrapf(err, "couldn't check out commit %s", shortCommit)
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
	repos, err := forklift.ListVersionedRepos(workspace.LocalEnvFS(wpath))
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
				err, "couldn't download %s at commit %s", repo.Path(), repo.Lock.ShortCommit(),
			)
		}
	}
	if !changed {
		fmt.Printf("Done! No further actions are needed at this time.\n")
		return nil
	}

	// TODO: download all Docker images used by packages in the repo - either by inspecting the
	// Docker stack definitions or by allowing packages to list Docker images used.
	fmt.Printf("Done! Next, you'll probably want to run `forklift depl apply`.\n")
	return nil
}

func cacheRmAction(c *cli.Context) error {
	wpath := c.String("workspace")
	fmt.Printf("Removing cache from workspace %s...\n", wpath)
	return errors.Wrap(workspace.RemoveCache(wpath), "couldn't remove cache")
}
