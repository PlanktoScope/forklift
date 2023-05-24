package forklift

// Repository specifications

type RepoSpec struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

type RepoConfig struct {
	Repository RepoSpec `yaml:"repository"`
}

// Package specifications

type PkgConfig struct {
	Package    PkgSpec                   `yaml:"package"`
	Host       PkgHostSpec               `yaml:"host"`
	Deployment PkgDeplSpec               `yaml:"deployment"`
	Features   map[string]PkgFeatureSpec `yaml:"features"`
}

type PkgMaintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type PkgSpec struct {
	Description string          `yaml:"description"`
	Maintainers []PkgMaintainer `yaml:"maintainers"`
	License     string          `yaml:"license"`
	LicenseFile string          `yaml:"license-file"`
	Sources     []string        `yaml:"sources"`
}

type ProvidedResources struct {
	Listeners []ListenerResource `yaml:"listeners"`
	Networks  []NetworkResource  `yaml:"networks"`
	Services  []ServiceResource  `yaml:"services"`
}

type RequiredResources struct {
	Networks []NetworkResource `yaml:"networks"`
	Services []ServiceResource `yaml:"services"`
}

type ListenerResource struct {
	Description string `yaml:"description"`
	Port        int    `yaml:"port"`
	Protocol    string `yaml:"protocol"`
}

type NetworkResource struct {
	Description string `yaml:"description"`
	Name        string `yaml:"name"`
}

type ServiceResource struct {
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Port        int      `yaml:"port"`
	Protocol    string   `yaml:"protocol"`
	Paths       []string `yaml:"paths"`
}

type PkgHostSpec struct {
	Provides ProvidedResources `yaml:"provides"`
}

type PkgDeplSpec struct {
	Name           string            `yaml:"name"`
	DefinitionFile string            `yaml:"definition-file"`
	Requires       RequiredResources `yaml:"requires"`
	Provides       ProvidedResources `yaml:"provides"`
}

type PkgFeatureSpec struct {
	Description string            `yaml:"description"`
	Requires    RequiredResources `yaml:"requires"`
	Provides    ProvidedResources `yaml:"provides"`
}
