package pallets

import (
	"fmt"
	"io/fs"
	"path"
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
	if p.Pkg.Def, err = LoadPkgDef(p.FS, PkgDefFile); err != nil {
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
	searchPattern = path.Join(searchPattern, PkgDefFile)
	pkgDefFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for package configs matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	pkgs := make([]*FSPkg, 0, len(pkgDefFiles))
	for _, pkgDefFilePath := range pkgDefFiles {
		if path.Base(pkgDefFilePath) != PkgDefFile {
			continue
		}

		pkg, err := LoadFSPkg(fsys, path.Dir(pkgDefFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load package from %s", pkgDefFilePath)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

// AttachFSRepo updates the FSPkg instance's RepoPath, Subdir, Pkg.Repo, and Repo fields based on
// the provided repo.
func (p *FSPkg) AttachFSRepo(repo *FSRepo) error {
	p.RepoPath = repo.Def.Repository.Path
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
	return path.Join(p.RepoPath, p.Subdir)
}

// Check looks for errors in the construction of the package.
func (p Pkg) Check() (errs []error) {
	// TODO: implement a check method on PkgDef
	// errs = append(errs, ErrsWrap(p.Def.Check(), "invalid package config")...)
	if p.Repo != nil && p.RepoPath != p.Repo.Path() {
		errs = append(errs, errors.Errorf(
			"repo path %s of package is inconsistent with path %s of attached repo",
			p.RepoPath, p.Repo.Path(),
		))
	}
	return errs
}

// ResAttachmentSource returns the source path for resources under the Pkg instance.
// The resulting slice is useful for constructing [AttachedRes] instances.
func (p Pkg) ResAttachmentSource(parentSource []string) []string {
	return append(parentSource, fmt.Sprintf("package %s", p.Path()))
}

// ProvidedListeners returns a slice of all host port listeners provided by a deployment of the
// Pallet package with the specified features enabled.
func (p Pkg) ProvidedListeners(
	parentSource []string, enabledFeatures []string,
) (provided []AttachedRes[ListenerRes]) {
	parentSource = p.ResAttachmentSource(parentSource)
	provided = append(provided, p.Def.Host.Provides.AttachedListeners(
		p.Def.Host.ResAttachmentSource(parentSource),
	)...)
	provided = append(provided, p.Def.Deployment.Provides.AttachedListeners(
		p.Def.Deployment.ResAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Def.Features[featureName]
		provided = append(provided, feature.Provides.AttachedListeners(
			feature.ResAttachmentSource(parentSource, featureName),
		)...)
	}
	return provided
}

// RequiredNetworks returns a slice of all Docker networks required by a deployment of the Pallet
// package with the specified features enabled.
func (p Pkg) RequiredNetworks(
	parentSource []string, enabledFeatures []string,
) (required []AttachedRes[NetworkRes]) {
	parentSource = p.ResAttachmentSource(parentSource)
	required = append(required, p.Def.Deployment.Requires.AttachedNetworks(
		p.Def.Deployment.ResAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Def.Features[featureName]
		required = append(required, feature.Requires.AttachedNetworks(
			feature.ResAttachmentSource(parentSource, featureName),
		)...)
	}
	return required
}

// ProvidedNetworks returns a slice of all Docker networks provided by a deployment of the Pallet
// package with the specified features enabled.
func (p Pkg) ProvidedNetworks(
	parentSource []string, enabledFeatures []string,
) (provided []AttachedRes[NetworkRes]) {
	parentSource = p.ResAttachmentSource(parentSource)
	provided = append(provided, p.Def.Host.Provides.AttachedNetworks(
		p.Def.Host.ResAttachmentSource(parentSource),
	)...)
	provided = append(provided, p.Def.Deployment.Provides.AttachedNetworks(
		p.Def.Deployment.ResAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Def.Features[featureName]
		provided = append(provided, feature.Provides.AttachedNetworks(
			feature.ResAttachmentSource(parentSource, featureName),
		)...)
	}
	return provided
}

// RequiredServices returns a slice of all network services required by a deployment of the Pallet
// package with the specified features enabled.
func (p Pkg) RequiredServices(
	parentSource []string, enabledFeatures []string,
) (required []AttachedRes[ServiceRes]) {
	parentSource = p.ResAttachmentSource(parentSource)
	required = append(required, p.Def.Deployment.Requires.AttachedServices(
		p.Def.Deployment.ResAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Def.Features[featureName]
		required = append(required, feature.Requires.AttachedServices(
			feature.ResAttachmentSource(parentSource, featureName),
		)...)
	}
	return required
}

// ProvidedServices returns a slice of all network services provided by a deployment of the Pallet
// package with the specified features enabled.
func (p Pkg) ProvidedServices(
	parentSource []string, enabledFeatures []string,
) (provided []AttachedRes[ServiceRes]) {
	parentSource = p.ResAttachmentSource(parentSource)
	provided = append(provided, p.Def.Host.Provides.AttachedServices(
		p.Def.Host.ResAttachmentSource(parentSource),
	)...)
	provided = append(provided, p.Def.Deployment.Provides.AttachedServices(
		p.Def.Deployment.ResAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Def.Features[featureName]
		provided = append(provided, feature.Provides.AttachedServices(
			feature.ResAttachmentSource(parentSource, featureName),
		)...)
	}
	return provided
}

// PkgDef

// LoadPkgDef loads a PkgDef from the specified file path in the provided base filesystem.
func LoadPkgDef(fsys PathedFS, filePath string) (PkgDef, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return PkgDef{}, errors.Wrapf(
			err, "couldn't read package config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := PkgDef{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return PkgDef{}, errors.Wrap(err, "couldn't parse package config")
	}
	return config, nil
}

// PkgHostSpec

// ResAttachmentSource returns the source path for resources under the PkgHostSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgHostSpec instance.
// The resulting slice is useful for constructing [AttachedRes] instances.
func (s PkgHostSpec) ResAttachmentSource(parentSource []string) []string {
	return append(parentSource, "host specification")
}

// PkgDeplSpec

// ResAttachmentSource returns the source path for resources under the PkgDeplSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgDeplSpec instance.
// The resulting slice is useful for constructing [AttachedRes] instances.
func (s PkgDeplSpec) ResAttachmentSource(parentSource []string) []string {
	return append(parentSource, "deployment specification")
}

// DefinesStack determines whether the PkgDeplSpec instance defines a Docker stack to be deployed.
func (s PkgDeplSpec) DefinesStack() bool {
	return s.DefinitionFile != ""
}

// PkgFeatureSpec

// ResAttachmentSource returns the source path for resources under the PkgFeatureSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgFeatureSpec instance.
// The resulting slice is useful for constructing [AttachedRes] instances.
func (s PkgFeatureSpec) ResAttachmentSource(parentSource []string, featureName string) []string {
	return append(parentSource, fmt.Sprintf("feature %s", featureName))
}
