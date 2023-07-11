package forklift

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const deplsDirName = "deployments"

func DeplsFS(envFS fs.FS) (fs.FS, error) {
	return fs.Sub(envFS, deplsDirName)
}

// Depl

func (d *Depl) EnabledFeatures() (enabled map[string]PkgFeatureSpec, err error) {
	all := d.Pkg.Cached.Config.Features
	enabled = make(map[string]PkgFeatureSpec)
	for _, name := range d.Config.Features {
		featureSpec, ok := all[name]
		if !ok {
			return nil, errors.Errorf("unrecognized feature %s", name)
		}
		enabled[name] = featureSpec
	}
	return enabled, nil
}

func (d *Depl) DisabledFeatures() map[string]PkgFeatureSpec {
	all := d.Pkg.Cached.Config.Features
	enabled := make(map[string]struct{})
	for _, name := range d.Config.Features {
		enabled[name] = struct{}{}
	}
	disabled := make(map[string]PkgFeatureSpec)
	for name := range all {
		if _, ok := enabled[name]; ok {
			continue
		}
		disabled[name] = all[name]
	}
	return disabled
}

func sortKeys[Value any](m map[string]Value) (sorted []string) {
	sorted = make([]string, 0, len(m))
	for key := range m {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	return sorted
}

func (d *Depl) CheckAllConflicts(candidates []Depl) (conflicts []DeplConflict, err error) {
	conflicts = make([]DeplConflict, 0, len(candidates))
	for _, candidate := range candidates {
		conflict, err := d.CheckConflicts(candidate)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check conflicts with deployment %s", candidate.Name)
		}
		if conflict.HasConflict() {
			conflicts = append(conflicts, conflict)
		}
	}
	return conflicts, nil
}

func (d *Depl) CheckConflicts(candidate Depl) (DeplConflict, error) {
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return DeplConflict{}, errors.Errorf(
			"couldn't determine enabled features of deployment %s", d.Name,
		)
	}
	candidateEnabledFeatures, err := candidate.EnabledFeatures()
	if err != nil {
		return DeplConflict{}, errors.Errorf(
			"couldn't determine enabled features of deployment %s", candidate.Name,
		)
	}
	return DeplConflict{
		First:  *d,
		Second: candidate,
		Name:   d.Name == candidate.Name,
		Listeners: CheckResourcesConflicts(
			d.providedListeners(enabledFeatures), candidate.providedListeners(candidateEnabledFeatures),
		),
		Networks: CheckResourcesConflicts(
			d.providedNetworks(enabledFeatures), candidate.providedNetworks(candidateEnabledFeatures),
		),
		Services: CheckResourcesConflicts(
			d.providedServices(enabledFeatures), candidate.providedServices(candidateEnabledFeatures),
		),
	}, nil
}

func (d *Depl) providedListeners(
	enabledFeatures map[string]PkgFeatureSpec,
) (provided []AttachedResource[ListenerResource]) {
	pkgConfig := d.Pkg.Cached.Config
	parentSource := d.attachmentSource()

	provided = append(provided, pkgConfig.Host.Provides.AttachedListeners(
		pkgConfig.Host.attachmentSource(parentSource),
	)...)
	provided = append(provided, pkgConfig.Deployment.Provides.AttachedListeners(
		pkgConfig.Deployment.attachmentSource(parentSource),
	)...)

	orderedFeatureNames := sortKeys(enabledFeatures)
	for _, featureName := range orderedFeatureNames {
		feature := enabledFeatures[featureName]
		provided = append(provided, feature.Provides.AttachedListeners(
			feature.attachmentSource(parentSource, featureName),
		)...)
	}
	return provided
}

func (d *Depl) attachmentSource() []string {
	return []string{
		fmt.Sprintf("deployment %s", d.Name),
		fmt.Sprintf("package %s", d.Config.Package),
	}
}

func (d *Depl) requiredNetworks(
	enabledFeatures map[string]PkgFeatureSpec,
) (required []AttachedResource[NetworkResource]) {
	pkgConfig := d.Pkg.Cached.Config
	parentSource := d.attachmentSource()

	required = append(required, pkgConfig.Deployment.Requires.AttachedNetworks(
		pkgConfig.Deployment.attachmentSource(parentSource),
	)...)

	orderedFeatureNames := sortKeys(enabledFeatures)
	for _, featureName := range orderedFeatureNames {
		feature := enabledFeatures[featureName]
		required = append(required, feature.Requires.AttachedNetworks(
			feature.attachmentSource(parentSource, featureName),
		)...)
	}
	return required
}

func (d *Depl) providedNetworks(
	enabledFeatures map[string]PkgFeatureSpec,
) (provided []AttachedResource[NetworkResource]) {
	pkgConfig := d.Pkg.Cached.Config
	parentSource := d.attachmentSource()

	provided = append(provided, pkgConfig.Host.Provides.AttachedNetworks(
		pkgConfig.Host.attachmentSource(parentSource),
	)...)
	provided = append(provided, pkgConfig.Deployment.Provides.AttachedNetworks(
		pkgConfig.Deployment.attachmentSource(parentSource),
	)...)

	orderedFeatureNames := sortKeys(enabledFeatures)
	for _, featureName := range orderedFeatureNames {
		feature := enabledFeatures[featureName]
		provided = append(provided, feature.Provides.AttachedNetworks(
			feature.attachmentSource(parentSource, featureName),
		)...)
	}
	return provided
}

