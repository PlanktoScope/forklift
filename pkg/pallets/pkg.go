package pallets

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// FSPkg

// LoadFSPkg loads a FSPkg from the specified directory path in the provided base filesystem.
// In the loaded FSPkg's embedded [Pkg], the Pallet repository path is not initialized, nor is the
// Pallet package subdirectory initialized, nor is the pointer to the Pallet repository initialized.
func LoadFSPkg(fsys PathedFS, subdirPath string) (p *FSPkg, err error) {
	p = &FSPkg{}
	if p.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if p.Pkg.Config, err = LoadPkgConfig(p.FS, PkgSpecFile); err != nil {
		return nil, errors.Wrapf(err, "couldn't load package config")
	}
	return p, nil
}

// LoadFSPkgs loads all FSPkgs from the provided base filesystem matching the specified search
// pattern, modifying each FSPkg with the the optional processor function if a non-nil function is
// provided. The search pattern should be a [doublestar] pattern, such as `**`, matching package
// directories to search for.
// The Pallet repository path, and the Pallet package subdirectory, and the pointer to the Pallet
// repository are all left uninitialized.
func LoadFSPkgs(fsys PathedFS, searchPattern string) ([]*FSPkg, error) {
	searchPattern = filepath.Join(searchPattern, PkgSpecFile)
	pkgConfigFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for package configs matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	pkgs := make([]*FSPkg, 0, len(pkgConfigFiles))
	for _, pkgConfigFilePath := range pkgConfigFiles {
		if filepath.Base(pkgConfigFilePath) != PkgSpecFile {
			continue
		}

		pkg, err := LoadFSPkg(fsys, filepath.Dir(pkgConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load package from %s", pkgConfigFilePath)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func (p *FSPkg) AttachFSRepo(repo *FSRepo) error {
	p.RepoPath = repo.Config.Repository.Path
	if !strings.HasPrefix(p.FS.Path(), fmt.Sprintf("%s/", repo.FS.Path())) {
		return errors.Errorf(
			"package at %s is not within the scope of repo %s at %s",
			p.FS.Path(), repo.Path(), repo.FS.Path(),
		)
	}
	p.Subdir = strings.TrimPrefix(p.FS.Path(), fmt.Sprintf("%s/", repo.FS.Path()))
	p.Repo = repo
	return nil
}

// Check looks for errors in the construction of the package.
func (p *FSPkg) Check() (errs []error) {
	if p.Repo != nil {
		if p.Pkg.Repo != &p.Repo.Repo {
			errs = append(errs, errors.New(
				"inconsistent pointers to the repository between the package as a FSPkg and the package "+
					"as a Pkg",
			))
		}
	}
	errs = append(errs, p.Pkg.Check()...)
	return errs
}

func CompareFSPkgs(p, q *FSPkg) int {
	repoPathComparison := CompareRepoPaths(p.Repo.Repo, q.Repo.Repo)
	if repoPathComparison != CompareEQ {
		return repoPathComparison
	}
	if p.Subdir != q.Subdir {
		if p.Subdir < q.Subdir {
			return CompareLT
		}
		return CompareGT
	}
	repoVersionComparison := semver.Compare(p.Repo.Version, q.Repo.Version)
	if repoVersionComparison != CompareEQ {
		return repoVersionComparison
	}
	return CompareEQ
}

// Pkg

// Path returns the Pallet package path of the Pkg instance.
func (p Pkg) Path() string {
	return filepath.Join(p.RepoPath, p.Subdir)
}

// Check looks for errors in the construction of the package.
func (p Pkg) Check() (errs []error) {
	// TODO: implement a check method on PkgConfig
	// errs = append(errs, ErrsWrap(p.Config.Check(), "invalid package config")...)
	if p.Repo != nil && p.RepoPath != p.Repo.Path() {
		errs = append(errs, errors.Errorf(
			"repo path %s of package is inconsistent with path %s of attached repo",
			p.RepoPath, p.Repo.Path(),
		))
	}
	return errs
}

// PkgConfig

// LoadPkgConfig loads a PkgConfig from the specified file path in the provided base filesystem.
func LoadPkgConfig(fsys PathedFS, filePath string) (PkgConfig, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return PkgConfig{}, errors.Wrapf(
			err, "couldn't read package config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := PkgConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return PkgConfig{}, errors.Wrap(err, "couldn't parse package config")
	}
	return config, nil
}

// PkgHostSpec

// ResourceAttachmentSource returns the source path for resources under the PkgHostSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgHostSpec instance.
func (s PkgHostSpec) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, "host specification")
}

// PkgDeplSpec

// ResourceAttachmentSource returns the source path for resources under the PkgDeplSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgDeplSpec instance.
func (s PkgDeplSpec) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, "deployment specification")
}

// DefinesStack determines whether the PkgDeplSpec instance defines a Docker stack to be deployed.
func (s PkgDeplSpec) DefinesStack() bool {
	return s.DefinitionFile != ""
}

// PkgFeatureSpec

// ResourceAttachmentSource returns the source path for resources under the PkgFeatureSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgFeatureSpec instance.
func (s PkgFeatureSpec) ResourceAttachmentSource(parentSource []string, featureName string) []string {
	return append(parentSource, fmt.Sprintf("feature %s", featureName))
}
