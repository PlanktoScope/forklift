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

// AttachFSRepo updates the FSPkg instance's RepoPath, Subdir, Pkg.Repo, and Repo fields based on
// the provided repo.
func (p *FSPkg) AttachFSRepo(repo *FSRepo) error {
	p.RepoPath = repo.Config.Repository.Path
	if !strings.HasPrefix(p.FS.Path(), fmt.Sprintf("%s/", repo.FS.Path())) {
		return errors.Errorf(
			"package at %s is not within the scope of repo %s at %s",
			p.FS.Path(), repo.Path(), repo.FS.Path(),
		)
	}
	p.Subdir = strings.TrimPrefix(p.FS.Path(), fmt.Sprintf("%s/", repo.FS.Path()))
	p.Pkg.Repo = &repo.Repo
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

// ComparePkgs returns an integer comparing two [Pkg] instances according to their paths, and their
// respective repositories' versions. The result will be 0 if the p and q have the same paths and
// versions; -1 if r has a path which alphabetically comes before the path of s, or if the paths are
// the same but r has a lower version than s; or +1 if r has a path which alphabetically comes after
// the path of s, or if the paths are the same but r has a lower version than s.
func ComparePkgs(p, q Pkg) int {
	if result := ComparePaths(p.Path(), q.Path()); result != CompareEQ {
		return result
	}
	if result := semver.Compare(p.Repo.Version, q.Repo.Version); result != CompareEQ {
		return result
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

// ResourceAttachmentSource returns the source path for resources under the Pkg instance.
// The resulting slice is useful for constructing [AttachedResource] instances.
func (p Pkg) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, fmt.Sprintf("package %s", p.Path()))
}

// ProvidedListeners returns a slice of all host port listeners provided by a deployment of the
// Pallet package with the specified features enabled.
func (p Pkg) ProvidedListeners(
	parentSource []string, enabledFeatures []string,
) (provided []AttachedResource[ListenerResource]) {
	parentSource = p.ResourceAttachmentSource(parentSource)
	provided = append(provided, p.Config.Host.Provides.AttachedListeners(
		p.Config.Host.ResourceAttachmentSource(parentSource),
	)...)
	provided = append(provided, p.Config.Deployment.Provides.AttachedListeners(
		p.Config.Deployment.ResourceAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Config.Features[featureName]
		provided = append(provided, feature.Provides.AttachedListeners(
			feature.ResourceAttachmentSource(parentSource, featureName),
		)...)
	}
	return provided
}

// RequiredNetworks returns a slice of all Docker networks required by a deployment of the Pallet
// package with the specified features enabled.
func (p Pkg) RequiredNetworks(
	parentSource []string, enabledFeatures []string,
) (required []AttachedResource[NetworkResource]) {
	parentSource = p.ResourceAttachmentSource(parentSource)
	required = append(required, p.Config.Deployment.Requires.AttachedNetworks(
		p.Config.Deployment.ResourceAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Config.Features[featureName]
		required = append(required, feature.Requires.AttachedNetworks(
			feature.ResourceAttachmentSource(parentSource, featureName),
		)...)
	}
	return required
}

// ProvidedNetworks returns a slice of all Docker networks provided by a deployment of the Pallet
// package with the specified features enabled.
func (p Pkg) ProvidedNetworks(
	parentSource []string, enabledFeatures []string,
) (provided []AttachedResource[NetworkResource]) {
	parentSource = p.ResourceAttachmentSource(parentSource)
	provided = append(provided, p.Config.Host.Provides.AttachedNetworks(
		p.Config.Host.ResourceAttachmentSource(parentSource),
	)...)
	provided = append(provided, p.Config.Deployment.Provides.AttachedNetworks(
		p.Config.Deployment.ResourceAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Config.Features[featureName]
		provided = append(provided, feature.Provides.AttachedNetworks(
			feature.ResourceAttachmentSource(parentSource, featureName),
		)...)
	}
	return provided
}

// RequiredServices returns a slice of all network services required by a deployment of the Pallet
// package with the specified features enabled.
func (p Pkg) RequiredServices(
	parentSource []string, enabledFeatures []string,
) (required []AttachedResource[ServiceResource]) {
	parentSource = p.ResourceAttachmentSource(parentSource)
	required = append(required, p.Config.Deployment.Requires.AttachedServices(
		p.Config.Deployment.ResourceAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Config.Features[featureName]
		required = append(required, feature.Requires.AttachedServices(
			feature.ResourceAttachmentSource(parentSource, featureName),
		)...)
	}
	return required
}

// ProvidedServices returns a slice of all network services provided by a deployment of the Pallet
// package with the specified features enabled.
func (p Pkg) ProvidedServices(
	parentSource []string, enabledFeatures []string,
) (provided []AttachedResource[ServiceResource]) {
	parentSource = p.ResourceAttachmentSource(parentSource)
	provided = append(provided, p.Config.Host.Provides.AttachedServices(
		p.Config.Host.ResourceAttachmentSource(parentSource),
	)...)
	provided = append(provided, p.Config.Deployment.Provides.AttachedServices(
		p.Config.Deployment.ResourceAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Config.Features[featureName]
		provided = append(provided, feature.Provides.AttachedServices(
			feature.ResourceAttachmentSource(parentSource, featureName),
		)...)
	}
	return provided
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
// The resulting slice is useful for constructing [AttachedResource] instances.
func (s PkgHostSpec) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, "host specification")
}

// PkgDeplSpec

// ResourceAttachmentSource returns the source path for resources under the PkgDeplSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgDeplSpec instance.
// The resulting slice is useful for constructing [AttachedResource] instances.
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
// The resulting slice is useful for constructing [AttachedResource] instances.
func (s PkgFeatureSpec) ResourceAttachmentSource(parentSource []string, featureName string) []string {
	return append(parentSource, fmt.Sprintf("feature %s", featureName))
}
