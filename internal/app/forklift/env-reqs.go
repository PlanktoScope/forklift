package forklift

import (
	"fmt"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// PkgReq

// LoadRequiredFSPkg loads the specified package from the cache according to the specifications in
// the package requirements provided by the package requirement loader for the provided package
// path.
func LoadRequiredFSPkg(
	pkgReqLoader PkgReqLoader, pkgLoader FSPkgLoader, pkgPath string,
) (*pallets.FSPkg, PkgReq, error) {
	req, err := pkgReqLoader.LoadPkgReq(pkgPath)
	if err != nil {
		return nil, PkgReq{}, errors.Wrapf(
			err, "couldn't determine package requirement for package %s", pkgPath,
		)
	}
	fsPkg, err := pkgLoader.LoadFSPkg(req.Path(), req.Pallet.VersionLock.Version)
	if err != nil {
		return nil, PkgReq{}, errors.Wrapf(err, "couldn't load required package %s", req.GetQueryPath())
	}
	return fsPkg, req, nil
}

// GetCachePath returns the path of the package in caches, which is of form
// vcsRepoPath@version/palletSubdir/pkgSubdir
// (e.g. github.com/PlanktoScope/pallets@v0.1.0/core/infrastructure/caddy-ingress).
func (r PkgReq) GetCachePath() string {
	return path.Join(r.Pallet.GetCachePath(), r.PkgSubdir)
}

// GetQueryPath returns the path of the package in version queries, which is of form
// vcsPath/palletSubdir/pkgSubdir@version
// (e.g. github.com/PlanktoScope/pallets/core/infrastructure/caddy-ingress@v0.1.0).
func (r PkgReq) GetQueryPath() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.Pallet.VersionLock.Version)
}

// Path returns the package path of the required package.
func (r PkgReq) Path() string {
	return path.Join(r.Pallet.Path(), r.PkgSubdir)
}

// PalletReq

// LoadFSPallet loads a FSPalletReq from the specified directory path in the provided base
// filesystem, assuming the directory path is also the path of the required pallet.
func loadFSPalletReq(fsys pallets.PathedFS, palletPath string) (r *FSPalletReq, err error) {
	r = &FSPalletReq{}
	if r.FS, err = fsys.Sub(palletPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", palletPath, fsys.Path(),
		)
	}
	if r.VCSRepoPath, r.PalletSubdir, err = pallets.SplitRepoPathSubdir(
		palletPath,
	); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't parse path of requirement for pallet %s", palletPath,
		)
	}
	r.VersionLock, err = loadVersionLock(r.FS, VersionLockDefFile)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load version lock config of requirement for pallet %s", palletPath,
		)
	}
	return r, nil
}

// loadFSPalletReqContaining loads the FSPalletReq containing the specified sub-directory path in
// the provided base filesystem.
// The sub-directory path does not have to actually exist; however, it would usually be provided
// as a package path.
func loadFSPalletReqContaining(fsys pallets.PathedFS, subdirPath string) (*FSPalletReq, error) {
	palletCandidatePath := subdirPath
	for {
		if pallet, err := loadFSPalletReq(fsys, palletCandidatePath); err == nil {
			return pallet, nil
		}
		palletCandidatePath = path.Dir(palletCandidatePath)
		if palletCandidatePath == "/" || palletCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no pallet requirement config file found in any parent directory of %s", subdirPath,
			)
		}
	}
}

// loadFSPalletReqs loads all FSPalletReqs from the provided base filesystem matching the specified
// search pattern, assuming the directory paths in the base filesystem are also the paths of the
// required pallets. The search pattern should be a [doublestar] pattern, such as `**`, matching the
// pallet paths to search for.
// With a nil processor function, in the embedded [Pallet] of each loaded FSPallet, the VCS
// repository path and pallet subdirectory are initialized from the pallet path declared in the
// pallet's configuration file, while the pallet's version is not initialized.
func loadFSPalletReqs(fsys pallets.PathedFS, searchPattern string) ([]*FSPalletReq, error) {
	searchPattern = path.Join(searchPattern, VersionLockDefFile)
	palletReqFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for pallet requirement files matching %s/%s",
			fsys.Path(), searchPattern,
		)
	}

	reqs := make([]*FSPalletReq, 0, len(palletReqFiles))
	for _, palletReqDefFilePath := range palletReqFiles {
		if path.Base(palletReqDefFilePath) != VersionLockDefFile {
			continue
		}

		req, err := loadFSPalletReq(fsys, path.Dir(palletReqDefFilePath))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load pallet requirement from %s", palletReqDefFilePath,
			)
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

// Path returns the path of the required pallet.
func (r PalletReq) Path() string {
	return path.Join(r.VCSRepoPath, r.PalletSubdir)
}

// GetPkgSubdir returns the package subdirectory within the required pallet for the provided package
// path.
func (r PalletReq) GetPkgSubdir(pkgPath string) string {
	return strings.TrimPrefix(pkgPath, fmt.Sprintf("%s/", r.Path()))
}

// GetCachePath returns the path of the pallet in caches, which is of form
// vcsRepoPath@version/palletSubdir (e.g. github.com/PlanktoScope/pallets@v0.1.0/core).
func (r PalletReq) GetCachePath() string {
	return fmt.Sprintf("%s@%s/%s", r.VCSRepoPath, r.VersionLock.Version, r.PalletSubdir)
}

// GetQueryPath returns the path of the pallet in version queries, which is of form
// vcsRepoPath/palletSubdir@version (e.g. github.com/PlanktoScope/pallets/core@v0.1.0).
func (r PalletReq) GetQueryPath() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.VersionLock.Version)
}

// ComparePalletReqs returns an integer comparing two [PalletReq] instances according to their paths
// and versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func ComparePalletReqs(r, s PalletReq) int {
	if result := pallets.ComparePaths(r.Path(), s.Path()); result != pallets.CompareEQ {
		return result
	}
	if result := semver.Compare(
		r.VersionLock.Version, s.VersionLock.Version,
	); result != pallets.CompareEQ {
		return result
	}
	return pallets.CompareEQ
}
