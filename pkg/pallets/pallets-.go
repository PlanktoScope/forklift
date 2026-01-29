// Package pallets implements the Forklift pallets specification for deployment and composition of
// Forklift packages.
package pallets

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
)

// A FSPallet is a Forklift pallet stored at the root of a [fs.FS] filesystem.
type FSPallet struct {
	// Pallet is the pallet at the root of the filesystem.
	Pallet
	// FSPkgTree is the package tree at the root of the pallet's filesystem.
	FSPkgTree *fpkg.FSPkgTree
	// FS is a filesystem which contains the pallet's contents.
	FS ffs.PathedFS
}

// A Pallet is a Forklift pallet, a complete specification of all package deployments which should
// be active on a Docker host.
type Pallet struct {
	// Decl is the Forklift pallet definition for the pallet.
	Decl PalletDecl
	// Version is the version or pseudoversion of the pallet.
	Version string
}

// FSPallet

// LoadFSPallet loads a FSPallet from the specified directory path in the provided base filesystem.
func LoadFSPallet(fsys ffs.PathedFS, subdirPath string) (p *FSPallet, err error) {
	p = &FSPallet{}
	if p.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if p.Pallet.Decl, err = loadPalletDecl(p.FS, PalletDeclFile); err != nil {
		return nil, errors.Errorf("couldn't load pallet config")
	}
	p.FSPkgTree = &fpkg.FSPkgTree{
		FS:       p.FS,
		RootPath: p.Path(),
		Version:  p.Pallet.Version,
	}
	return p, nil
}

// LoadFSPalletContaining loads the FSPallet containing the specified sub-directory path in the
// provided base filesystem.
// The provided path should use the host OS's path separators.
// The sub-directory path does not have to actually exist.
func LoadFSPalletContaining(path string) (*FSPallet, error) {
	palletCandidatePath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
	}
	for {
		if fsPallet, err := LoadFSPallet(ffs.DirFS(palletCandidatePath), "."); err == nil {
			return fsPallet, nil
		}

		palletCandidatePath = filepath.Dir(palletCandidatePath)
		if palletCandidatePath == "/" || palletCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf("no pallet config file found in any parent directory of %s", path)
		}
	}
}

// LoadFSPallets loads all FSPallets from the provided base filesystem matching the specified search
// pattern. The search pattern should be a [doublestar] pattern, such as `**`, matching pallet
// directories to search for.
// In the embedded [Pallet] of each loaded FSPallet, the version is *not* initialized.
func LoadFSPallets(fsys ffs.PathedFS, searchPattern string) ([]*FSPallet, error) {
	searchPattern = path.Join(searchPattern, PalletDeclFile)
	palletDeclFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for pallet config files matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	orderedPallets := make([]*FSPallet, 0, len(palletDeclFiles))
	pallets := make(map[string]*FSPallet)
	for _, palletDeclFilePath := range palletDeclFiles {
		if path.Base(palletDeclFilePath) != PalletDeclFile {
			continue
		}
		pallet, err := LoadFSPallet(fsys, path.Dir(palletDeclFilePath))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load pallet from %s/%s", fsys.Path(), palletDeclFilePath,
			)
		}

		orderedPallets = append(orderedPallets, pallet)
		pallets[pallet.Path()] = pallet
	}

	return orderedPallets, nil
}

// Exists checks whether the pallet actually exists on the OS's filesystem.
func (p *FSPallet) Exists() bool {
	return ffs.DirExists(p.FS.Path())
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (p *FSPallet) Remove() error {
	return os.RemoveAll(p.FS.Path())
}

// LoadReadme loads the readme file defined by the pallet.
func (p *FSPallet) LoadReadme() ([]byte, error) {
	readmePath := p.Decl.Pallet.ReadmeFile
	bytes, err := fs.ReadFile(p.FS, readmePath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't read pallet readme %s/%s", p.FS.Path(), readmePath)
	}
	return bytes, nil
}

// Path returns either the pallet's path (if specified) or its path on the filesystem.
func (p *FSPallet) Path() string {
	if p.Decl.Pallet.Path == "" {
		return p.FS.Path()
	}
	return p.Decl.Pallet.Path
}

// FSPallet: Pallet Requirements

// GetPalletReqsFS returns the [fs.FS] in the pallet which contains pallet requirement
// definitions.
func (p *FSPallet) GetPalletReqsFS() (ffs.PathedFS, error) {
	return p.FS.Sub(path.Join(ReqsDirName, ReqsPalletsDirName))
}

// LoadFSPalletReq loads the FSPalletReq from the pallet for the pallet with the specified
// path.
func (p *FSPallet) LoadFSPalletReq(palletPath string) (r *FSPalletReq, err error) {
	palletsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for pallet requirements from pallet")
	}
	if r, err = loadFSPalletReq(palletsFS, palletPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't load pallet %s", palletPath)
	}
	return r, nil
}

// LoadFSPalletReqs loads all FSPalletReqs from the pallet matching the specified search
// pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching the pallet paths to
// search for.
func (p *FSPallet) LoadFSPalletReqs(searchPattern string) ([]*FSPalletReq, error) {
	palletReqsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for pallets in pallet")
	}
	return loadFSPalletReqs(palletReqsFS, searchPattern)
}

