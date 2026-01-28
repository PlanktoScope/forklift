package packaging

import (
	"fmt"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// A FSPkg is a Forklift package stored at the root of a [fs.FS] filesystem.
type FSPkg struct {
	// Pkg is the Forklift package at the root of the filesystem.
	Pkg
	// FS is a filesystem which contains the package's contents.
	FS ffs.PathedFS
	// FSPkgTree is a pointer to the [FSPkgTree] instance which provides the package.
	FSPkgTree *FSPkgTree
}

// A Pkg is a Forklift package, a configuration of a software application which can be deployed on a
// Docker host.
type Pkg struct {
	// ParentPath is the path of the package tree which provides the package.
	ParentPath string
	// Subdir is the path of the package within the package tree which provides the package.
	Subdir string
	// Decl is the definition of the package.
	Decl PkgDecl
}

// FSPkg

// LoadFSPkg loads a FSPkg from the specified directory path in the provided base filesystem.
// In the loaded FSPkg's embedded [Pkg], the repo path is not initialized, nor is the repo
// subdirectory initialized, nor is the pointer to the repo initialized.
func LoadFSPkg(fsys ffs.PathedFS, subdirPath string) (p *FSPkg, err error) {
	p = &FSPkg{}
	if p.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if p.Pkg.Decl, err = LoadPkgDecl(p.FS, PkgDeclFile); err != nil {
		return nil, errors.Wrapf(err, "couldn't load package declaration")
	}
	return p, nil
}

// LoadFSPkgs loads all FSPkgs from the provided base filesystem matching the specified search
// pattern. The search pattern should be a [doublestar] pattern, such as `**`, matching package
// directories to search for.
// The pkg tree path, and the package subdirectory, and the pointer to the pkg tree are all left
// uninitialized.
func LoadFSPkgs(fsys ffs.PathedFS, searchPattern string) ([]*FSPkg, error) {
	searchPattern = path.Join(searchPattern, PkgDeclFile)
	pkgDeclFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for package declarations matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	pkgs := make([]*FSPkg, 0, len(pkgDeclFiles))
	for _, pkgDeclFilePath := range pkgDeclFiles {
		if path.Base(pkgDeclFilePath) != PkgDeclFile {
			continue
		}

		pkg, err := LoadFSPkg(fsys, path.Dir(pkgDeclFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load package from %s", pkgDeclFilePath)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

// AttachFSPkgTree updates the FSPkg instance's Subdir, Pkg.FSPkgTree, and FSPkgTree fields
// based on the provided pkg tree.
func (p *FSPkg) AttachFSPkgTree(pkgTree *FSPkgTree) error {
	p.ParentPath = pkgTree.Path()
	if !strings.HasPrefix(p.FS.Path(), fmt.Sprintf("%s/", pkgTree.FS.Path())) {
		return errors.Errorf(
			"package at %s is not within the scope of pkg tree %s at %s",
			p.FS.Path(), pkgTree.FS.Path(), pkgTree.FS.Path(),
		)
	}
	p.Subdir = strings.TrimPrefix(p.FS.Path(), fmt.Sprintf("%s/", pkgTree.FS.Path()))
	p.FSPkgTree = pkgTree
	return nil
}

// Check looks for errors in the construction of the package.
func (p *FSPkg) Check() (errs []error) {
	return p.Pkg.Check()
}

// CompareFSPkgs returns an integer comparing two [FSPkg] instances according to their paths, and their
// respective [FSPkgTree]s' versions. The result will be 0 if the p and q have the same paths and
// versions; -1 if r has a path which alphabetically comes before the path of s, or if the paths are
// the same but r has a lower version than s; or +1 if r has a path which alphabetically comes after
// the path of s, or if the paths are the same but r has a higher version than s.
func CompareFSPkgs(p, q *FSPkg) int {
	if result := ComparePaths(p.Path(), q.Path()); result != CompareEQ {
		return result
	}
	if result := semver.Compare(p.FSPkgTree.Version, q.FSPkgTree.Version); result != CompareEQ {
		return result
	}
	return CompareEQ
}

// Pkg

// Path returns the package path of the Pkg instance.
func (p Pkg) Path() string {
	return path.Join(p.ParentPath, p.Subdir)
}

// Check looks for errors in the construction of the package.
func (p Pkg) Check() (errs []error) {
	// TODO: implement a check method on PkgDecl
	// errs = append(errs, ErrsWrap(p.Decl.Check(), "invalid package declaration")...)
	return errs
}
