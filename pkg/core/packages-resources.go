package core

import (
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