// loadPalletReq loads the PalletReq from the pallet with the specified pallet path.
func (p *FSPallet) loadPalletReq(palletPath string) (r PalletReq, err error) {
	fsPalletReq, err := p.LoadFSPalletReq(palletPath)
	if err != nil {
		return PalletReq{}, errors.Wrapf(err, "couldn't find pallet %s", palletPath)
	}
	return fsPalletReq.PalletReq, nil
}

// FSPallet: Package Requirements

// LoadPkgReq loads the PkgReq from the pallet for the package with the specified package path.
func (p *FSPallet) LoadPkgReq(pkgPath string) (r PkgReq, err error) {
	if path.IsAbs(pkgPath) { // special case: package should be provided by the pallet itself
		return PkgReq{
			PkgSubdir: strings.TrimLeft(pkgPath, "/"),
			Pallet: PalletReq{
				GitRepoReq{RequiredPath: p.Decl.Pallet.Path},
			},
		}, nil
	}

	palletsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return r, errors.Wrap(err, "couldn't open directory for pallet requirements from pallet")
	}
	fsPalletReq, err := LoadFSPalletReqContaining(palletsFS, pkgPath)
	if err != nil {
		return r, errors.Wrapf(err, "couldn't find pallet providing package %s in pallet", pkgPath)
	}
	r.Pallet = fsPalletReq.PalletReq
	r.PkgSubdir = fsPalletReq.GetPkgSubdir(pkgPath)
	return r, nil
}

// FSPallet: Deployments

// GetDeplsFS returns the [fs.FS] in the pallet which contains package deployment declarations.
func (p *FSPallet) GetDeplsFS() (ffs.PathedFS, error) {
	return p.FS.Sub(DeplsDirName)
}

// LoadDepl loads the Depl with the specified name from the pallet.
func (p *FSPallet) LoadDepl(name string) (depl Depl, err error) {
	deplsFS, err := p.GetDeplsFS()
	if err != nil {
		return Depl{}, errors.Wrap(
			err, "couldn't open directory for package deployment declarations from pallet",
		)
	}
	if depl, err = loadDepl(deplsFS, name); err != nil {
		return Depl{}, errors.Wrapf(err, "couldn't load package deployment for %s", name)
	}
	return depl, nil
}

// LoadDepls loads all package deployment declarations matching the specified search pattern.
// The search pattern should not include the file extension for deployment specification files - the
// file extension will be appended to the search pattern by LoadDepls.
func (p *FSPallet) LoadDepls(searchPattern string) ([]Depl, error) {
	fsys, err := p.GetDeplsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for package deployment declarations from pallet",
		)
	}
	return loadDepls(fsys, searchPattern)
}

// FSPallet: Packages

// LoadFSPkg loads a package at the specified filesystem path from the FSPallet instance
// The loaded package is fully initialized.
func (p *FSPallet) LoadFSPkg(pkgSubdir string) (pkg *fpkg.FSPkg, err error) {
	return p.FSPkgTree.LoadFSPkg(pkgSubdir)
}

// LoadFSPkgs loads all packages in the FSPallet instance.
// The loaded packages are fully initialized.
func (p *FSPallet) LoadFSPkgs(searchPattern string) ([]*fpkg.FSPkg, error) {
	return p.FSPkgTree.LoadFSPkgs(searchPattern)
}

// FSPallet: Imports

// LoadImport loads the Import with the specified name from the pallet.
func (p *FSPallet) LoadImport(name string) (imp Import, err error) {
	impsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return Import{}, errors.Wrap(err, "couldn't open directory for import groups from pallet")
	}
	if imp, err = loadImport(impsFS, name, ImportDeclFileExt); err != nil {
		return Import{}, errors.Wrapf(err, "couldn't load import group for %s", name)
	}
	return imp, nil
}

// LoadImports loads all package deployment declarations matching the specified search pattern.
// The search pattern should not include the file extension for import group files - the
// file extension will be appended to the search pattern by LoadImports.
func (p *FSPallet) LoadImports(searchPattern string) ([]Import, error) {
	fsys, err := p.GetPalletReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for import groups from pallet")
	}
	return loadImports(fsys, searchPattern, ImportDeclFileExt)
}

// Pallet

// Path returns the pallet path of the Pallet instance.
func (p Pallet) Path() string {
	return p.Decl.Pallet.Path
}

// VersionQuery represents the Pallet instance as a version query.
func (p Pallet) VersionQuery() string {
	return fmt.Sprintf("%s@%s", p.Path(), p.Version)
}

// Check looks for errors in the construction of the pallet.
func (p Pallet) Check() (errs []error) {
	errs = append(errs, errsWrap(p.Decl.Check(), "invalid pallet config")...)
	return errs
}

func errsWrap(errs []error, message string) []error {
	wrapped := make([]error, 0, len(errs))
	for _, err := range errs {
		wrapped = append(wrapped, errors.Wrap(err, message))
	}
	return wrapped
}

// ComparePallets returns an integer comparing two [Pallet] instances according to their paths and
// versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func ComparePallets(r, s Pallet) int {
	if result := cmp.Compare(r.Path(), s.Path()); result != 0 {
		return result
	}
	if result := semver.Compare(r.Version, s.Version); result != 0 {
		return result
	}
	return 0
}
