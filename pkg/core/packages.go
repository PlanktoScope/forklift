package core

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	res "github.com/forklift-run/forklift/pkg/resources"
)

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
// The repo path, and the package subdirectory, and the pointer to the repo are all left
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

// AttachFSPkgTree updates the FSPkg instance's PkgTreePath, Subdir, Pkg.PkgTree, and PkgTree fields
// based on the provided repo.
func (p *FSPkg) AttachFSPkgTree(repo *FSPkgTree) error {
	p.PkgTreePath = repo.Decl.PkgTree.Path
	if !strings.HasPrefix(p.FS.Path(), fmt.Sprintf("%s/", repo.FS.Path())) {
		return errors.Errorf(
			"package at %s is not within the scope of repo %s at %s",
			p.FS.Path(), repo.Path(), repo.FS.Path(),
		)
	}
	p.Subdir = strings.TrimPrefix(p.FS.Path(), fmt.Sprintf("%s/", repo.FS.Path()))
	p.Pkg.PkgTree = &repo.PkgTree
	p.PkgTree = repo
	return nil
}

// Check looks for errors in the construction of the package.
func (p *FSPkg) Check() (errs []error) {
	if p.PkgTree != nil {
		if p.Pkg.PkgTree != &p.PkgTree.PkgTree {
			errs = append(errs, errors.New(
				"inconsistent pointers to the repo between the package as a FSPkg and the package as a Pkg",
			))
		}
	}
	errs = append(errs, p.Pkg.Check()...)
	return errs
}

// ComparePkgs returns an integer comparing two [Pkg] instances according to their paths, and their
// respective repos' versions. The result will be 0 if the p and q have the same paths and
// versions; -1 if r has a path which alphabetically comes before the path of s, or if the paths are
// the same but r has a lower version than s; or +1 if r has a path which alphabetically comes after
// the path of s, or if the paths are the same but r has a higher version than s.
func ComparePkgs(p, q Pkg) int {
	if result := ComparePaths(p.Path(), q.Path()); result != CompareEQ {
		return result
	}
	if result := semver.Compare(p.PkgTree.Version, q.PkgTree.Version); result != CompareEQ {
		return result
	}
	return CompareEQ
}

// Pkg

// Path returns the package path of the Pkg instance.
func (p Pkg) Path() string {
	return path.Join(p.PkgTreePath, p.Subdir)
}

// Check looks for errors in the construction of the package.
func (p Pkg) Check() (errs []error) {
	// TODO: implement a check method on PkgDecl
	// errs = append(errs, ErrsWrap(p.Decl.Check(), "invalid package declaration")...)
	if p.PkgTree != nil && p.PkgTreePath != p.PkgTree.Path() {
		errs = append(errs, errors.Errorf(
			"repo path %s of package is inconsistent with path %s of attached repo",
			p.PkgTreePath, p.PkgTree.Path(),
		))
	}
	return errs
}

// ResAttachmentSource returns the source path for resources under the Pkg instance.
// The resulting slice is useful for constructing [res.Attached] instances.
func (p Pkg) ResAttachmentSource(parentSource []string) []string {
	return append(parentSource, fmt.Sprintf("package %s", p.Path()))
}

// ProvidedListeners returns a slice of all host port listeners provided by a deployment of the
// package with the specified features enabled.
func (p Pkg) ProvidedListeners(
	parentSource []string, enabledFeatures []string,
) (provided []res.Attached[ListenerRes, []string]) {
	return providedResources(
		p, parentSource, enabledFeatures, func(res ProvidedRes) attachedResGetter[ListenerRes] {
			return res.AttachedListeners
		},
	)
}

type (
	attachedResGetter[Resource any] func(source []string) []res.Attached[Resource, []string]
	providedResGetter[Resource any] func(res ProvidedRes) attachedResGetter[Resource]
)

