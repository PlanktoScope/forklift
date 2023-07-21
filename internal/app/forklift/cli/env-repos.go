package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/git"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintEnvRepos(indent int, envPath string) error {
	repos, err := forklift.ListVersionedRepos(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	sort.Slice(repos, func(i, j int) bool {
		return forklift.CompareVersionedRepos(repos[i], repos[j]) < 0
	})
	for _, repo := range repos {
		IndentedPrintf(indent, "%s\n", repo.Path())
	}
	return nil
}

func PrintRepoInfo(
	indent int, envPath, cachePath string, replacementRepos map[string]pallets.FSRepo,
	repoPath string,
) error {
	reposFS, err := forklift.VersionedReposFS(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(
			err, "couldn't open directory for Pallet repositories in environment %s", envPath,
		)
	}
	versionedRepo, err := forklift.LoadVersionedRepo(reposFS, repoPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load Pallet repo versioning config %s from environment %s", repoPath, envPath,
		)
	}
	// TODO: maybe the version should be computed and error-handled when the repo is loaded, so that
	// we don't need error-checking for every subsequent access of the version
	version, err := versionedRepo.Config.Version()
	if err != nil {
		return errors.Wrapf(err, "couldn't determine configured version of Pallet repo %s", repoPath)
	}
	printVersionedRepo(indent, versionedRepo)
	fmt.Println()

	var cachedRepo pallets.FSRepo
	replacementRepo, ok := replacementRepos[repoPath]
	if ok {
		cachedRepo = replacementRepo
	} else {
		if cachedRepo, err = forklift.FindCachedRepo(
			os.DirFS(cachePath), repoPath, version,
		); err != nil {
			return errors.Wrapf(
				err,
				"couldn't find Pallet repository %s@%s in cache, please update the local cache of repos",
				repoPath, version,
			)
		}
	}
	if filepath.IsAbs(cachedRepo.FS.Path()) {
		IndentedPrint(indent+1, "External path (replacing cached package): ")
	} else {
		IndentedPrint(indent+1, "Path in cache: ")
	}
	fmt.Println(cachedRepo.FS.Path())
	IndentedPrintf(indent+1, "Description: %s\n", cachedRepo.Config.Repository.Description)
	// TODO: show the README file
	return nil
}

func printVersionedRepo(indent int, repo forklift.VersionedRepo) {
	IndentedPrintf(indent, "Pallet repository: %s\n", repo.Path())
	indent++
	version, _ := repo.Config.Version() // assume that the validity of the version was already checked
	IndentedPrintf(indent, "Locked version: %s\n", version)
	IndentedPrintf(indent, "Provided by Git repository: %s\n", repo.VCSRepoPath)
}

// Download

func DownloadRepos(indent int, envPath, cachePath string) (changed bool, err error) {
	repos, err := forklift.ListVersionedRepos(os.DirFS(envPath))
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	changed = false
	for _, repo := range repos {
		downloaded, err := downloadRepo(indent, cachePath, repo)
		changed = changed || downloaded
		if err != nil {
			return false, errors.Wrapf(
				err, "couldn't download %s at commit %s", repo.Path(), repo.Config.ShortCommit(),
			)
		}
	}
	return changed, nil
}

func downloadRepo(
	indent int, palletsPath string, repo forklift.VersionedRepo,
) (downloaded bool, err error) {
	if !repo.Config.IsCommitLocked() {
		return false, errors.Errorf(
			"the local environment's versioning config for repository %s has no commit lock", repo.Path(),
		)
	}
	vcsRepoPath := repo.VCSRepoPath
	version, err := repo.Config.Version()
	if err != nil {
		return false, errors.Wrapf(err, "couldn't determine version for %s", vcsRepoPath)
	}
	path := filepath.Join(palletsPath, fmt.Sprintf("%s@%s", repo.VCSRepoPath, version))
	if workspace.Exists(path) {
		// TODO: perform a disk checksum
		return false, nil
	}

	IndentedPrintf(indent, "Downloading %s@%s...\n", repo.VCSRepoPath, version)
	gitRepo, err := git.Clone(vcsRepoPath, path)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't clone repo %s to %s", vcsRepoPath, path)
	}

	// Validate commit
	shortCommit := repo.Config.ShortCommit()
	if err = validateCommit(repo, gitRepo); err != nil {
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
	if err = gitRepo.Checkout(repo.Config.Commit); err != nil {
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

func validateCommit(versionedRepo forklift.VersionedRepo, gitRepo *git.Repo) error {
	// Check commit time
	commitTimestamp, err := forklift.GetCommitTimestamp(gitRepo, versionedRepo.Config.Commit)
	if err != nil {
		return err
	}
	versionedTimestamp := versionedRepo.Config.Timestamp
	if commitTimestamp != versionedTimestamp {
		return errors.Errorf(
			"commit %s was made at %s, while the repository versioning config file expects it to have "+
				"been made at %s",
			versionedRepo.Config.ShortCommit(), commitTimestamp, versionedTimestamp,
		)
	}

	// TODO: implement remaining checks specified in https://go.dev/ref/mod#pseudo-versions
	// (if base version is specified, there must be a corresponding semantic version tag that is an
	// ancestor of the revision described by the pseudo-version; and the revision must be an ancestor
	// of one of the module repository's branches or tags)
	return nil
}
