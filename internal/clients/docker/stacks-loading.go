package docker

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/schema"
	"github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
)

// LoadStackDefinition parses the specified Docker Compose file and returns its Config and version.
func LoadStackDefinition(parentFS fs.FS, filePath string) (*types.Config, error) {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/loader package's
	// LoadComposefile function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in the command.Cli parameter and opts parameters.
	configDetails, err := getConfigDetails(parentFS, filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get config details")
	}

	dicts := getDictsFrom(configDetails.ConfigFiles)
	config, err := loader.Load(configDetails)
	if err != nil {
		if fpe, ok := err.(*loader.ForbiddenPropertiesError); ok {
			return nil, errors.Errorf("compose file contains forbidden options %+v", fpe.Properties)
		}
		return nil, errors.Wrapf(err, "couldn't load config")
	}
	unsupportedProperties := loader.GetUnsupportedProperties(dicts...)
	if len(unsupportedProperties) > 0 {
		return nil, errors.Errorf(
			"compose file contains unsupported options %+v", unsupportedProperties,
		)
	}
	deprecatedProperties := loader.GetDeprecatedProperties(dicts...)
	if len(deprecatedProperties) > 0 {
		return nil, errors.Errorf(
			"compose file contains deprecated options %+v", deprecatedProperties,
		)
	}

	return config, nil
}

// getConfigDetails parses the composefile and returns its ConfigDetails.
func getConfigDetails(parentFS fs.FS, filePath string) (types.ConfigDetails, error) {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/loader package's
	// GetConfigDetails function, which is licensed under Apache-2.0. This function was changed to
	// take a single compose file from a fs.FS.
	details := types.ConfigDetails{
		WorkingDir: filepath.Dir(filePath),
	}
	configFile, err := loadConfigFile(parentFS, filePath)
	if err != nil {
		return types.ConfigDetails{}, err
	}
	details.ConfigFiles = []types.ConfigFile{configFile}
	details.Version = schema.Version(configFile.Config)
	details.Environment, err = buildEnvironment(os.Environ())
	return details, err
}

func buildEnvironment(env []string) (map[string]string, error) {
	// This function is copied verbatim from the github.com/docker/cli/cli/command/stack/loader
	// package's buildEnvironment function, which is licensed under Apache-2.0.
	result := make(map[string]string, len(env))
	for _, s := range env {
		if runtime.GOOS == "windows" && len(s) > 0 {
			// cmd.exe can have special environment variables whose names start with "=". They are only
			// there for MS-DOC compatibility and we should ignore them.
			//
			// https://ss64.com/nt/syntax-variables.html
			// https://devblogs.microsoft.com/oldnewthing/20100506-00/?p=14133
			// https://github.com/docker/cli/issues/4078
			if s[0] == '=' {
				continue
			}
		}

		k, v, ok := strings.Cut(s, "=")
		if !ok || k == "" {
			return result, errors.Errorf("unexpected environment variable '%s'", s)
		}
		// value may be set, but empty if "s" is like "K=", not "K"
		result[k] = v
	}
	return result, nil
}

func loadConfigFile(parentFS fs.FS, filePath string) (types.ConfigFile, error) {
	file, err := parentFS.Open(filePath)
	if err != nil {
		return types.ConfigFile{}, errors.Wrapf(err, "couldn't open file %s", filePath)
	}
	buf, err := loadFile(file)
	if err != nil {
		return types.ConfigFile{}, errors.Wrap(err, "couldn't read Docker Compose config file")
	}
	config, err := loader.ParseYAML(buf.Bytes())
	if err != nil {
		return types.ConfigFile{}, errors.Wrap(err, "couldn't parse package config")
	}
	return types.ConfigFile{
		Filename: filePath,
		Config:   config,
	}, nil
}

func loadFile(file fs.File) (bytes.Buffer, error) {
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(file)
	return buf, errors.Wrap(err, "couldn't load file")
}

func getDictsFrom(configFiles []types.ConfigFile) []map[string]interface{} {
	// This function is copied verbatim from the github.com/docker/cli/cli/command/stack/loader
	// package's getDictsFrom function, which is licensed under Apache-2.0.
	dicts := make([]map[string]interface{}, 0, len(configFiles))
	for _, configFile := range configFiles {
		dicts = append(dicts, configFile.Config)
	}

	return dicts
}
