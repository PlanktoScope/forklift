package core

import (
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// FSRepo

// LoadFSRepo loads a FSRepo from the specified directory path in the provided base filesystem.
// In the loaded FSRepo's embedded [Repo], the version is *not* initialized.
func LoadFSRepo(fsys PathedFS, subdirPath string) (r *FSRepo, err error) {
	r = &FSRepo{}
	if r.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if r.Repo.Def, err = LoadRepoDef(r.FS, RepoDefFile); err != nil {
		return nil, errors.Wrapf(err, "couldn't load repo declaration")
	}
	return r, nil
}

// LoadFSRepoContaining loads the FSRepo containing the specified sub-directory path in the
// provided base filesystem.
// The sub-directory path does not have to actually exist.
// In the loaded FSRepo's embedded [Repo], the version is *not* initialized.
func LoadFSRepoContaining(fsys PathedFS, subdirPath string) (*FSRepo, error) {
	repoCandidatePath := subdirPath
	for {
		if repo, err := LoadFSRepo(fsys, repoCandidatePath); err == nil {
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

// LoadFSRepos loads all FSRepos from the provided base filesystem matching the specified search
// pattern. The search pattern should be a [doublestar] pattern, such as `**`, matching repo
// directories to search for.
// In the embedded [Repo] of each loaded FSRepo, the version is *not* initialized.
func LoadFSRepos(fsys PathedFS, searchPattern string) ([]*FSRepo, error) {
	searchPattern = path.Join(searchPattern, RepoDefFile)
	repoDefFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for repo declaration files matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	orderedRepos := make([]*FSRepo, 0, len(repoDefFiles))
	repos := make(map[string]*FSRepo)
	for _, repoDefFilePath := range repoDefFiles {
		if path.Base(repoDefFilePath) != RepoDefFile {
			continue
		}
		repo, err := LoadFSRepo(fsys, path.Dir(repoDefFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load repo from %s/%s", fsys.Path(), repoDefFilePath)
		}

		orderedRepos = append(orderedRepos, repo)
		repos[repo.Path()] = repo
	}

	return orderedRepos, nil
}

// LoadFSPkg loads a package at the specified filesystem path from the FSRepo instance
// The loaded package is fully initialized.
func (r *FSRepo) LoadFSPkg(pkgSubdir string) (pkg *FSPkg, err error) {
	if pkg, err = LoadFSPkg(r.FS, pkgSubdir); err != nil {
		return nil, errors.Wrapf(err, "couldn't load package %s from repo %s", pkgSubdir, r.Path())
	}
	if err = pkg.AttachFSRepo(r); err != nil {
		return nil, errors.Wrap(err, "couldn't attach repo to package")
	}
	return pkg, nil
}

// LoadFSPkgs loads all packages in the FSRepo instance.
// The loaded packages are fully initialized.
func (r *FSRepo) LoadFSPkgs(searchPattern string) ([]*FSPkg, error) {
	pkgs, err := LoadFSPkgs(r.FS, searchPattern)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		if err = pkg.AttachFSRepo(r); err != nil {
			return nil, errors.Wrap(err, "couldn't attach repo to package")
		}
	}
	return pkgs, nil
}

// LoadReadme loads the readme file defined by the repo.
func (r *FSRepo) LoadReadme() ([]byte, error) {
	readmePath := r.Def.Repo.ReadmeFile
	bytes, err := fs.ReadFile(r.FS, readmePath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't read repo readme %s/%s", r.FS.Path(), readmePath)
	}
	return bytes, nil
}

// Repo

// Path returns the repo path of the Repo instance.
func (r Repo) Path() string {
	return r.Def.Repo.Path
}

// VersionQuery represents the Repo instance as a version query.
func (r Repo) VersionQuery() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.Version)
}

// Check looks for errors in the construction of the repo.
func (r Repo) Check() (errs []error) {
	errs = append(errs, ErrsWrap(r.Def.Check(), "invalid repo declaration")...)
	return errs
}

// The result of comparison functions is one of these values.
const (
	CompareLT = -1
	CompareEQ = 0
	CompareGT = 1
)

// CompareRepos returns an integer comparing two [Repo] instances according to their paths and
// versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func CompareRepos(r, s Repo) int {
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

// RepoDef

// LoadRepoDef loads a RepoDef from the specified file path in the provided base filesystem.
func LoadRepoDef(fsys PathedFS, filePath string) (RepoDef, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return RepoDef{}, errors.Wrapf(
			err, "couldn't read repo declaration file %s/%s", fsys.Path(), filePath,
		)
	}
	declaration := RepoDef{}
	if err = yaml.Unmarshal(bytes, &declaration); err != nil {
		return RepoDef{}, errors.Wrap(err, "couldn't parse repo declaration")
	}
	return declaration, nil
}

// Check looks for errors in the construction of the repo declaration.
func (d RepoDef) Check() (errs []error) {
	return ErrsWrap(d.Repo.Check(), "invalid repo spec")
}

// WriteRepoDef creates a repo definition file at the specified path.
func WriteRepoDef(repoDef RepoDef, outputPath string) error {
	marshaled, err := yaml.Marshal(repoDef)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal bundled repo declaration")
	}
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(outputPath, marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save repo declaration to %s", outputPath)
	}
	return nil
}

// RepoSpec

// Check looks for errors in the construction of the repo spec.
func (s RepoSpec) Check() (errs []error) {
	if s.Path == "" {
		errs = append(errs, errors.Errorf("repo spec is missing `path` parameter"))
	}
	return errs
}
