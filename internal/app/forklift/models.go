// Package forklift provides the core functionality of the forklift tool
package forklift

// Versioned repository specification

type RepoVersionConfig struct {
	Release string `yaml:"release"`
}

type RepoVersionLock struct {
	Version   string `yaml:"version"`
	Timestamp string `yaml:"timestamp"`
	Commit    string `yaml:"commit"`
}

type VersionedRepo struct {
	VCSRepoPath string
	RepoSubdir  string
	Config      RepoVersionConfig
	Lock        RepoVersionLock
}

// Repository caching

type RepoConfig struct {
	Path string `yaml:"path"`
}

type CachedRepo struct {
	VCSRepoPath string
	Version     string
	RepoSubdir  string
	Config      RepoConfig
}
