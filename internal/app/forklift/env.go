package forklift

import (
	"io/fs"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

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
