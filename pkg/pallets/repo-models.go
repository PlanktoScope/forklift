// Package pallets implements the specification for the Pallets package management system.
package pallets

import (
	"io/fs"
)

// The result of comparison functions is one of these values.
const (
	CompareLT = -1
	CompareEQ = 0
	CompareGT = 1
)

// A Repo is a Pallet repository.
type Repo struct {
	// VCSRepoPath is the path of the VCS repository path which provides the Pallet repository.
	VCSRepoPath string
	// Subdir is the path of the repository within the VCS repository which provides the Pallet
	// repository.
	Subdir string
	// Config is the Pallet repository specification for the repository.
	Config RepoConfig
	// Version is the Pallet repository version or pseudoversion of the repository.
	Version string
}

// A FSRepo is a Pallet repository stored at the root of a [fs.FS] filesystem.
type FSRepo struct {
	// Repo is the Pallet repository at the root of the filesystem.
	Repo
	// FS is a filesystem which contains the repository's contents.
	FS fs.FS
	// Path is the path of the repository's filesystem.
	FSPath string
}

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
