package forklift

import (
	"fmt"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	"github.com/forklift-run/forklift/pkg/versioning"
)

const (
	// ReqsPalletsDirName is the subdirectory in the requirements directory of a Forklift pallet which
	// contains pallet requirement declarations.
	ReqsPalletsDirName = "pallets"
)

// A FSPalletReq is a pallet requirement stored at the root of a [fs.FS] filesystem.
type FSPalletReq struct {
	// PalletReq is the pallet requirement at the root of the filesystem.
	PalletReq
	// FS is a filesystem which contains the pallet requirement's contents.
	FS ffs.PathedFS
}

// A PalletReq is a requirement for a specific pallet at a specific version.
type PalletReq struct {
	GitRepoReq `yaml:",inline"`
}

// FSPalletLoader is a source of [FSPallet]s indexed by path and version.
type FSPalletLoader interface {
	// LoadFSPallet loads the FSPallet with the specified path and version.
	LoadFSPallet(palletPath string, version string) (*FSPallet, error)
	// LoadFSPallets loads all FSPallets matching the specified search pattern.
	LoadFSPallets(searchPattern string) ([]*FSPallet, error)
}

// FSPalletReq

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
	palletCandidatePath := subdirPath
	for {
		if req, err := loadFSPalletReq(fsys, palletCandidatePath); err == nil {
			return req, nil
		}
		palletCandidatePath = path.Dir(palletCandidatePath)
		if palletCandidatePath == "/" || palletCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no pallet requirement declaration found in any parent directory of %s", subdirPath,
			)
		}
	}
}

// palletReqLoader is a source of pallet requirements.
type palletReqLoader interface {
	loadPalletReq(palletPath string) (PalletReq, error)
}

// loadRequiredFSPallet loads the specified pallet from the cache according to the specifications in
// the pallet requirements provided by the pallet requirement loader for the provided pallet
// path.
func loadRequiredFSPallet(
	palletReqLoader palletReqLoader, palletLoader FSPalletLoader, palletPath string,
) (*FSPallet, error) {
	req, err := palletReqLoader.loadPalletReq(palletPath)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't determine pallet requirement for pallet %s", palletPath,
		)
	}
	fsPallet, err := palletLoader.LoadFSPallet(req.Path(), req.VersionLock.Version)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load required pallet %s", req.GetQueryPath(),
		)
	}
	return fsPallet, nil
}

// PalletReq

// GetPkgSubdir returns the package subdirectory within the required repo for the provided package
// path.
func (r PalletReq) GetPkgSubdir(pkgPath string) string {
	return strings.TrimPrefix(pkgPath, fmt.Sprintf("%s/", r.Path()))
}
