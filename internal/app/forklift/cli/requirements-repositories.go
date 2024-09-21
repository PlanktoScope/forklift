package cli

import (
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

func GetRepoCache(
	wpath string, pallet *forklift.FSPallet, requireCache bool,
) (*forklift.LayeredRepoCache, *forklift.RepoOverrideCache, error) {
	cache := &forklift.LayeredRepoCache{}
	override, err := makeRepoOverrideCacheFromPallet(pallet, true)
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

	if requireCache && !fsCache.Exists() {
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
	pallet *forklift.FSPallet, generateRepoFromPallet bool,
) (*forklift.RepoOverrideCache, error) {
	palletAsRepo, err := core.LoadFSRepo(pallet.FS, ".")
	if err != nil {
		if !generateRepoFromPallet {
			return nil, nil
		}
		palletAsRepo = &core.FSRepo{
			Repo: core.Repo{
				Version: pallet.Version,
				Def: core.RepoDef{
					ForkliftVersion: pallet.Def.ForkliftVersion,
					Repo: core.RepoSpec{
						Path:        pallet.Def.Pallet.Path,
						Description: pallet.Def.Pallet.Description,
						ReadmeFile:  pallet.Def.Pallet.ReadmeFile,
					},
				},
			},
			FS: pallet.FS,
		}
	}
	return forklift.NewRepoOverrideCache(
		[]*core.FSRepo{palletAsRepo}, map[string][]string{
			// In a pallet which is a repo, the implicit repo requirement is for an empty version string
			palletAsRepo.Path(): {""},
		},
	)
}

// Printing

func PrintRequiredRepos(indent int, pallet *forklift.FSPallet) error {
	loadedRepos, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify repos")
	}
	slices.SortFunc(loadedRepos, func(a, b *forklift.FSRepoReq) int {
		return forklift.CompareGitRepoReqs(a.RepoReq.GitRepoReq, b.RepoReq.GitRepoReq)
	})
	for _, repo := range loadedRepos {
		IndentedPrintf(indent, "%s\n", repo.Path())
	}
	return nil
}

func PrintRequiredRepoLocation(
	pallet *forklift.FSPallet, cache forklift.PathedRepoCache, requiredRepoPath string,
) error {
	req, err := pallet.LoadFSRepoReq(requiredRepoPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load repo version lock definition %s from pallet %s",
			requiredRepoPath, pallet.FS.Path(),
		)
	}

	version := req.VersionLock.Version
	cachedRepo, err := cache.LoadFSRepo(requiredRepoPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find repo %s@%s in cache, please update the local cache of repos",
			requiredRepoPath, version,
		)
	}
	fmt.Println(cachedRepo.FS.Path())
	return nil
}

func PrintRequiredRepoInfo(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedRepoCache, requiredRepoPath string,
) error {
	req, err := pallet.LoadFSRepoReq(requiredRepoPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load repo version lock definition %s from pallet %s",
			requiredRepoPath, pallet.FS.Path(),
		)
	}
	printRepoReq(indent, req.RepoReq)
	indent++

	version := req.VersionLock.Version
	cachedRepo, err := cache.LoadFSRepo(requiredRepoPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find repo %s@%s in cache, please update the local cache of repos",
			requiredRepoPath, version,
		)
	}
	return PrintCachedRepo(indent, cache, cachedRepo, false)
}

func printRepoReq(indent int, req forklift.RepoReq) {
	IndentedPrintf(indent, "Repo: %s\n", req.Path())
	indent++
	IndentedPrintf(indent, "Locked repo version: %s\n", req.VersionLock.Version)
}

// Add

func AddRepoReqs(
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
		if err = writeVersionLock(req.VersionLock, repoReqPath); err != nil {
			return errors.Wrapf(err, "couldn't write version lock for repo requirement")
		}
	}
	return nil
}

func writeVersionLock(lock forklift.VersionLock, writePath string) error {
	marshaled, err := yaml.Marshal(lock.Def)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal version lock")
	}
	parentDir := filepath.FromSlash(path.Dir(writePath))
	if err := forklift.EnsureExists(parentDir); err != nil {
		return errors.Wrapf(err, "couldn't make directory %s", parentDir)
	}
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(filepath.FromSlash(writePath), marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save version lock to %s", filepath.FromSlash(writePath))
	}
	return nil
}

// Remove

