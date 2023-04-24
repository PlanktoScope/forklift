package forklift

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const deplsDirName = "deployments"

func DeplsFS(envFS fs.FS) (fs.FS, error) {
	return fs.Sub(envFS, deplsDirName)
}

func (d *Depl) EnabledFeatures(
	all map[string]PkgFeatureSpec,
) (enabled map[string]PkgFeatureSpec, err error) {
	enabled = make(map[string]PkgFeatureSpec)
	for _, name := range d.Config.Features {
		featureSpec, ok := all[name]
		if !ok {
			return nil, errors.Errorf("unrecognized feature%s", name)
		}
		enabled[name] = featureSpec
	}
	return enabled, nil
}

func (d *Depl) DisabledFeatures(
	all map[string]PkgFeatureSpec,
) (disabled map[string]PkgFeatureSpec, err error) {
	enabled := make(map[string]struct{})
	for _, name := range d.Config.Features {
		enabled[name] = struct{}{}
	}
	disabled = make(map[string]PkgFeatureSpec)
	for name := range all {
		if _, ok := enabled[name]; ok {
			continue
		}
		disabled[name] = all[name]
	}
	return disabled, nil
}

func loadDeplConfig(deplsFS fs.FS, filePath string) (DeplConfig, error) {
	file, err := deplsFS.Open(filePath)
	if err != nil {
		return DeplConfig{}, errors.Wrapf(err, "couldn't open file %s", filePath)
	}
	buf, err := loadFile(file)
	if err != nil {
		return DeplConfig{}, errors.Wrap(err, "couldn't read deployment config file")
	}
	config := DeplConfig{}
	if err = yaml.Unmarshal(buf.Bytes(), &config); err != nil {
		return DeplConfig{}, errors.Wrap(err, "couldn't parse deployment config")
	}
	return config, nil
}

func LoadDepl(envFS, cacheFS fs.FS, deplName string) (Depl, error) {
	deplsFS, err := DeplsFS(envFS)
	if err != nil {
		return Depl{}, errors.Wrap(
			err, "couldn't open directory for Pallet package deployments in local environment",
		)
	}
	reposFS, err := VersionedReposFS(envFS)
	if err != nil {
		return Depl{}, errors.Wrap(
			err, "couldn't open directory for Pallet repositories in local environment",
		)
	}

	depl := Depl{
		Name: deplName,
	}
	depl.Config, err = loadDeplConfig(deplsFS, fmt.Sprintf("%s.deploy.yml", deplName))
	if err != nil {
		return Depl{}, errors.Wrapf(err, "couldn't load package deployment config for %s", deplName)
	}

	pkgPath := depl.Config.Package
	depl.Pkg, err = LoadVersionedPkg(reposFS, cacheFS, pkgPath)
	if err != nil {
		return Depl{}, errors.Wrapf(
			err, "couldn't load versioned package %s to be deployed by local environment", pkgPath,
		)
	}

	return depl, nil
}

func ListDepls(envFS fs.FS, cacheFS fs.FS) ([]Depl, error) {
	deplsFS, err := DeplsFS(envFS)
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for Pallet package deployments in local environment",
		)
	}
	files, err := doublestar.Glob(deplsFS, "*.deploy.yml")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for Pallet package deployment configs")
	}

	deplNames := make([]string, 0, len(files))
	deplMap := make(map[string]Depl)
	for _, filePath := range files {
		deplName := strings.TrimSuffix(filePath, ".deploy.yml")
		if _, ok := deplMap[deplName]; ok {
			return nil, errors.Errorf(
				"package deployment %s repeatedly specified by the local environment", deplName,
			)
		}
		deplNames = append(deplNames, deplName)
		deplMap[deplName], err = LoadDepl(envFS, cacheFS, deplName)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load package deployment specification %s", deplName)
		}
	}

	orderedDepls := make([]Depl, 0, len(deplNames))
	for _, deplName := range deplNames {
		orderedDepls = append(orderedDepls, deplMap[deplName])
	}
	return orderedDepls, nil
}
