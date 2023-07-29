package forklift

// A VersionLock is a specification of a particular version of a Pallet repository or package.
type VersionLock struct {
	// Def is the Pallet version lock definition.
	Def VersionLockDef
	// Version is the version string corresponding to the configured version.
	Version string
}

// VersionLockDefFile is the name of the file defining each version lock of a Pallet repository.
// TODO: rename this to "forklift-version-lock.yml".
const VersionLockDefFile = "forklift-repo.yml"

// A VersionLockDef defines a requirement for a Pallet repository or package at a specific
// version.
type VersionLockDef struct {
	// BaseVersion specifies the VCS repository tag for the version, if it exists.
	BaseVersion string `yaml:"base-version,omitempty"`
	// Timestamp specifies the commit time (in UTC) of the commit corresponding to the version, as
	// a 14-character string.
	Timestamp string `yaml:"timestamp,omitempty"`
	// Commit specifies the full hash of the commit corresponding to the version.
	Commit string `yaml:"commit,omitempty"`
}

const Timestamp = "20060102150405"
