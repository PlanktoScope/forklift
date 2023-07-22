package forklift

const (
	// DeplsDirName is the directory in a Forklift environment which contains deployment
	// configurations.
	DeplsDirName = "deployments"
	// DeplsFileExt is the file extension for deployment configuration files.
	DeplsFileExt = ".deploy.yml"
)

// A Depl is a Pallets package deployment, a complete configuration of how a Pallet package is to be
// deployed on a Docker host.
type Depl struct {
	Name   string
	Config DeplConfig
	// TODO: populate this with a Resolve() method
	Pkg *VersionedPkg
}

// A DeplConfig defines a Pallets package deployment.
type DeplConfig struct {
	Package  string   `yaml:"package,omitempty"`
	Features []string `yaml:"features,omitempty"`
}
