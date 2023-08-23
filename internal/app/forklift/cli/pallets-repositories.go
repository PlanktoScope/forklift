package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/git"
	"github.com/PlanktoScope/forklift/pkg/core"
)

// Print

func PrintPalletRepos(indent int, pallet *forklift.FSPallet) error {
	loadedRepos, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify repos")
	}
	sort.Slice(loadedRepos, func(i, j int) bool {
		return forklift.CompareRepoReqs(loadedRepos[i].RepoReq, loadedRepos[j].RepoReq) < 0
	})
	for _, repo := range loadedRepos {
		IndentedPrintf(indent, "%s\n", repo.Path())
	}
	return nil
}

func PrintRepoInfo(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedRepoCache, repoPath string,
) error {
	req, err := pallet.LoadFSRepoReq(repoPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load repo version lock definition %s from pallet %s",
			repoPath, pallet.FS.Path(),
		)
	}
	printRepoReq(indent, req.RepoReq)
	fmt.Println()
	indent++

	version := req.VersionLock.Version
	cachedRepo, err := cache.LoadFSRepo(repoPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find repo %s@%s in cache, please update the local cache of repos",
			repoPath, version,
		)
	}
	if core.CoversPath(cache, cachedRepo.FS.Path()) {
		IndentedPrintf(
			indent, "Path in cache: %s\n", core.GetSubdirPath(cache, cachedRepo.FS.Path()),
		)
	} else {
		IndentedPrintf(indent, "External path (replacing cached repo): %s\n", cachedRepo.FS.Path())
	}
	IndentedPrintf(indent, "Description: %s\n", cachedRepo.Def.Repo.Description)

	readme, err := cachedRepo.LoadReadme()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load readme file for repo %s@%s from cache", repoPath, version,
		)
	}
	IndentedPrintln(indent, "Readme:")
	const widthLimit = 100
	PrintReadme(indent+1, readme, widthLimit)
	return nil
}

func printRepoReq(indent int, req forklift.RepoReq) {
	IndentedPrintf(indent, "Repo: %s\n", req.Path())
	indent++
	IndentedPrintf(indent, "Locked version: %s\n", req.VersionLock.Version)
}

func PrintReadme(indent int, readme []byte, widthLimit int) {
	lines := strings.Split(string(readme), "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			IndentedPrintln(indent)
		}
		for len(line) > 0 {
			if len(line) < widthLimit { // we've printed everything!
				IndentedPrintln(indent, line)
				break
			}
			IndentedPrintln(indent, line[:widthLimit])
			line = line[widthLimit:]
		}
	}
}

// Download

func DownloadRepos(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedRepoCache,
) (changed bool, err error) {
	loadedRepos, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify repos")
	}
	changed = false
	for _, repo := range loadedRepos {
		downloaded, err := downloadRepo(indent, cache.Path(), repo.RepoReq)
		changed = changed || downloaded
		if err != nil {
			return false, errors.Wrapf(
				err, "couldn't download %s at commit %s", repo.Path(), repo.VersionLock.Def.ShortCommit(),
			)
		}
	}
	return changed, nil
}

func downloadRepo(
	indent int, cachePath string, repo forklift.RepoReq,
) (downloaded bool, err error) {
	if !repo.VersionLock.Def.IsCommitLocked() {
		return false, errors.Errorf(
			"the pallet's version lock definition for repo %s has no commit lock", repo.Path(),
		)
	}
	repoPath := repo.Path()
	version := repo.VersionLock.Version
	repoCachePath := filepath.Join(
		filepath.FromSlash(cachePath), fmt.Sprintf("%s@%s", filepath.FromSlash(repoPath), version),
	)
	if forklift.Exists(repoCachePath) {
		// TODO: perform a disk checksum
		return false, nil
	}

	IndentedPrintf(indent, "Downloading %s@%s...\n", repoPath, version)
	gitRepo, err := git.Clone(repoPath, repoCachePath)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't clone git repo %s to %s", repoPath, repoCachePath)
	}

	// Validate commit
	shortCommit := repo.VersionLock.Def.ShortCommit()
	if err = validateCommit(repo, gitRepo); err != nil {
		// TODO: this should instead be a Clear method on a WritableFS at that path
		if cerr := os.RemoveAll(repoCachePath); cerr != nil {
			IndentedPrintf(
				indent,
				"Error: couldn't clean up %s after failed validation! You'll need to delete it yourself.\n",
				repoCachePath,
			)
		}
		return false, errors.Wrapf(
			err, "commit %s for repo %s failed version validation", shortCommit, repoPath,
		)
	}

	// Checkout commit
	if err = gitRepo.Checkout(repo.VersionLock.Def.Commit); err != nil {
		if cerr := os.RemoveAll(repoCachePath); cerr != nil {
			IndentedPrintf(
				indent, "Error: couldn't clean up %s! You'll need to delete it yourself.\n", repoCachePath,
			)
		}
		return false, errors.Wrapf(err, "couldn't check out commit %s", shortCommit)
	}
	if err = os.RemoveAll(filepath.Join(repoCachePath, ".git")); err != nil {
		return false, errors.Wrap(err, "couldn't detach from git")
	}
	return true, nil
}

func validateCommit(req forklift.RepoReq, gitRepo *git.Repo) error {
	// Check commit time
	commitTimestamp, err := forklift.GetCommitTimestamp(gitRepo, req.VersionLock.Def.Commit)
	if err != nil {
		return err
	}
	versionedTimestamp := req.VersionLock.Def.Timestamp
	if commitTimestamp != versionedTimestamp {
		return errors.Errorf(
			"commit %s was made at %s, while the repo version lock definition expects it to have "+
				"been made at %s",
			req.VersionLock.Def.ShortCommit(), commitTimestamp, versionedTimestamp,
		)
	}

	// TODO: implement remaining checks specified in https://go.dev/ref/mod#pseudo-versions
	// (if base version is specified, there must be a corresponding semantic version tag that is an
	// ancestor of the revision described by the pseudo-version; and the revision must be an ancestor
	// of one of the module repository's branches or tags)
	return nil
}
