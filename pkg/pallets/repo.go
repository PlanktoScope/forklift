package pallets

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// Paths

// SplitRepoPathSubdir splits paths of form github.com/user-name/git-repo-name/pallets-repo-subdir
// into github.com/user-name/git-repo-name and pallets-repo-subdir.
func SplitRepoPathSubdir(repoPath string) (vcsRepoPath, repoSubdir string, err error) {
	const sep = "/"
	pathParts := strings.Split(repoPath, sep)
	if pathParts[0] != "github.com" {
		return "", "", errors.Errorf(
			"pallet repository %s does not begin with github.com, but support for non-GitHub "+
				"repositories is not yet implemented",
			repoPath,
		)
	}
	const splitIndex = 3
	if len(pathParts) < splitIndex {
		return "", "", errors.Errorf(
			"pallet repository %s does not appear to be within a GitHub Git repository", repoPath,
		)
	}
	return strings.Join(pathParts[0:splitIndex], sep), strings.Join(pathParts[splitIndex:], sep), nil
}

// FSRepo

// LoadFSRepo loads a FSRepo from the specified directory path in the provided base filesystem.
// In the loaded FSRepo's embedded [Repo], the VCS repository path and Pallet repository
// subdirectory are initialized from the Pallet repository path declared in the repository's
// configuration file, while the Pallet repository version is *not* initialized.
func LoadFSRepo(fsys PathedFS, subdirPath string) (r *FSRepo, err error) {
	r = &FSRepo{}
	if r.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if r.Repo.Def, err = LoadRepoDef(r.FS, RepoDefFile); err != nil {
		return nil, errors.Wrapf(err, "couldn't load repo config")
	}
	if r.VCSRepoPath, r.Subdir, err = SplitRepoPathSubdir(r.Def.Repository.Path); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't parse path of Pallet repo %s", r.Def.Repository.Path,
		)
	}
	return r, nil
}

// LoadFSRepoContaining loads the FSRepo containing the specified sub-directory path in the provided
// base filesystem.
// The sub-directory path does not have to actually exist.
// In the loaded FSRepo's embedded [Repo], the VCS repository path and Pallet repository
// subdirectory are initialized from the Pallet repository path declared in the repository's
// configuration file, while the Pallet repository version is *not* initialized.
func LoadFSRepoContaining(fsys PathedFS, subdirPath string) (*FSRepo, error) {
	repoCandidatePath := subdirPath
	for {
		if repo, err := LoadFSRepo(fsys, repoCandidatePath); err == nil {
			return repo, nil
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
		if repoCandidatePath == "/" || repoCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no repository config file found in any parent directory of %s", subdirPath,
			)
		}
	}
}

// LoadFSRepos loads all FSRepos from the provided base filesystem matching the specified search
// pattern, modifying each FSRepo with the the optional processor function if a non-nil function is
// provided. The search pattern should be a [doublestar] pattern, such as `**`, matching repo
// directories to search for.
// With a nil processor function, in the embedded [Repo] of each loaded FSRepo, the VCS repository
// path and Pallet repository subdirectory are initialized from the Pallet repository path declared
// in the repository's configuration file, while the Pallet repository version is not initialized.
// After the processor is applied to each repository, all repositories are checked to enforce that
// multiple copies of the same repository with the same version are not allowed to be in the
// provided filesystem.
func LoadFSRepos(
	fsys PathedFS, searchPattern string, processor func(repo *FSRepo) error,
) ([]*FSRepo, error) {
	searchPattern = filepath.Join(searchPattern, RepoDefFile)
	repoDefFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for repo config files matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	orderedRepos := make([]*FSRepo, 0, len(repoDefFiles))
	repos := make(map[string]*FSRepo)
	for _, repoDefFilePath := range repoDefFiles {
		if filepath.Base(repoDefFilePath) != RepoDefFile {
			continue
		}
		repo, err := LoadFSRepo(fsys, filepath.Dir(repoDefFilePath))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load repo from %s/%s", fsys.Path(), repoDefFilePath,
			)
		}
		if processor != nil {
			if err = processor(repo); err != nil {
				return nil, errors.Wrap(err, "couldn't run processors on loaded repo")
			}
		}

		repoPath := repo.Def.Repository.Path
		if prevRepo, ok := repos[repoPath]; ok {
			if prevRepo.FromSameVCSRepo(repo.Repo) && prevRepo.Version == repo.Version &&
				prevRepo.FS.Path() == repo.FS.Path() {
				return nil, errors.Errorf(
					"the same version of repository %s was found in multiple different locations: %s, %s",
					repoPath, prevRepo.FS.Path(), repo.FS.Path(),
				)
			}
		}
		orderedRepos = append(orderedRepos, repo)
		repos[repoPath] = repo
	}

	return orderedRepos, nil
}

// LoadFSPkg loads a package at the specified filesystem path from the FSRepo instance
// The loaded package is fully initialized.
func (r *FSRepo) LoadFSPkg(pkgSubdir string) (pkg *FSPkg, err error) {
	if pkg, err = LoadFSPkg(r.FS, pkgSubdir); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load package %s from repo %s", pkgSubdir, r.Path(),
		)
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

// Repo

// Path returns the Pallet repository path of the Repo instance.
func (r Repo) Path() string {
	return filepath.Join(r.VCSRepoPath, r.Subdir)
}

// FromSameVCSRepo determines whether the candidate Pallet repository is provided by the same VCS
// repo as the Repo instance.
func (r Repo) FromSameVCSRepo(candidate Repo) bool {
	return r.VCSRepoPath == candidate.VCSRepoPath && r.Version == candidate.Version
}

// Check looks for errors in the construction of the repository.
func (r Repo) Check() (errs []error) {
	if r.Path() != r.Def.Repository.Path {
		errs = append(errs, errors.Errorf(
			"repo path %s is inconsistent with path %s specified in repo configuration",
			r.Path(), r.Def.Repository.Path,
		))
	}
	errs = append(errs, ErrsWrap(r.Def.Check(), "invalid repo config")...)
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
			err, "couldn't read repo config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := RepoDef{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return RepoDef{}, errors.Wrap(err, "couldn't parse repo config")
	}
	return config, nil
}

// Check looks for errors in the construction of the repository configuration.
func (c RepoDef) Check() (errs []error) {
	return ErrsWrap(c.Repository.Check(), "invalid repo spec")
}

// RepoSpec

// Check looks for errors in the construction of the repo spec.
func (s RepoSpec) Check() (errs []error) {
	if s.Path == "" {
		errs = append(errs, errors.Errorf("repo spec is missing `path` parameter"))
	}
	return errs
}
