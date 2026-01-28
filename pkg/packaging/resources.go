package packaging

import (
	"fmt"

	res "github.com/forklift-run/forklift/pkg/resources"
)

// RequiredRes describes a set of resource requirements for some aspect of a package.
type RequiredRes struct {
	// Networks is a list of requirements for Docker networks.
	Networks []NetworkRes `yaml:"networks,omitempty"`
	// Services is a list of requirements for network services.
	Services []ServiceRes `yaml:"services,omitempty"`
	// Filesets is a list of requirements for files/directories.
	Filesets []FilesetRes `yaml:"filesets,omitempty"`
}

// ProvidedRes describes a set of resources provided by some aspect of a package.
type ProvidedRes struct {
	// Listeners is a list of host port listeners.
	Listeners []ListenerRes `yaml:"listeners,omitempty"`
	// Networks is a list of Docker networks.
	Networks []NetworkRes `yaml:"networks,omitempty"`
	// Services is a list of network services.
	Services []ServiceRes `yaml:"services,omitempty"`
	// Filesets is a list of files/directories.
	Filesets []FilesetRes `yaml:"filesets,omitempty"`
	// FileExports is a list of files/directories.
	FileExports []FileExportRes `yaml:"file-exports,omitempty"`
}

// Pkg

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

// RequiredRes

// AttachedNetworks returns a list of [res.Attached] instances for each respective Docker
// network resource requirement in the RequiredRes instance, adding a string to the provided
// list of origin elements which describes the origin of the RequiredRes instance.
func (r RequiredRes) AttachedNetworks(origin []string) []res.Attached[NetworkRes, []string] {
	return res.Attach(r.Networks, append(origin, requiresSourcePart))
}

// AttachedServices returns a list of [res.Attached] instances for each respective network
// service resource requirement in the RequiredRes instance, adding a string to the provided
// list of origin elements which describes the origin of the RequiredRes instance.
func (r RequiredRes) AttachedServices(origin []string) []res.Attached[ServiceRes, []string] {
	return res.Attach(r.Services, append(origin, requiresSourcePart))
}

// AttachedFilesets returns a list of [res.Attached] instances for each respective fileset
// resource requirement in the RequiredRes instance, adding a string to the provided
// list of origin elements which describes the origin of the RequiredRes instance.
func (r RequiredRes) AttachedFilesets(origin []string) []res.Attached[FilesetRes, []string] {
	return res.Attach(r.Filesets, append(origin, requiresSourcePart))
}

// ProvidedRes

const (
	providesSourcePart = "provides resource"
	requiresSourcePart = "requires resource"
)

// AttachedListeners returns a list of [res.Attached] instances for each respective host port
// listener in the ProvidedRes instance, adding a string to the provided list of origin
// elements which describes the origin of the ProvidedRes instance.
func (r ProvidedRes) AttachedListeners(origin []string) []res.Attached[ListenerRes, []string] {
	return res.Attach(r.Listeners, append(origin, providesSourcePart))
}

// AttachedNetworks returns a list of [res.Attached] instances for each respective Docker
// network in the ProvidedRes instance, adding a string to the provided list of origin
// elements which describes the origin of the ProvidedRes instance.
func (r ProvidedRes) AttachedNetworks(origin []string) []res.Attached[NetworkRes, []string] {
	return res.Attach(r.Networks, append(origin, providesSourcePart))
}

// AttachedServices returns a list of [res.Attached] instances for each respective network
// service in the ProvidedRes instance, adding a string to the provided list of origin
// elements which describes the origin of the ProvidedRes instance.
func (r ProvidedRes) AttachedServices(origin []string) []res.Attached[ServiceRes, []string] {
	return res.Attach(r.Services, append(origin, providesSourcePart))
}

// AttachedFilesets returns a list of [res.Attached] instances for each respective fileset
// in the ProvidedRes instance, adding a string to the provided list of origin
// elements which describes the origin of the ProvidedRes instance.
func (r ProvidedRes) AttachedFilesets(origin []string) []res.Attached[FilesetRes, []string] {
	return res.Attach(r.Filesets, append(origin, providesSourcePart))
}

// AttachedFileExports returns a list of [res.Attached] instances for each respective file export
// in the ProvidedRes instance, adding a string to the provided list of origin
// elements which describes the origin of the ProvidedRes instance.
func (r ProvidedRes) AttachedFileExports(origin []string) []res.Attached[FileExportRes, []string] {
	return res.Attach(r.FileExports, append(origin, providesSourcePart))
}

// AddDefaults makes a copy with empty values replaced by default values.
func (r ProvidedRes) AddDefaults() ProvidedRes {
	updatedFileExports := make([]FileExportRes, 0, len(r.FileExports))
	for _, fileExport := range r.FileExports {
		updatedFileExports = append(updatedFileExports, fileExport.AddDefaults())
	}
	r.FileExports = updatedFileExports
	return r
}
