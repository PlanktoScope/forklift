package forklift

// A GitRepoQuery holds settings for upgrading the locked version of a Git repo from the
// refs (branches & tags) in its origin.
type GitRepoQuery struct {
	// Path is the path of the pallet or Forklift repo being queried
	// (e.g. github.com/PlanktoScope/pallet-standard)
	Path string `yaml:"path"`
	// VersionQuery is the version query of the pallet or Forklift repo being queried
	// (e.g. edge or stable or v2024.0.0-beta.0)
	VersionQuery string `yaml:"version-query"`
}
