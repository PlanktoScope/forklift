package pallets

import (
	"fmt"
)

// PkgHostSpec

func (s PkgHostSpec) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, "host specification")
}

// PkgDeplSpec

func (s PkgDeplSpec) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, "deployment specification")
}

func (s PkgDeplSpec) DefinesStack() bool {
	return s.DefinitionFile != ""
}

// PkgFeatureSpec

func (s PkgFeatureSpec) ResourceAttachmentSource(parentSource []string, featureName string) []string {
	return append(parentSource, fmt.Sprintf("feature %s", featureName))
}
