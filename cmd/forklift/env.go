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

var errMissingEnv = errors.Errorf(
	"you first need to set up a local environment with `forklift env clone`",
)

// clone

func envCloneAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(wpath) {
		fmt.Printf("Making a new workspace at %s...", wpath)
	}
	if err := workspace.EnsureExists(wpath); err != nil {
		return errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
	}
	remoteRelease := c.Args().First()
	remote, release, err := git.ParseRemoteRelease(remoteRelease)
	if err != nil {
		return errors.Wrapf(err, "couldn't parse remote release %s", remoteRelease)
	}
	local := workspace.LocalEnvPath(wpath)
	fmt.Printf("Cloning environment %s to %s...\n", remote, local)
	gitRepo, err := git.Clone(remote, local)
	if err != nil {
		if !errors.Is(err, git.ErrRepositoryAlreadyExists) {
			return errors.Wrapf(
				err, "couldn't clone environment %s at release %s to %s", remote, release, local,
			)
		}
		if !c.Bool("force") {
			return errors.Wrap(
				err,
				"you need to first delete your local environment with `forklift env rm` before "+
					"cloning another remote release to it",
			)
		}
		fmt.Printf(
			"Removing local environment from workspace %s, because it already exists and the "+
				"command's --force flag was enabled...\n",
			wpath,
		)
		if err = workspace.RemoveLocalEnv(wpath); err != nil {
			return errors.Wrap(err, "couldn't remove local environment")
		}
		fmt.Printf("Cloning environment %s to %s...\n", remote, local)
		if gitRepo, err = git.Clone(remote, local); err != nil {
			return errors.Wrapf(
				err, "couldn't clone environment %s at release %s to %s", remote, release, local,
			)
		}
	}
	fmt.Printf("Checking out release %s...\n", release)
	if err = gitRepo.Checkout(release); err != nil {
		return errors.Wrapf(
			err, "couldn't check out release %s at %s", release, local,
		)
	}
	fmt.Println("Done! Next, you'll probably want to run `forklift env cache`.")
	return nil
}

// cache

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

func envCacheAction(c *cli.Context) error {
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

// fetch

func envFetchAction(c *cli.Context) error {
	wpath := c.String("workspace")
	envPath := workspace.LocalEnvPath(wpath)
	if !workspace.Exists(envPath) {
		return errMissingEnv
	}
	fmt.Println("Fetching updates...")
	updated, err := git.Fetch(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't fetch changes from the remote release")
	}
	if !updated {
		fmt.Print("No updates from the remote release.")
	}
	// TODO: display changes
	return nil
}

// pull

func envPullAction(c *cli.Context) error {
	wpath := c.String("workspace")
	envPath := workspace.LocalEnvPath(wpath)
	if !workspace.Exists(envPath) {
		return errMissingEnv
	}
	fmt.Println("Attempting to fast-forward the local environment...")
	updated, err := git.Pull(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't fast-forward the local environment")
	}
	if !updated {
		fmt.Println("No changes from the remote release.")
	}
	// TODO: display changes
	return nil
}

// rm

func envRmAction(c *cli.Context) error {
	wpath := c.String("workspace")
	fmt.Printf("Removing local environment from workspace %s...\n", wpath)
	return errors.Wrap(workspace.RemoveLocalEnv(wpath), "couldn't remove local environment")
}
