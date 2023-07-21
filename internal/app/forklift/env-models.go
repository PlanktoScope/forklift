// Package forklift provides the core functionality of the forklift tool
package forklift

type EnvConfig struct {
	Environment EnvSpec `yaml:"environment,omitempty"`
}

type EnvSpec struct {
	Description string `yaml:"description,omitempty"`
}
