package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/git"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintEnvRepos(indent int, env *forklift.FSEnv) error {
	repos, err := env.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	sort.Slice(repos, func(i, j int) bool {
		return forklift.CompareRepoReqs(repos[i].RepoReq, repos[j].RepoReq) < 0
	})
	for _, repo := range repos {
		IndentedPrintf(indent, "%s\n", repo.Path())
	}
	return nil
}

func PrintRepoInfo(
	indent int, env *forklift.FSEnv, cache forklift.PathedCache, repoPath string,
) error {
	versionedRepo, err := env.LoadFSRepoReq(repoPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load Pallet repo versioning config %s from environment %s",
			repoPath, env.FS.Path(),
		)
	}
	// TODO: maybe the version should be computed and error-handled when the repo is loaded, so that
	// we don't need error-checking for every subsequent access of the version
	printRepoReq(indent, versionedRepo.RepoReq)
	fmt.Println()

	version := versionedRepo.VersionLock.Version
	cachedRepo, err := cache.LoadFSRepo(repoPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find Pallet repository %s@%s in cache, please update the local cache of repos",
			repoPath, version,
		)
	}
	if pallets.CoversPath(cache, cachedRepo.FS.Path()) {
		IndentedPrintf(
			indent+1, "Path in cache: %s\n", pallets.GetSubdirPath(cache, cachedRepo.FS.Path()),
		)
	} else {
		IndentedPrintf(indent+1, "External path (replacing cached repo): %s\n", cachedRepo.FS.Path())
	}
	IndentedPrintf(indent+1, "Description: %s\n", cachedRepo.Config.Repository.Description)
	// TODO: show the README file
	return nil
}

func printRepoReq(indent int, req forklift.RepoReq) {
	IndentedPrintf(indent, "Pallet repository: %s\n", req.Path())
	indent++
	IndentedPrintf(indent, "Locked version: %s\n", req.VersionLock.Version)
	IndentedPrintf(indent, "Provided by Git repository: %s\n", req.VCSRepoPath)
}

// Download

func DownloadRepos(
	indent int, env *forklift.FSEnv, cache forklift.PathedCache,
) (changed bool, err error) {
	repos, err := env.LoadFSRepoReqs("**")
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	changed = false
	for _, repo := range repos {
		downloaded, err := downloadRepo(indent, cache.Path(), repo.RepoReq)
		changed = changed || downloaded
		if err != nil {
			return false, errors.Wrapf(
				err, "couldn't download %s at commit %s",
				repo.Path(), repo.VersionLock.Config.ShortCommit(),
			)
		}
	}
	return changed, nil
}

func downloadRepo(
	indent int, cachePath string, repo forklift.RepoReq,
) (downloaded bool, err error) {
	if !repo.VersionLock.Config.IsCommitLocked() {
		return false, errors.Errorf(
			"the local environment's versioning config for repository %s has no commit lock", repo.Path(),
		)
	}
	vcsRepoPath := repo.VCSRepoPath
	version := repo.VersionLock.Version
	path := filepath.Join(cachePath, fmt.Sprintf("%s@%s", repo.VCSRepoPath, version))
	if forklift.Exists(path) {
		// TODO: perform a disk checksum
		return false, nil
	}

	IndentedPrintf(indent, "Downloading %s@%s...\n", repo.VCSRepoPath, version)
	gitRepo, err := git.Clone(vcsRepoPath, path)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't clone repo %s to %s", vcsRepoPath, path)
	}

	// Validate commit
	shortCommit := repo.VersionLock.Config.ShortCommit()
	if err = validateCommit(repo, gitRepo); err != nil {
		// TODO: this should instead be a Clear method on a WritableFS at that path
		if cerr := os.RemoveAll(path); cerr != nil {
			IndentedPrintf(
				indent,
				"Error: couldn't clean up %s after failed validation! You'll need to delete it yourself.\n",
				path,
			)
		}
		return false, errors.Wrapf(
			err, "commit %s for github repo %s failed repo version validation", shortCommit, vcsRepoPath,
		)
	}

	// Checkout commit
	if err = gitRepo.Checkout(repo.VersionLock.Config.Commit); err != nil {
		if cerr := os.RemoveAll(path); cerr != nil {
			IndentedPrintf(
				indent, "Error: couldn't clean up %s! You will need to delete it yourself.\n", path,
			)
		}
		return false, errors.Wrapf(err, "couldn't check out commit %s", shortCommit)
	}
	if err = os.RemoveAll(filepath.Join(path, ".git")); err != nil {
		return false, errors.Wrap(err, "couldn't detach from git")
	}
	return true, nil
}

func validateCommit(versionedRepo forklift.RepoReq, gitRepo *git.Repo) error {
	// Check commit time
	commitTimestamp, err := forklift.GetCommitTimestamp(
		gitRepo, versionedRepo.VersionLock.Config.Commit,
	)
	if err != nil {
		return err
	}
	versionedTimestamp := versionedRepo.VersionLock.Config.Timestamp
	if commitTimestamp != versionedTimestamp {
		return errors.Errorf(
			"commit %s was made at %s, while the repository versioning config file expects it to have "+
				"been made at %s",
			versionedRepo.VersionLock.Config.ShortCommit(), commitTimestamp, versionedTimestamp,
		)
	}

	// TODO: implement remaining checks specified in https://go.dev/ref/mod#pseudo-versions
	// (if base version is specified, there must be a corresponding semantic version tag that is an
	// ancestor of the revision described by the pseudo-version; and the revision must be an ancestor
	// of one of the module repository's branches or tags)
	return nil
}
