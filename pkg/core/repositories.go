package core

import (
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/bmatcuk/doublestar/v4"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// FSPkgTree

// LoadFSPkgTree loads a FSPkgTree from the specified directory path in the provided base filesystem.
// In the loaded FSPkgTree's embedded [PkgTree], the version is *not* initialized.
func LoadFSPkgTree(fsys ffs.PathedFS, subdirPath string) (r *FSPkgTree, err error) {
	r = &FSPkgTree{}
	if r.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	return r, nil
}

// LoadFSPkgTreeContaining loads the FSPkgTree containing the specified sub-directory path in the
// provided base filesystem.
// The sub-directory path does not have to actually exist.
// In the loaded FSPkgTree's embedded [PkgTree], the version is *not* initialized.
func LoadFSPkgTreeContaining(fsys ffs.PathedFS, subdirPath string) (*FSPkgTree, error) {
	repoCandidatePath := subdirPath
	for {
		if repo, err := LoadFSPkgTree(fsys, repoCandidatePath); err == nil {
			return repo, nil
		}
		repoCandidatePath = path.Dir(repoCandidatePath)
		if repoCandidatePath == "/" || repoCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no repo declaration file was found in any parent directory of %s", subdirPath,
			)
		}
	}
}

// LoadFSPkgTrees loads all FSPkgTrees from the provided base filesystem matching the specified search
// pattern. The search pattern should be a [doublestar] pattern, such as `**`, matching repo
// directories to search for.
// In the embedded [PkgTree] of each loaded FSPkgTree, the version is *not* initialized.
func LoadFSPkgTrees(fsys ffs.PathedFS, searchPattern string) ([]*FSPkgTree, error) {
	searchPattern = path.Join(searchPattern, PkgTreeDeclFile)
	repoDeclFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for repo declaration files matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	orderedPkgTrees := make([]*FSPkgTree, 0, len(repoDeclFiles))
	repos := make(map[string]*FSPkgTree)
	for _, repoDeclFilePath := range repoDeclFiles {
		if path.Base(repoDeclFilePath) != PkgTreeDeclFile {
			continue
		}
		repo, err := LoadFSPkgTree(fsys, path.Dir(repoDeclFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load repo from %s/%s", fsys.Path(), repoDeclFilePath)
		}

		orderedPkgTrees = append(orderedPkgTrees, repo)
		repos[repo.Path()] = repo
	}

	return orderedPkgTrees, nil
}

// LoadFSPkg loads a package at the specified filesystem path from the FSPkgTree instance
// The loaded package is fully initialized.
func (r *FSPkgTree) LoadFSPkg(pkgSubdir string) (pkg *FSPkg, err error) {
	if pkg, err = LoadFSPkg(r.FS, pkgSubdir); err != nil {
		return nil, errors.Wrapf(err, "couldn't load package %s from repo %s", pkgSubdir, r.Path())
	}
	if err = pkg.AttachFSPkgTree(r); err != nil {
		return nil, errors.Wrap(err, "couldn't attach repo to package")
	}
	return pkg, nil
}

// LoadFSPkgs loads all packages in the FSPkgTree instance.
// The loaded packages are fully initialized.
func (r *FSPkgTree) LoadFSPkgs(searchPattern string) ([]*FSPkg, error) {
	pkgs, err := LoadFSPkgs(r.FS, searchPattern)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		if err = pkg.AttachFSPkgTree(r); err != nil {
			return nil, errors.Wrap(err, "couldn't attach repo to package")
		}
	}
	return pkgs, nil
}

// LoadReadme loads the readme file defined by the repo.
func (r *FSPkgTree) LoadReadme() ([]byte, error) {
	readmePath := r.Decl.PkgTree.ReadmeFile
	bytes, err := fs.ReadFile(r.FS, readmePath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't read repo readme %s/%s", r.FS.Path(), readmePath)
	}
	return bytes, nil
}

// PkgTree

// Path returns the repo path of the PkgTree instance.
func (r PkgTree) Path() string {
	return r.Decl.PkgTree.Path
}

// VersionQuery represents the PkgTree instance as a version query.
func (r PkgTree) VersionQuery() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.Version)
}

// Check looks for errors in the construction of the repo.
func (r PkgTree) Check() (errs []error) {
	errs = append(errs, ErrsWrap(r.Decl.Check(), "invalid repo declaration")...)
	return errs
}

// The result of comparison functions is one of these values.
const (
	CompareLT = -1
	CompareEQ = 0
	CompareGT = 1
)

// ComparePkgTrees returns an integer comparing two [PkgTree] instances according to their paths and
// versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func ComparePkgTrees(r, s PkgTree) int {
	if result := ComparePaths(r.Path(), s.Path()); result != CompareEQ {
		return result
	}
	if result := semver.Compare(r.Version, s.Version); result != CompareEQ {
		return result
	}
	return CompareEQ
}

// ComparePaths returns an integer comparing two paths. The result will be 0 if the r and s are
// the same; -1 if r alphabetically comes before s; or +1 if r alphabetically comes after s.
func ComparePaths(r, s string) int {
	if r < s {
		return CompareLT
	}
	if r > s {
		return CompareGT
	}
	return CompareEQ
}

// PkgTreeDecl

// LoadPkgTreeDecl loads a PkgTreeDecl from the specified file path in the provided base filesystem.
func LoadPkgTreeDecl(fsys ffs.PathedFS, filePath string) (PkgTreeDecl, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return PkgTreeDecl{}, errors.Wrapf(
			err, "couldn't read repo declaration file %s/%s", fsys.Path(), filePath,
		)
	}
	declaration := PkgTreeDecl{}
	if err = yaml.Unmarshal(bytes, &declaration); err != nil {
		return PkgTreeDecl{}, errors.Wrap(err, "couldn't parse repo declaration")
	}
	return declaration, nil
}

// Check looks for errors in the construction of the repo declaration.
func (d PkgTreeDecl) Check() (errs []error) {
	return ErrsWrap(d.PkgTree.Check(), "invalid repo spec")
}

// WritePkgTreeDecl creates a repo definition file at the specified path.
func WritePkgTreeDecl(repoDecl PkgTreeDecl, outputPath string) error {
	marshaled, err := yaml.Marshal(repoDecl)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal bundled repo declaration")
	}
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(outputPath, marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save repo declaration to %s", outputPath)
	}
	return nil
}

// PkgTreeSpec

// Check looks for errors in the construction of the repo spec.
func (s PkgTreeSpec) Check() (errs []error) {
	if s.Path == "" {
		errs = append(errs, errors.Errorf("repo spec is missing `path` parameter"))
	}
	return errs
}