func (d *Depl) requiredServices(
	enabledFeatures map[string]PkgFeatureSpec,
) (required []AttachedResource[ServiceResource]) {
	pkgConfig := d.Pkg.Cached.Config
	parentSource := d.attachmentSource()

	required = append(required, pkgConfig.Deployment.Requires.AttachedServices(
		pkgConfig.Deployment.attachmentSource(parentSource),
	)...)

	orderedFeatureNames := sortKeys(enabledFeatures)
	for _, featureName := range orderedFeatureNames {
		feature := enabledFeatures[featureName]
		required = append(required, feature.Requires.AttachedServices(
			feature.attachmentSource(parentSource, featureName),
		)...)
	}
	return required
}

func (d *Depl) providedServices(
	enabledFeatures map[string]PkgFeatureSpec,
) (provided []AttachedResource[ServiceResource]) {
	pkgConfig := d.Pkg.Cached.Config
	parentSource := d.attachmentSource()

	provided = append(provided, pkgConfig.Host.Provides.AttachedServices(
		pkgConfig.Host.attachmentSource(parentSource),
	)...)
	provided = append(provided, pkgConfig.Deployment.Provides.AttachedServices(
		pkgConfig.Deployment.attachmentSource(parentSource),
	)...)

	orderedFeatureNames := sortKeys(enabledFeatures)
	for _, featureName := range orderedFeatureNames {
		feature := enabledFeatures[featureName]
		provided = append(provided, feature.Provides.AttachedServices(
			feature.attachmentSource(parentSource, featureName),
		)...)
	}
	return provided
}

func (d *Depl) CheckMissingDependencies(candidates []Depl) (MissingDeplDependencies, error) {
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return MissingDeplDependencies{}, errors.Errorf(
			"couldn't determine enabled features of deployment %s", d.Name,
		)
	}
	candidateEnabledFeatures := make([]map[string]PkgFeatureSpec, 0, len(candidates))
	for _, candidate := range candidates {
		f, err := candidate.EnabledFeatures()
		if err != nil {
			return MissingDeplDependencies{}, errors.Errorf(
				"couldn't determine enabled features of deployment %s", candidate.Name,
			)
		}
		candidateEnabledFeatures = append(candidateEnabledFeatures, f)
	}

	var (
		allProvidedNetworks []AttachedResource[NetworkResource]
		allProvidedServices []AttachedResource[ServiceResource]
	)
	for i, candidate := range candidates {
		allProvidedNetworks = append(
			allProvidedNetworks, candidate.providedNetworks(candidateEnabledFeatures[i])...,
		)
		allProvidedServices = append(
			allProvidedServices, candidate.providedServices(candidateEnabledFeatures[i])...,
		)
	}

	return MissingDeplDependencies{
		Depl: *d,
		Networks: CheckResourcesDependencies(
			d.requiredNetworks(enabledFeatures), allProvidedNetworks,
		),
		Services: CheckResourcesDependencies(
			splitMultiPathServiceResources(d.requiredServices(enabledFeatures)), allProvidedServices,
		),
	}, nil
}

func splitMultiPathServiceResources(
	serviceResources []AttachedResource[ServiceResource],
) (split []AttachedResource[ServiceResource]) {
	split = make([]AttachedResource[ServiceResource], 0, len(serviceResources))
	for _, service := range serviceResources {
		if len(service.Resource.Paths) == 0 {
			split = append(split, service)
		}
		for _, path := range service.Resource.Paths {
			pathService := service.Resource
			pathService.Paths = []string{path}
			split = append(split, AttachedResource[ServiceResource]{
				Resource: pathService,
				Source:   service.Source,
			})
		}
	}
	return split
}

// Loading

func loadDeplConfig(deplsFS fs.FS, filePath string) (DeplConfig, error) {
	bytes, err := fs.ReadFile(deplsFS, filePath)
	if err != nil {
		return DeplConfig{}, errors.Wrapf(err, "couldn't read deployment config file %s", filePath)
	}
	config := DeplConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
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

// Constraint-checking functions

func CheckDeplConflicts(depls []Depl) (conflicts []DeplConflict, err error) {
	for i, depl := range depls {
		deplConflicts, err := depl.CheckAllConflicts(depls[i+1:])
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check for conflicts with deployment %s", depl.Name)
		}
		conflicts = append(conflicts, deplConflicts...)
	}
	return conflicts, nil
}

func CheckDeplDependencies(
	depls []Depl,
) (missingDeps []MissingDeplDependencies, err error) {
	for _, depl := range depls {
		deplMissingDeps, err := depl.CheckMissingDependencies(depls)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check dependencies of deployment %s", depl.Name)
		}
		if deplMissingDeps.HasMissingDependency() {
			missingDeps = append(missingDeps, deplMissingDeps)
		}
	}
	return missingDeps, nil
}
