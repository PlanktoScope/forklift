package cli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
)

func GetRepoCache(
	wpath string, pallet *forklift.FSPallet, ensureCache bool,
) (*forklift.LayeredRepoCache, *forklift.RepoOverrideCache, error) {
	cache := &forklift.LayeredRepoCache{}
	override, err := makeRepoOverrideCacheFromPallet(pallet)
	if err != nil {
		return nil, nil, err
	}
	cache.Overlay = override

	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, nil, err
	}
	fsCache, err := workspace.GetRepoCache()
	if err != nil && override == nil {
		return nil, nil, err
	}
	cache.Underlay = fsCache

	if ensureCache && !fsCache.Exists() {
		repoReqs, err := pallet.LoadFSRepoReqs("**")
		if err != nil {
			return nil, nil, errors.Wrap(err, "couldn't check whether the pallet requires any repos")
		}
		if len(repoReqs) > 0 {
			return nil, nil, errors.New("you first need to cache the repos specified by your pallet")
		}
	}
	return cache, override, nil
}

func makeRepoOverrideCacheFromPallet(
	pallet *forklift.FSPallet,
) (*forklift.RepoOverrideCache, error) {
	palletAsRepo, err := core.LoadFSRepo(pallet.FS, ".")
	if err != nil {
		// The common case is that the pallet is not a repo (and thus can't be loaded as one), so we
		// mask the error:
		return nil, nil
	}
	return forklift.NewRepoOverrideCache(
		[]*core.FSRepo{palletAsRepo}, map[string][]string{
			// In a pallet which is a repo, the implicit repo requirement is for an empty version string
			palletAsRepo.Path(): {""},
		},
	)
}

// Print

