package forklift

type DeplConfig struct {
	Package  string   `yaml:"package,omitempty"`
	Features []string `yaml:"features,omitempty"`
}

type Depl struct {
	Name   string
	Config DeplConfig
	Pkg    VersionedPkg
}
