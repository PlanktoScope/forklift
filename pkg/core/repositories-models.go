// Package core implements the specification of repositories and packages for the Forklift package
// management system.
package core

import ffs "github.com/forklift-run/forklift/pkg/fs"

// A FSRepo is a Forklift repository stored at the root of a [fs.FS] filesystem.
type FSRepo struct {
	// Repo is the Forklift repository at the root of the filesystem.
	Repo
	// FS is a filesystem which contains the repository's contents.
	FS ffs.PathedFS
}

// A Repo is a collection of Forklift packages which are tested, released, distributed, and
// upgraded together.
type Repo struct {
	// Decl is the definition of the repository.
	Decl RepoDecl
	// Version is the version or pseudoversion of the repository.
	Version string
}

// RepoDeclFile is the name of the file defining each repository.
const RepoDeclFile = "forklift-repository.yml"

// A RepoDecl defines a repository.
type RepoDecl struct {
	// ForkliftVersion indicates that the repo was written assuming the semantics of a given version
	// of Forklift. The version must be a valid Forklift version, and it sets the minimum version of
	// Forklift required to use the repository. The Forklift tool refuses to use repositories
	// declaring newer Forklift versions for any operations beyond printing information.
	ForkliftVersion string `yaml:"forklift-version"`
	// Repo defines the basic metadata for the repository.
	Repo RepoSpec `yaml:"repository"`
}

// RepoSpec defines the basic metadata for a repository.
type RepoSpec struct {
	// Path is the repository path, which acts as the canonical name for the repository. It should
	// just be the path of the VCS repository for the Forklift repository.
	Path string `yaml:"path"`
	// Description is a short description of the repository to be shown to users.
	Description string `yaml:"description"`
	// ReadmeFile is the name of a readme file to be shown to users.
	ReadmeFile string `yaml:"readme-file"`
}
