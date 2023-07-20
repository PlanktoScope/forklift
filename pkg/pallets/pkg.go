package pallets

import (
	"fmt"
)

// PkgHostSpec

// ResourceAttachmentSource returns the source path for resources under the PkgHostSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgHostSpec instance.
func (s PkgHostSpec) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, "host specification")
}

// PkgDeplSpec

// ResourceAttachmentSource returns the source path for resources under the PkgDeplSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgDeplSpec instance.
func (s PkgDeplSpec) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, "deployment specification")
}

// DefinesStack determines whether the PkgDeplSpec instance defines a Docker stack to be deployed.
func (s PkgDeplSpec) DefinesStack() bool {
	return s.DefinitionFile != ""
}

// PkgFeatureSpec

// ResourceAttachmentSource returns the source path for resources under the PkgFeatureSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgFeatureSpec instance.
func (s PkgFeatureSpec) ResourceAttachmentSource(parentSource []string, featureName string) []string {
	return append(parentSource, fmt.Sprintf("feature %s", featureName))
}
