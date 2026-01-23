package forklift

import (
	"fmt"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/forklift-run/forklift/pkg/core"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	"github.com/forklift-run/forklift/pkg/versioning"
)

// GitRepoReq

// Path returns the path of the required Git repo.
func (r GitRepoReq) Path() string {
	return r.RequiredPath
}

// GetQueryPath returns the path of the Git repo in version queries, which is of form
// gitRepoPath@version (e.g. github.com/PlanktoScope/pallet-standard@v2024.0.0).
func (r GitRepoReq) GetQueryPath() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.VersionLock.Version)
}

// CompareGitRepoReqs returns an integer comparing two [RepoReq] instances according to their paths
// and versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func CompareGitRepoReqs(r, s GitRepoReq) int {
	if result := core.ComparePaths(r.Path(), s.Path()); result != core.CompareEQ {
		return result
	}
	if result := semver.Compare(
		r.VersionLock.Version, s.VersionLock.Version,
	); result != core.CompareEQ {
		return result
	}
	return core.CompareEQ
}

// PalletReq

// LoadFSPallet loads a FSPalletReq from the specified directory path in the provided base
// filesystem, assuming the directory path is also the path of the required pallet.
func loadFSPalletReq(fsys ffs.PathedFS, palletPath string) (r *FSPalletReq, err error) {
	r = &FSPalletReq{}
	if r.FS, err = fsys.Sub(palletPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", palletPath, fsys.Path(),
		)
	}
	r.RequiredPath = palletPath
	r.VersionLock, err = versioning.LoadLock(r.FS, versioning.LockDeclFile)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load version lock declaration of requirement for pallet %s", palletPath,
		)
	}
	return r, nil
}

// loadFSPalletReqs loads all FSPalletReqs from the provided base filesystem matching the specified
// search pattern, assuming the directory paths in the base filesystem are also the paths of the
// required pallets. The search pattern should be a [doublestar] pattern, such as `**`, matching the
// pallet paths to search for.
func loadFSPalletReqs(fsys ffs.PathedFS, searchPattern string) ([]*FSPalletReq, error) {
	searchPattern = path.Join(searchPattern, versioning.LockDeclFile)
	palletReqFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for pallet requirement files matching %s/%s",
			fsys.Path(), searchPattern,
		)
	}

	reqs := make([]*FSPalletReq, 0, len(palletReqFiles))
	for _, palletReqDeclFilePath := range palletReqFiles {
		if path.Base(palletReqDeclFilePath) != versioning.LockDeclFile {
			continue
		}

		req, err := loadFSPalletReq(fsys, path.Dir(palletReqDeclFilePath))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load pallet requirement from %s", palletReqDeclFilePath,
			)
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

// LoadFSPalletReqContaining loads the FSPalletReq containing the specified sub-directory path in
// the provided base filesystem.
// The sub-directory path does not have to actually exist; however, it would usually be provided
// as a package path.
func LoadFSPalletReqContaining(fsys ffs.PathedFS, subdirPath string) (*FSPalletReq, error) {
	repoCandidatePath := subdirPath
	for {
		if repo, err := loadFSPalletReq(fsys, repoCandidatePath); err == nil {
			return repo, nil
		}
		repoCandidatePath = path.Dir(repoCandidatePath)
		if repoCandidatePath == "/" || repoCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no pallet requirement declaration found in any parent directory of %s", subdirPath,
			)
		}
	}
}

// LoadRequiredFSPallet loads the specified pallet from the cache according to the specifications in
// the pallet requirements provided by the pallet requirement loader for the provided pallet
// path.
func LoadRequiredFSPallet(
	palletReqLoader PalletReqLoader, palletLoader FSPalletLoader, palletPath string,
) (*FSPallet, PalletReq, error) {
	req, err := palletReqLoader.LoadPalletReq(palletPath)
	if err != nil {
		return nil, PalletReq{}, errors.Wrapf(
			err, "couldn't determine pallet requirement for pallet %s", palletPath,
		)
	}
	fsPallet, err := palletLoader.LoadFSPallet(req.Path(), req.VersionLock.Version)
	if err != nil {
		return nil, PalletReq{}, errors.Wrapf(
			err, "couldn't load required pallet %s", req.GetQueryPath(),
		)
	}
	return fsPallet, req, nil
}

// GetCachePath returns the path of the pallet in caches, which is of form
// palletPath@version (e.g. github.com/PlanktoScope/pallet-standard@v2024.0.0).
func (r PalletReq) GetCachePath() string {
	return r.GetQueryPath()
}

