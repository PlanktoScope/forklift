package docker

import (
	"bytes"
	"context"
	"io/fs"
	"path"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	dct "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/core"
)

// LoadAppDefinition parses the specified Docker Compose files from the provided fs as the current
// working directory, applies the provided environment variables, and returns a representation of
// the app as a Docker Compose project.
func LoadAppDefinition(
	fsys core.PathedFS, name string, configPaths []string, env map[string]string, resolvePaths bool,
) (*dct.Project, error) {
	// This function is adapted from the github.com/compose-spec/compose-go/cli package's
	// ProjectFromOptions function, which is licensed under Apache-2.0. This function was changed to
	// load files from a provided [core.PathedFS] and to integrate custom label-setting functionality
	// from the Apache-2.0-licensed github.com/docker/compose/cmd/compose package's ToProject
	// function.
	var configs []dct.ConfigFile
	for _, configPath := range configPaths {
		filename := path.Join(fsys.Path(), configPath)
		file, err := fsys.Open(configPath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't open file %s", filename)
		}
		buf, err := loadFile(file)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't read Docker Compose config file %s", filename)
		}
		configs = append(configs, dct.ConfigFile{
			Filename: filename,
			Content:  buf.Bytes(),
		})
	}

	if env == nil {
		env = map[string]string{}
	}
	project, err := loader.LoadWithContext(context.TODO(), dct.ConfigDetails{
		ConfigFiles: configs,
		WorkingDir:  fsys.Path(),
		Environment: env,
	}, func(opts *loader.Options) {
		opts.SetProjectName(name, true)
		opts.ResolvePaths = resolvePaths
	})
	if err != nil {
		return nil, err
	}
	project.Name = name
	project.ComposeFiles = configPaths

	// Add the standard labels used & expected by Docker Compose for the services (and thus the
	// containers of the services):
	for i, s := range project.Services {
		s.CustomLabels = map[string]string{
			api.ProjectLabel:     project.Name,
			api.ServiceLabel:     s.Name,
			api.VersionLabel:     api.ComposeVersion,
			api.WorkingDirLabel:  project.WorkingDir,
			api.ConfigFilesLabel: strings.Join(project.ComposeFiles, ","),
			api.OneoffLabel:      "False",
		}
		project.Services[i] = s
	}
	project.WithoutUnnecessaryResources()
	return project, nil
}

func loadFile(file fs.File) (bytes.Buffer, error) {
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(file)
	return buf, errors.Wrap(err, "couldn't load file")
}
