package pallets

import (
	"fmt"
	"path"

	"github.com/pkg/errors"

	fpkg "github.com/forklift-run/forklift/exp/packaging"
)

// A PkgReq is a requirement for a package at a specific version.
type PkgReq struct {
	// PkgSubdir is the package subdirectory in the pallet which should provide the required package.
	PkgSubdir string
	// Pallet is the pallet which should provide the required package.
	Pallet PalletReq
}

// FSPkgLoader is a source of [fpkg.FSPkg]s indexed by path and version.
type FSPkgLoader interface {
	// LoadFSPkg loads the FSPkg with the specified path and version.
	LoadFSPkg(pkgPath string, version string) (*fpkg.FSPkg, error)
	// LoadFSPkgs loads all FSPkgs matching the specified search pattern.
	LoadFSPkgs(searchPattern string) ([]*fpkg.FSPkg, error)
}

// PkgReqLoader is a source of package requirements.
type PkgReqLoader interface {
	LoadPkgReq(pkgPath string) (PkgReq, error)
}

// LoadRequiredFSPkg loads the specified package from the cache according to the specifications in
// the package requirements provided by the package requirement loader for the provided package
// path.
func LoadRequiredFSPkg(
	pkgReqLoader PkgReqLoader, pkgLoader FSPkgLoader, pkgPath string,
) (*fpkg.FSPkg, PkgReq, error) {
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

// GetQueryPath returns the path of the package in version queries, which is of form
// palletPath/pkgSubdir@version
// (e.g. github.com/PlanktoScope/pallet-standard/packages/fpkg/infra/caddy-ingress@v2024.0.0).
func (r PkgReq) GetQueryPath() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.Pallet.VersionLock.Version)
}

// Path returns the package path of the required package.
func (r PkgReq) Path() string {
	return path.Join(r.Pallet.Path(), r.PkgSubdir)
}
