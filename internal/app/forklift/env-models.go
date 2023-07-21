// Package forklift provides the core functionality of the forklift tool
package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type FSEnv struct {
	FS pallets.PathedFS
}

type EnvConfig struct {
	Environment EnvSpec `yaml:"environment,omitempty"`
}

type EnvSpec struct {
	Description string `yaml:"description,omitempty"`
}
