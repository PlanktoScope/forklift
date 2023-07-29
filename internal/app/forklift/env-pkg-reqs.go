package forklift

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

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
	fsPkg, err := pkgLoader.LoadFSPkg(req.Path(), req.Repo.VersionLock.Version)
	if err != nil {
		return nil, PkgReq{}, errors.Wrapf(err, "couldn't load required package %s", req.GetQueryPath())
	}
	return fsPkg, req, nil
}

// GetCachePath returns the path of the package in caches, which is of form
// vcsPath@version/repoSubdir/pkgSubdir
// (e.g. github.com/PlanktoScope/pallets@v0.1.0/core/infrastructure/caddy-ingress).
func (r PkgReq) GetCachePath() string {
	return filepath.Join(r.Repo.GetCachePath(), r.PkgSubdir)
}

// GetQueryPath returns the path of the package in version queries, which is of form
// vcsPath/repoSubdir/pkgSubdir@version
// (e.g. github.com/PlanktoScope/pallets/core/infrastructure/caddy-ingress@v0.1.0).
func (r PkgReq) GetQueryPath() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.Repo.VersionLock.Version)
}

// Path returns the Pallet package path of the required package.
func (r PkgReq) Path() string {
	return filepath.Join(r.Repo.Path(), r.PkgSubdir)
}
