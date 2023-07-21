package pallets

import (
	"io/fs"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// FSRepo

// LoadFSRepo loads a FSRepo from the specified directory path in the provided base filesystem.
// The base path should correspond to the location of the base filesystem. In the loaded FSRepo's
// embedded [Repo], the Pallet repository path is not initialized, nor is the Pallet repository
// subdirectory initialized.
func LoadFSRepo(baseFS fs.FS, baseFSPath, repoFSPath string) (p FSRepo, err error) {
	p = FSRepo{
		FSPath: filepath.Join(baseFSPath, repoFSPath),
	}
	if p.FS, err = fs.Sub(baseFS, repoFSPath); err != nil {
		return FSRepo{}, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", repoFSPath, baseFSPath,
		)
	}
	if p.Repo.Config, err = LoadRepoConfig(p.FS, p.FSPath, RepoSpecFile); err != nil {
		return FSRepo{}, errors.Wrapf(err, "couldn't load repo config")
	}
	return p, nil
}

// RepoConfig

// LoadRepoConfig loads a RepoConfig from the specified file path in the provided base filesystem.
// The base path should correspond to the location of the base filesystem.
func LoadRepoConfig(baseFS fs.FS, baseFSPath, filePath string) (RepoConfig, error) {
	bytes, err := fs.ReadFile(baseFS, filePath)
	if err != nil {
		return RepoConfig{}, errors.Wrapf(
			err, "couldn't read repo config file %s/%s", baseFSPath, filePath,
		)
	}
	config := RepoConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return RepoConfig{}, errors.Wrap(err, "couldn't parse repo config")
	}
	return config, nil
}

func (c RepoConfig) Check() (errs []error) {
	return ErrsWrap(c.Repository.Check(), "invalid repo spec")
}

// RepoSpec

func (s RepoSpec) Check() (errs []error) {
	if s.Path == "" {
		errs = append(errs, errors.Errorf("repo spec is missing `path` parameter"))
	}
	return errs
}
