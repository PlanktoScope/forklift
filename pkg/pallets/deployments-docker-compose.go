package forklift

import (
	"strings"

	dct "github.com/compose-spec/compose-go/v2/types"
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/clients/docker"
)

// ResolvedDepl: Docker Compose

// GetComposeFilenames returns a list of the paths of the Compose files which must be merged into
// the Compose app, with feature-flagged Compose files ordered based on the alphabetical order of
// enabled feature flags.
func (d *ResolvedDepl) GetComposeFilenames() ([]string, error) {
	composeFiles := append([]string{}, d.Pkg.Decl.Deployment.ComposeFiles...)

	// Add compose files from features
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't determine enabled features of deployment %s", d.Name)
	}
	for _, name := range sortKeys(enabledFeatures) {
		composeFiles = append(composeFiles, enabledFeatures[name].ComposeFiles...)
	}
	return composeFiles, nil
}

// DefinesComposeApp determines whether the deployment defines a Docker Compose app to be deployed.
func (d *ResolvedDepl) DefinesComposeApp() (bool, error) {
	composeFiles, err := d.GetComposeFilenames()
	if err != nil {
		return false, errors.Wrap(err, "couldn't determine Compose files for deployment")
	}
	if len(composeFiles) == 0 {
		return false, nil
	}
	for _, file := range composeFiles {
		if len(file) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// LoadComposeAppDefinition loads the deployment's Docker Compose app.
func (d *ResolvedDepl) LoadComposeAppDefinition(resolvePaths bool) (*dct.Project, error) {
	composeFiles, err := d.GetComposeFilenames()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't determine Compose files for deployment")
	}

	appDef, err := docker.LoadAppDefinition(
		d.Pkg.FS, GetComposeAppName(d.Name), composeFiles, nil, resolvePaths,
	)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load Docker Compose app definition for deployment %s of %s",
			d.Name, d.Pkg.FS.Path(),
		)
	}
	return appDef, nil
}

// GetComposeAppName converts the deployment's name into a string which is allowed for use as a
// Docker Compose app name. It assumes that the resulting name will not be excessively long for
// Docker Compose.
func GetComposeAppName(deplName string) string {
	return strings.ReplaceAll(deplName, "/", "_")
}