func providedResources[Resource any](
	p Pkg, parentSource []string, enabledFeatures []string, getter providedResGetter[Resource],
) (provided []res.Attached[Resource, []string]) {
	parentSource = p.ResAttachmentSource(parentSource)
	provided = append(provided, getter(p.Decl.Host.Provides)(
		p.Decl.Host.ResAttachmentSource(parentSource),
	)...)
	provided = append(provided, getter(p.Decl.Deployment.Provides)(
		p.Decl.Deployment.ResAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Decl.Features[featureName]
		provided = append(provided, getter(feature.Provides)(
			feature.ResAttachmentSource(parentSource, featureName),
		)...)
	}
	return provided
}

// RequiredNetworks returns a slice of all Docker networks required by a deployment of the package
// with the specified features enabled.
func (p Pkg) RequiredNetworks(
	parentSource []string, enabledFeatures []string,
) (required []res.Attached[NetworkRes, []string]) {
	return requiredResources(
		p, parentSource, enabledFeatures, func(res RequiredRes) attachedResGetter[NetworkRes] {
			return res.AttachedNetworks
		},
	)
}

type requiredResGetter[Resource any] func(res RequiredRes) attachedResGetter[Resource]

func requiredResources[Resource any](
	p Pkg, parentSource []string, enabledFeatures []string, getter requiredResGetter[Resource],
) (required []res.Attached[Resource, []string]) {
	parentSource = p.ResAttachmentSource(parentSource)
	required = append(required, getter(p.Decl.Deployment.Requires)(
		p.Decl.Deployment.ResAttachmentSource(parentSource),
	)...)

	for _, featureName := range enabledFeatures {
		feature := p.Decl.Features[featureName]
		required = append(required, getter(feature.Requires)(
			feature.ResAttachmentSource(parentSource, featureName),
		)...)
	}
	return required
}

// ProvidedNetworks returns a slice of all Docker networks provided by a deployment of the package
// with the specified features enabled.
func (p Pkg) ProvidedNetworks(
	parentSource []string, enabledFeatures []string,
) (provided []res.Attached[NetworkRes, []string]) {
	return providedResources(
		p, parentSource, enabledFeatures, func(res ProvidedRes) attachedResGetter[NetworkRes] {
			return res.AttachedNetworks
		},
	)
}

// RequiredServices returns a slice of all network services required by a deployment of the package
// with the specified features enabled.
func (p Pkg) RequiredServices(
	parentSource []string, enabledFeatures []string,
) (required []res.Attached[ServiceRes, []string]) {
	return requiredResources(
		p, parentSource, enabledFeatures, func(res RequiredRes) attachedResGetter[ServiceRes] {
			return res.AttachedServices
		},
	)
}

// ProvidedServices returns a slice of all network services provided by a deployment of the package
// with the specified features enabled.
func (p Pkg) ProvidedServices(
	parentSource []string, enabledFeatures []string,
) (provided []res.Attached[ServiceRes, []string]) {
	return providedResources(
		p, parentSource, enabledFeatures, func(res ProvidedRes) attachedResGetter[ServiceRes] {
			return res.AttachedServices
		},
	)
}

// RequiredFilesets returns a slice of all filesets required by a deployment of the package
// with the specified features enabled.
func (p Pkg) RequiredFilesets(
	parentSource []string, enabledFeatures []string,
) (required []res.Attached[FilesetRes, []string]) {
	return requiredResources(
		p, parentSource, enabledFeatures, func(res RequiredRes) attachedResGetter[FilesetRes] {
			return res.AttachedFilesets
		},
	)
}

// ProvidedFilesets returns a slice of all filesets provided by a deployment of the package
// with the specified features enabled.
func (p Pkg) ProvidedFilesets(
	parentSource []string, enabledFeatures []string,
) (provided []res.Attached[FilesetRes, []string]) {
	return providedResources(
		p, parentSource, enabledFeatures, func(res ProvidedRes) attachedResGetter[FilesetRes] {
			return res.AttachedFilesets
		},
	)
}

// ProvidedFileExports returns a slice of all file exports provided by a deployment of the package
// with the specified features enabled.
func (p Pkg) ProvidedFileExports(
	parentSource []string, enabledFeatures []string,
) (provided []res.Attached[FileExportRes, []string]) {
	return providedResources(
		p, parentSource, enabledFeatures, func(res ProvidedRes) attachedResGetter[FileExportRes] {
			return res.AttachedFileExports
		},
	)
}

// PkgDecl

// LoadPkgDecl loads a PkgDecl from the specified file path in the provided base filesystem.
func LoadPkgDecl(fsys ffs.PathedFS, filePath string) (PkgDecl, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return PkgDecl{}, errors.Wrapf(
			err, "couldn't read package declaration file %s/%s", fsys.Path(), filePath,
		)
	}
	declaration := PkgDecl{}
	if err = yaml.Unmarshal(bytes, &declaration); err != nil {
		return PkgDecl{}, errors.Wrap(err, "couldn't parse package declaration")
	}

	return declaration.AddDefaults(), nil
}

// AddDefaults makes a copy with empty values replaced by default values.
func (d PkgDecl) AddDefaults() PkgDecl {
	d.Host = d.Host.AddDefaults()
	d.Deployment = d.Deployment.AddDefaults()
	updatedFeatures := make(map[string]PkgFeatureSpec)
	for name, feature := range d.Features {
		updatedFeatures[name] = feature.AddDefaults()
	}
	d.Features = updatedFeatures
	return d
}

// PkgHostSpec

// ResAttachmentSource returns the source path for resources under the PkgHostSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgHostSpec instance.
// The resulting slice is useful for constructing [res.Attached] instances.
func (s PkgHostSpec) ResAttachmentSource(parentSource []string) []string {
	return append(parentSource, "host specification")
}

// AddDefaults makes a copy with empty values replaced by default values.
func (s PkgHostSpec) AddDefaults() PkgHostSpec {
	s.Provides = s.Provides.AddDefaults()
	return s
}

// PkgDeplSpec

// ResAttachmentSource returns the source path for resources under the PkgDeplSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgDeplSpec instance.
// The resulting slice is useful for constructing [res.Attached] instances.
func (s PkgDeplSpec) ResAttachmentSource(parentSource []string) []string {
	return append(parentSource, "deployment specification")
}

// AddDefaults makes a copy with empty values replaced by default values.
func (s PkgDeplSpec) AddDefaults() PkgDeplSpec {
	s.Provides = s.Provides.AddDefaults()
	return s
}

// PkgFeatureSpec

// ResAttachmentSource returns the source path for resources under the PkgFeatureSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgFeatureSpec instance.
// The resulting slice is useful for constructing [res.Attached] instances.
func (s PkgFeatureSpec) ResAttachmentSource(parentSource []string, featureName string) []string {
	return append(parentSource, fmt.Sprintf("feature %s", featureName))
}

// AddDefaults makes a copy with empty values replaced by default values.
func (s PkgFeatureSpec) AddDefaults() PkgFeatureSpec {
	s.Provides = s.Provides.AddDefaults()
	return s
}