func PrintPalletRepos(indent int, pallet *forklift.FSPallet) error {
	loadedRepos, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify repos")
	}
	sort.Slice(loadedRepos, func(i, j int) bool {
		return forklift.CompareGitRepoReqs(
			loadedRepos[i].RepoReq.GitRepoReq, loadedRepos[j].RepoReq.GitRepoReq,
		) < 0
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
	indent++

	version := req.VersionLock.Version
	cachedRepo, err := cache.LoadFSRepo(repoPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find repo %s@%s in cache, please update the local cache of repos",
			repoPath, version,
		)
	}
	IndentedPrintf(indent, "Forklift version: %s\n", cachedRepo.Def.ForkliftVersion)
	fmt.Println()

	if core.CoversPath(cache, cachedRepo.FS.Path()) {
		IndentedPrintf(
			indent, "Path in cache: %s\n", core.GetSubdirPath(cache, cachedRepo.FS.Path()),
		)
	} else {
		IndentedPrintf(indent, "Absolute path (replacing any cached copy): %s\n", cachedRepo.FS.Path())
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
	IndentedPrintf(indent, "Locked repo version: %s\n", req.VersionLock.Version)
}

// Add

func AddRepoRequirements(
	indent int, pallet *forklift.FSPallet, cachePath string, repoQueries []string,
) error {
	if err := validateGitRepoQueries(repoQueries); err != nil {
		return errors.Wrap(err, "one or more repo queries is invalid")
	}
	resolved, err := ResolveQueriesUsingLocalMirrors(0, cachePath, repoQueries, true)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Printf("Saving configurations to %s...\n", pallet.FS.Path())
	for _, repoQuery := range repoQueries {
		req, ok := resolved[repoQuery]
		if !ok {
			return errors.Errorf("couldn't find configuration for %s", repoQuery)
		}
		reqsReposFS, err := pallet.GetRepoReqsFS()
		if err != nil {
			return err
		}
		repoReqPath := path.Join(reqsReposFS.Path(), req.Path(), forklift.VersionLockDefFile)
		marshaled, err := yaml.Marshal(req.VersionLock.Def)
		if err != nil {
			return errors.Wrapf(err, "couldn't marshal repo requirement for %s", repoReqPath)
		}
		if err := forklift.EnsureExists(filepath.FromSlash(path.Dir(repoReqPath))); err != nil {
			return errors.Wrapf(
				err, "couldn't make directory %s", filepath.FromSlash(path.Dir(repoReqPath)),
			)
		}
		const perm = 0o644 // owner rw, group r, public r
		if err := os.WriteFile(filepath.FromSlash(repoReqPath), marshaled, perm); err != nil {
			return errors.Wrapf(
				err, "couldn't save repo requirement to %s", filepath.FromSlash(repoReqPath),
			)
		}
	}
	return nil
}

// Remove

func RemoveRepoRequirements(
	indent int, pallet *forklift.FSPallet, repoPaths []string, force bool,
) error {
	usedRepoReqs, err := determineUsedRepoReqs(indent, pallet, force)
	if err != nil {
		return errors.Wrap(
			err,
			"couldn't determine repos used by declared package deployments, to check which repositories "+
				"to remove are still required by declared deployments; to skip this check, "+
				"enable the --force flag",
		)
	}

	fmt.Printf("Removing requirements from %s...\n", pallet.FS.Path())
	for _, repoPath := range repoPaths {
		if actualRepoPath, _, ok := strings.Cut(repoPath, "@"); ok {
			IndentedPrintf(
				indent,
				"Warning: provided repo path %s is actually a repo query; removing %s instead...\n",
				repoPath, actualRepoPath,
			)
			repoPath = actualRepoPath
		}
		reqsReposFS, err := pallet.GetRepoReqsFS()
		if err != nil {
			return err
		}
		repoReqPath := path.Join(reqsReposFS.Path(), repoPath)
		if !force && len(usedRepoReqs[repoPath]) > 0 {
			return errors.Errorf(
				"couldn't remove requirement for repo %s because it's needed by package deployments %+v; "+
					"to skip this check, enable the --force flag",
				repoPath, usedRepoReqs[repoPath],
			)
		}
		if err = os.RemoveAll(repoReqPath); err != nil {
			return errors.Wrapf(
				err, "couldn't remove requirement for repo %s, at %s", repoPath, repoReqPath,
			)
		}
	}
	// TODO: maybe it'd be better to remove everything we can remove and then report errors at the
	// end?
	return nil
}

func determineUsedRepoReqs(
	indent int, pallet *forklift.FSPallet, force bool,
) (map[string][]string, error) {
	depls, err := pallet.LoadDepls("**/*")
	if err != nil {
		err = errors.Wrap(err, "couldn't load package deployments")
		if !force {
			return nil, err
		}
		IndentedPrintf(indent, "Warning: %s\n", err.Error())
	}
	usedRepoReqs := make(map[string][]string)
	for _, depl := range depls {
		pkgPath := depl.Def.Package
		if path.IsAbs(pkgPath) { // special case: package is provided by the pallet itself
			continue
		}
		repoReqsFS, err := pallet.GetRepoReqsFS()
		if err != nil {
			return nil, errors.Wrap(err, "couldn't open directory for repo requirements from pallet")
		}
		fsRepoReq, err := forklift.LoadFSRepoReqContaining(repoReqsFS, pkgPath)
		if err != nil {
			err = errors.Wrapf(
				err, "couldn't find repo requirement needed for deployment %s of package %s",
				depl.Name, pkgPath,
			)
			if !force {
				return nil, err
			}
			IndentedPrintf(indent, "Warning: %s\n", err.Error())
		}
		usedRepoReqs[fsRepoReq.Path()] = append(usedRepoReqs[fsRepoReq.Path()], depl.Name)
	}
	return usedRepoReqs, nil
}

// Download

func DownloadRequiredRepos(
	indent int, pallet *forklift.FSPallet, cachePath string,
) (changed bool, err error) {
	loadedRepoReqs, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify repos")
	}
	changed = false
	for _, req := range loadedRepoReqs {
		downloaded, err := DownloadLockedGitRepoUsingLocalMirror(
			indent, cachePath, req.Path(), req.VersionLock,
		)
		changed = changed || downloaded
		if err != nil {
			return false, errors.Wrapf(
				err, "couldn't download %s at commit %s", req.Path(), req.VersionLock.Def.ShortCommit(),
			)
		}
	}
	return changed, nil
}
