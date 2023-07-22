// Package forklift provides the core functionality of the forklift tool
package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// A FSEnv is a Forklift environment configuration stored at the root of a [fs.FS] filesystem.
type FSEnv struct {
	// Env is the Forklift environment at the root of the filesystem.
	Env
	// FS is a filesystem which contains the environment's contents.
	FS pallets.PathedFS
}

// An Env is a Forklift environment, a complete specification of all Pallet package deployments
// which should be active on a Docker host.
type Env struct {
	// Config is the Forklift environment specification for the environment.
	Config EnvConfig
}

// EnvSpecFile is the name of the file defining each Forklift environment.
const EnvSpecFile = "forklift-env.yml"

// A EnvConfig defines a Forklift environment.
type EnvConfig struct {
	// Environment defines the basic metadata for the environment.
	Environment EnvSpec `yaml:"environment,omitempty"`
}

// EnvSpec defines the basic metadata for a Forklift environment.
type EnvSpec struct {
	// Description is a short description of the environment to be shown to users.
	Description string `yaml:"description,omitempty"`
}
