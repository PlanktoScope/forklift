package forklift

// A VersionLock is a specification of a particular version of a repo or package.
type VersionLock struct {
	// Def is the version lock definition.
	Def VersionLockDef
	// Version is the version string corresponding to the configured version.
	Version string
}

// VersionLockDefFile is the name of the file defining each version lock of a repo.
const VersionLockDefFile = "forklift-version-lock.yml"

// A VersionLockDef defines a requirement for a repo or package at a specific
// version.
type VersionLockDef struct {
	// Type specifies the type of version lock (either "version" or "pseudoversion")
	Type string `yaml:"type"`
	// Tag specifies the VCS repository tag associated with the version or pseudoversion, if it
	// exists. If the type is "version", the tag should point to the commit corresponding to the
	// version; if the type is "pseudoversion", the tag should be the highest-versioned tag in the
	// ancestry of the commit corresponding to the version (and it is used as a "base version").
	Tag string `yaml:"tag,omitempty"`
	// Timestamp specifies the commit time (in UTC) of the commit corresponding to the version, as
	// a 14-character string.
	Timestamp string `yaml:"timestamp"`
	// Commit specifies the full hash of the commit corresponding to the version.
	Commit string `yaml:"commit"`
}

const Timestamp = "20060102150405"
