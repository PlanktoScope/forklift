// Package pallets implements the specification for the Pallets package management system.
package pallets

// RepoSpecFile is the name of the file defining each Pallet repository.
const RepoSpecFile = "pallet-repository.yml"

// A RepoConfig defines a Pallet repository.
type RepoConfig struct {
	// Repository defines the basic metadata for the repository.
	Repository RepoSpec `yaml:"repository"`
}

// RepoSpec defines the basic metadata for a Pallet repository.
type RepoSpec struct {
	// Path is the Pallet repository path, which acts as the canonical name for the repository.
	// Typically, it consists of a GitHub repository root path followed by either a subdirectory or
	// by  nothing at all.
	Path string `yaml:"path"`
	// Description is a short description of the repository to be shown to users.
	Description string `yaml:"description"`
	// ReadmeFile is the name of a readme file to be shown to users.
	ReadmeFile string `yaml:"readme-file"`
}
