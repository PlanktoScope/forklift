// Package forklift provides the core functionality of the forklift tool
package forklift

type EnvSpec struct {
	Description string `yaml:"description,omitempty"`
}

type EnvConfig struct {
	Environment EnvSpec `yaml:"environment,omitempty"`
}