// GetQueryPath returns the path of the pallet in version queries, which is of form
// palletPath@version (e.g. github.com/PlanktoScope/pallet-standard@v2024.0.0).
func (r PalletReq) GetQueryPath() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.VersionLock.Version)
}

// RepoReq

// LoadFSRepo loads a FSRepoReq from the specified directory path in the provided base
// filesystem, assuming the directory path is also the path of the required repo.
func loadFSRepoReq(fsys ffs.PathedFS, repoPath string) (r *FSRepoReq, err error) {
	r = &FSRepoReq{}
	if r.FS, err = fsys.Sub(repoPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", repoPath, fsys.Path(),
		)
	}
	r.RequiredPath = repoPath
	r.VersionLock, err = versioning.LoadLock(r.FS, versioning.LockDeclFile)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load version lock declaration of requirement for repo %s", repoPath,
		)
	}
	return r, nil
}

// LoadFSRepoReqContaining loads the FSRepoReq containing the specified sub-directory path in
// the provided base filesystem.
// The sub-directory path does not have to actually exist; however, it would usually be provided
// as a package path.
func LoadFSRepoReqContaining(fsys ffs.PathedFS, subdirPath string) (*FSRepoReq, error) {
	repoCandidatePath := subdirPath
	for {
		if repo, err := loadFSRepoReq(fsys, repoCandidatePath); err == nil {
			return repo, nil
		}
		repoCandidatePath = path.Dir(repoCandidatePath)
		if repoCandidatePath == "/" || repoCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no repo requirement declaration found in any parent directory of %s", subdirPath,
			)
		}
	}
}

// loadFSRepoReqs loads all FSRepoReqs from the provided base filesystem matching the specified
// search pattern, assuming the directory paths in the base filesystem are also the paths of the
// required repos. The search pattern should be a [doublestar] pattern, such as `**`, matching the
// repo paths to search for.
func loadFSRepoReqs(fsys ffs.PathedFS, searchPattern string) ([]*FSRepoReq, error) {
	searchPattern = path.Join(searchPattern, versioning.LockDeclFile)
	repoReqFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for repo requirement files matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	reqs := make([]*FSRepoReq, 0, len(repoReqFiles))
	for _, repoReqDeclFilePath := range repoReqFiles {
		if path.Base(repoReqDeclFilePath) != versioning.LockDeclFile {
			continue
		}

		req, err := loadFSRepoReq(fsys, path.Dir(repoReqDeclFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load repo requirement from %s", repoReqDeclFilePath)
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

// GetPkgSubdir returns the package subdirectory within the required repo for the provided package
// path.
func (r RepoReq) GetPkgSubdir(pkgPath string) string {
	return strings.TrimPrefix(pkgPath, fmt.Sprintf("%s/", r.Path()))
}

// GetCachePath returns the path of the repo in caches, which is of form
// repoPath@version (e.g. github.com/PlanktoScope/pallet-standard@v2024.0.0).
func (r RepoReq) GetCachePath() string {
	return r.GetQueryPath()
}

// PkgReq

// LoadRequiredFSPkg loads the specified package from the cache according to the specifications in
// the package requirements provided by the package requirement loader for the provided package
// path.
func LoadRequiredFSPkg(
	pkgReqLoader PkgReqLoader, pkgLoader FSPkgLoader, pkgPath string,
) (*core.FSPkg, PkgReq, error) {
	req, err := pkgReqLoader.LoadPkgReq(pkgPath)
	if err != nil {
		return nil, PkgReq{}, errors.Wrapf(
			err, "couldn't determine package requirement for package %s", pkgPath,
		)
	}
	fsPkg, err := pkgLoader.LoadFSPkg(req.Path(), req.Repo.VersionLock.Version)
	if err != nil {
		return nil, PkgReq{}, errors.Wrapf(err, "couldn't load required package %s", req.GetQueryPath())
	}
	return fsPkg, req, nil
}

// GetCachePath returns the path of the package in caches, which is of form
// repoPath@version/pkgSubdir
// (e.g. github.com/PlanktoScope/pallet-standard@v2024.0.0/packages/core/infra/caddy-ingress).
func (r PkgReq) GetCachePath() string {
	return path.Join(r.Repo.GetCachePath(), r.PkgSubdir)
}

// GetQueryPath returns the path of the package in version queries, which is of form
// repoPath/pkgSubdir@version
// (e.g. github.com/PlanktoScope/pallet-standard/packages/core/infra/caddy-ingress@v2024.0.0).
func (r PkgReq) GetQueryPath() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.Repo.VersionLock.Version)
}

// Path returns the package path of the required package.
func (r PkgReq) Path() string {
	return path.Join(r.Repo.Path(), r.PkgSubdir)
}
