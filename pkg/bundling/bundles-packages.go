package bundling

import (
	"path"
	"strings"

	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/pkg/errors"
)

// packagesDirName is the name of the directory containing bundled files for each package.
const packagesDirName = "packages"

// FSBundle: Packages

func (b *FSBundle) getPackagesPath() string {
	return path.Join(b.FS.Path(), packagesDirName)
}

// FSBundle: FSPkgLoader

func (b *FSBundle) LoadFSPkg(pkgPath string, version string) (*fpkg.FSPkg, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	return b.FSPkgTree.LoadFSPkg(strings.TrimLeft(pkgPath, "/"))
}

func (b *FSBundle) LoadFSPkgs(searchPattern string) ([]*fpkg.FSPkg, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	return b.FSPkgTree.LoadFSPkgs(searchPattern)
}

// FSBundle: PkgReqLoader

func (b *FSBundle) LoadPkgReq(pkgPath string) (r fplt.PkgReq, err error) {
	return fplt.PkgReq{
		PkgSubdir: strings.TrimLeft(pkgPath, "/"),
	}, nil
}
