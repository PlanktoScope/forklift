package forklift

// Repos

type VersionedRepo struct {
	VCSRepoPath string
	RepoSubdir  string
	Config      RepoVersionConfig
}

type RepoVersionConfig struct {
	BaseVersion string `yaml:"base-version,omitempty"`
	Timestamp   string `yaml:"timestamp,omitempty"`
	Commit      string `yaml:"commit,omitempty"`
}

// Pkgs

type VersionedPkg struct {
	Path   string
	Repo   VersionedRepo
	Cached CachedPkg
}
