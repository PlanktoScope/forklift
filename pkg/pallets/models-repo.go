// Package pallets implements the specification for the Pallets package management system
package pallets

const RepoSpecFile = "pallet-repository.yml"

type RepoSpec struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	ReadmeFile  string `yaml:"readme-file"`
}

type RepoConfig struct {
	Repository RepoSpec `yaml:"repository"`
}
