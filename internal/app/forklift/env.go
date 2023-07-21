package forklift

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// FSEnv

func FindParentEnv(cwd string) (path string, err error) {
	envCandidatePath, err := filepath.Abs(cwd)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't convert '%s' into an absolute path", cwd)
	}
	for envCandidatePath != "." && envCandidatePath != "/" {
		f := os.DirFS(envCandidatePath)
		_, err := fs.ReadFile(f, "forklift-env.yml")
		if err == nil {
			return envCandidatePath, nil
		}
		envCandidatePath = filepath.Dir(envCandidatePath)
	}
	return "", errors.Errorf(
		"no environment config file found in any parent directory of %s", cwd,
	)
}

func LoadFSEnv(path string) (*FSEnv, error) {
	if !Exists(path) {
		return nil, errors.Errorf("couldn't find environment at %s", path)
	}
	return &FSEnv{
		FS: pallets.AttachPath(os.DirFS(path), path),
	}, nil
}

func (e *FSEnv) Exists() bool {
	return Exists(e.FS.Path())
}

func (e *FSEnv) Remove() error {
	return os.RemoveAll(e.FS.Path())
}

// EnvConfig

func LoadEnvConfig(envPath string) (EnvConfig, error) {
	f := os.DirFS(envPath)
	bytes, err := fs.ReadFile(f, "forklift-env.yml")
	if err != nil {
		return EnvConfig{}, errors.Wrap(err, "couldn't read forklift environment config file")
	}
	config := EnvConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return EnvConfig{}, errors.Wrap(err, "couldn't parse environment config")
	}
	return config, nil
}
