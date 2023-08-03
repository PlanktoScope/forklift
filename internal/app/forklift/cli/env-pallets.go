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
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintEnvPallets(indent int, env *forklift.FSEnv) error {
	loadedPallets, err := env.LoadFSPalletReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify pallets")
	}
	sort.Slice(loadedPallets, func(i, j int) bool {
		return forklift.ComparePalletReqs(loadedPallets[i].PalletReq, loadedPallets[j].PalletReq) < 0
	})
	for _, pallet := range loadedPallets {
		IndentedPrintf(indent, "%s\n", pallet.Path())
	}
	return nil
}

func PrintPalletInfo(
	indent int, env *forklift.FSEnv, cache forklift.PathedPalletCache, palletPath string,
) error {
	req, err := env.LoadFSPalletReq(palletPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load pallet version lock definition %s from environment %s",
			palletPath, env.FS.Path(),
		)
	}
	printPalletReq(indent, req.PalletReq)
	fmt.Println()
	indent++

	version := req.VersionLock.Version
	cachedPallet, err := cache.LoadFSPallet(palletPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find pallet %s@%s in cache, please update the local cache of pallets",
			palletPath, version,
		)
	}
	if pallets.CoversPath(cache, cachedPallet.FS.Path()) {
		IndentedPrintf(
			indent, "Path in cache: %s\n", pallets.GetSubdirPath(cache, cachedPallet.FS.Path()),
		)
	} else {
		IndentedPrintf(indent, "External path (replacing cached pallet): %s\n", cachedPallet.FS.Path())
	}
	IndentedPrintf(indent, "Description: %s\n", cachedPallet.Def.Pallet.Description)

	readme, err := cachedPallet.LoadReadme()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load readme file for pallet %s@%s from cache", palletPath, version,
		)
	}
	IndentedPrintln(indent, "Readme:")
	const widthLimit = 100
	PrintReadme(indent+1, readme, widthLimit)
	return nil
}

func printPalletReq(indent int, req forklift.PalletReq) {
	IndentedPrintf(indent, "Pallet: %s\n", req.Path())
	indent++
	IndentedPrintf(indent, "Locked version: %s\n", req.VersionLock.Version)
	IndentedPrintf(indent, "Provided by Git repository: %s\n", req.VCSRepoPath)
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

func DownloadPallets(
	indent int, env *forklift.FSEnv, cache forklift.PathedPalletCache,
) (changed bool, err error) {
	loadedPallets, err := env.LoadFSPalletReqs("**")
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify pallets")
	}
	changed = false
	for _, pallet := range loadedPallets {
		downloaded, err := downloadPallet(indent, cache.Path(), pallet.PalletReq)
		changed = changed || downloaded
		if err != nil {
			return false, errors.Wrapf(
				err, "couldn't download %s at commit %s",
				pallet.Path(), pallet.VersionLock.Def.ShortCommit(),
			)
		}
	}
	return changed, nil
}

func downloadPallet(
	indent int, cachePath string, pallet forklift.PalletReq,
) (downloaded bool, err error) {
	if !pallet.VersionLock.Def.IsCommitLocked() {
		return false, errors.Errorf(
			"the local environment's version lock definition for pallet %s has no commit lock",
			pallet.Path(),
		)
	}
	vcsRepoPath := pallet.VCSRepoPath
	version := pallet.VersionLock.Version
	palletCachePath := filepath.Join(
		filepath.FromSlash(cachePath), fmt.Sprintf("%s@%s", pallet.VCSRepoPath, version),
	)
	if forklift.Exists(palletCachePath) {
		// TODO: perform a disk checksum
		return false, nil
	}

	IndentedPrintf(indent, "Downloading %s@%s...\n", pallet.VCSRepoPath, version)
	gitRepo, err := git.Clone(vcsRepoPath, palletCachePath)
	if err != nil {
		return false, errors.Wrapf(
			err, "couldn't clone git repo %s to %s", vcsRepoPath, palletCachePath,
		)
	}

	// Validate commit
	shortCommit := pallet.VersionLock.Def.ShortCommit()
	if err = validateCommit(pallet, gitRepo); err != nil {
		// TODO: this should instead be a Clear method on a WritableFS at that path
		if cerr := os.RemoveAll(palletCachePath); cerr != nil {
			IndentedPrintf(
				indent,
				"Error: couldn't clean up %s after failed validation! You'll need to delete it yourself.\n",
				palletCachePath,
			)
		}
		return false, errors.Wrapf(
			err, "commit %s for github repo %s failed pallet version validation",
			shortCommit, vcsRepoPath,
		)
	}

	// Checkout commit
	if err = gitRepo.Checkout(pallet.VersionLock.Def.Commit); err != nil {
		if cerr := os.RemoveAll(palletCachePath); cerr != nil {
			IndentedPrintf(
				indent, "Error: couldn't clean up %s! You will need to delete it yourself.\n",
				palletCachePath,
			)
		}
		return false, errors.Wrapf(err, "couldn't check out commit %s", shortCommit)
	}
	if err = os.RemoveAll(filepath.Join(palletCachePath, ".git")); err != nil {
		return false, errors.Wrap(err, "couldn't detach from git")
	}
	return true, nil
}

func validateCommit(req forklift.PalletReq, gitRepo *git.Repo) error {
	// Check commit time
	commitTimestamp, err := forklift.GetCommitTimestamp(gitRepo, req.VersionLock.Def.Commit)
	if err != nil {
		return err
	}
	versionedTimestamp := req.VersionLock.Def.Timestamp
	if commitTimestamp != versionedTimestamp {
		return errors.Errorf(
			"commit %s was made at %s, while the pallet version lock definition expects it to have "+
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
