package forklift

// Repository specifications

type RepoSpec struct {
	Path        string `yaml:"path,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type RepoConfig struct {
	Repository RepoSpec `yaml:"repository,omitempty"`
}

// Package specifications

type PkgConfig struct {
	Package    PkgSpec                   `yaml:"package,omitempty"`
	Host       PkgHostSpec               `yaml:"host,omitempty"`
	Deployment PkgDeplSpec               `yaml:"deployment,omitempty"`
	Features   map[string]PkgFeatureSpec `yaml:"features,omitempty"`
}

type PkgMaintainer struct {
	Name  string `yaml:"name,omitempty"`
	Email string `yaml:"email,omitempty"`
}

type PkgSpec struct {
	Description string          `yaml:"description,omitempty"`
	Maintainers []PkgMaintainer `yaml:"maintainers,omitempty"`
	License     string          `yaml:"license,omitempty"`
	LicenseFile string          `yaml:"license-file,omitempty"`
	Sources     []string        `yaml:"sources,omitempty"`
}

type ProvidedResources struct {
	Listeners []ListenerResource `yaml:"listeners,omitempty"`
	Networks  []NetworkResource  `yaml:"networks,omitempty"`
	Services  []ServiceResource  `yaml:"services,omitempty"`
}

type RequiredResources struct {
	Networks []NetworkResource `yaml:"networks,omitempty"`
	Services []ServiceResource `yaml:"services,omitempty"`
}

type ListenerResource struct {
	Description string `yaml:"description,omitempty"`
	Port        int    `yaml:"port,omitempty"`
	Protocol    string `yaml:"protocol,omitempty"`
}

type NetworkResource struct {
	Description string `yaml:"description,omitempty"`
	Name        string `yaml:"name,omitempty"`
}

type ServiceResource struct {
	Description string   `yaml:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Port        int      `yaml:"port,omitempty"`
	Protocol    string   `yaml:"protocol,omitempty"`
	Paths       []string `yaml:"paths,omitempty"`
}

type PkgHostSpec struct {
	Provides ProvidedResources `yaml:"provides,omitempty"`
}

type PkgDeplSpec struct {
	Name           string            `yaml:"name,omitempty"`
	DefinitionFile string            `yaml:"definition-file,omitempty"`
	Requires       RequiredResources `yaml:"requires,omitempty"`
	Provides       ProvidedResources `yaml:"provides,omitempty"`
}

type PkgFeatureSpec struct {
	Description string            `yaml:"description,omitempty"`
	Requires    RequiredResources `yaml:"requires,omitempty"`
	Provides    ProvidedResources `yaml:"provides,omitempty"`
}