func RemoveRepoReqs(
	indent int, pallet *forklift.FSPallet, repoPaths []string, force bool,
) error {
	usedRepoReqs, err := determineUsedRepoReqs(indent, pallet, force)
	if err != nil {
		return errors.Wrap(
			err,
			"couldn't determine repos used by declared package deployments, to check which repositories "+
				"to remove are still required by declared deployments; to skip this check, enable the "+
				"--force flag",
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
		if err = os.RemoveAll(filepath.FromSlash(path.Join(
			repoReqPath, forklift.VersionLockDefFile,
		))); err != nil {
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
	if len(depls) == 0 {
		return usedRepoReqs, nil
	}

	repoReqsFS, err := pallet.GetRepoReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for repo requirements from pallet")
	}
	for _, depl := range depls {
		pkgPath := depl.Def.Package
		if path.IsAbs(pkgPath) { // special case: package is provided by the pallet itself
			continue
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

func DownloadAllRequiredRepos(
	indent int, pallet *forklift.FSPallet, repoCache forklift.PathedRepoCache,
	palletCache forklift.PathedPalletCache, skipPalletQueries structures.Set[string],
) (changed bool, err error) {
	loadedRepoReqs, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify repos")
	}
	if len(loadedRepoReqs) == 0 {
		return false, nil
	}

	IndentedPrintln(indent, "Downloading required repos...")
	return downloadRequiredRepos(indent+1, loadedRepoReqs, repoCache, palletCache, skipPalletQueries)
}

func downloadRequiredRepos(
	indent int, reqs []*forklift.FSRepoReq, repoCache forklift.PathedRepoCache,
	palletCache forklift.PathedPalletCache, skipPalletQueries structures.Set[string],
) (changed bool, err error) {
	allSkip := make(structures.Set[string])
	maps.Insert(allSkip, maps.All(skipPalletQueries))
	for _, req := range reqs {
		downloaded, err := DownloadLockedGitRepoUsingLocalMirror(
			indent, repoCache.Path(), req.Path(), req.VersionLock,
		)
		changed = changed || downloaded
		if err != nil {
			return changed, errors.Wrapf(
				err, "couldn't download %s at commit %s", req.Path(), req.VersionLock.Def.ShortCommit(),
			)
		}
		if !changed {
			continue
		}
		if _, err := forklift.LoadFSPallet(
			core.AttachPath(os.DirFS(repoCache.Path()), repoCache.Path()), req.GetQueryPath(),
		); err != nil {
			// the repo is not a pallet, so we can use it directly as a repo
			continue
		}

		if err = os.RemoveAll(filepath.FromSlash(path.Join(
			repoCache.Path(), req.GetQueryPath(),
		))); err != nil {
			return changed, errors.Wrapf(
				err, "couldn't delete download of repo %s in order to cache it as a layered pallet",
				req.GetQueryPath(),
			)
		}
		IndentedPrintln(indent+1, "Re-downloading repo as pallet...")
		downloadedPallets, err := downloadRequiredPallets(indent+1, []*forklift.FSPalletReq{
			{
				PalletReq: forklift.PalletReq{GitRepoReq: req.GitRepoReq},
				FS:        req.FS,
			},
		}, palletCache, allSkip)
		maps.Insert(allSkip, maps.All(downloadedPallets))
		if err != nil {
			return changed, err
		}
		if err = cacheRepoFromCachedPallet(
			req.Path(), req.VersionLock.Version, repoCache, palletCache,
		); err != nil {
			return changed, errors.Wrapf(
				err, "couldn't create cached repo %s from pallet", req.GetQueryPath(),
			)
		}
	}
	return changed, nil
}

func cacheRepoFromCachedPallet(
	repoPath, repoVersion string,
	repoCache forklift.PathedRepoCache, palletCache forklift.PathedPalletCache,
) error {
	plt, err := palletCache.LoadFSPallet(repoPath, repoVersion)
	if err != nil {
		return err
	}
	merged, err := forklift.MergeFSPallet(plt, palletCache, nil)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't merge repo %s as a pallet with any pallets required by it", plt.Path(),
		)
	}
	if err = forklift.CopyFS(
		merged.FS, filepath.FromSlash(path.Join(
			repoCache.Path(), fmt.Sprintf("%s@%s", repoPath, repoVersion),
		)),
	); err != nil {
		return errors.Wrapf(err, "couldn't copy merged pallet %s into repo cache", plt.Path())
	}
	if _, err = repoCache.LoadFSRepo(repoPath, repoVersion); err != nil {
		if err = core.WriteRepoDef(
			core.RepoDef{
				ForkliftVersion: merged.Def.ForkliftVersion,
				Repo: core.RepoSpec{
					Path:        merged.Def.Pallet.Path,
					Description: merged.Def.Pallet.Description,
					ReadmeFile:  merged.Def.Pallet.ReadmeFile,
				},
			},
			filepath.FromSlash(path.Join(
				repoCache.Path(), fmt.Sprintf("%s@%s", repoPath, repoVersion), core.RepoDefFile,
			)),
		); err != nil {
			return errors.Wrap(err, "couldn't initialize repo declaration from pallet declaration")
		}
	}
	return nil
}
