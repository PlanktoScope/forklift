package forklift

import (
	"io/fs"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
)

func LoadExternalRepo(dir fs.FS, repoConfigFilePath string) (CachedRepo, error) {
	config, err := loadRepoConfig(dir, repoConfigFilePath)
	if err != nil {
		return CachedRepo{}, errors.Wrapf(
			err, "couldn't load cached repo config from %s", repoConfigFilePath,
		)
	}

	repo := CachedRepo{
		ConfigPath: filepath.Dir(repoConfigFilePath),
		Config:     config,
	}
	repo.VCSRepoPath, repo.RepoSubdir, err = SplitRepoPathSubdir(config.Repository.Path)
	if err != nil {
		return CachedRepo{}, errors.Wrapf(
			err, "couldn't parse path of Pallet repo %s", config.Repository.Path,
		)
	}
	return repo, nil
}

func ListExternalRepos(dir fs.FS) ([]CachedRepo, error) {
	repoConfigFiles, err := doublestar.Glob(dir, "**/pallet-repository.yml")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for cached repo configs")
	}

	repoPaths := make([]string, 0, len(repoConfigFiles))
	repoMap := make(map[string]CachedRepo)
	for _, repoConfigFilePath := range repoConfigFiles {
		filename := filepath.Base(repoConfigFilePath)
		if filename != "pallet-repository.yml" {
			continue
		}
		repo, err := LoadExternalRepo(dir, repoConfigFilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached repo from %s", repoConfigFilePath)
		}

		repoPath := repo.Config.Repository.Path
		if prevRepo, ok := repoMap[repoPath]; ok {
			if prevRepo.FromSameVCSRepo(repo) && prevRepo.ConfigPath != repo.ConfigPath {
				return nil, errors.Errorf(
					"repository repeatedly defined in the same version of the same Github repo: %s, %s",
					prevRepo.ConfigPath, repo.ConfigPath,
				)
			}
		}
		repoPaths = append(repoPaths, repoPath)
		repoMap[repoPath] = repo
	}

	orderedRepos := make([]CachedRepo, 0, len(repoPaths))
	for _, path := range repoPaths {
		orderedRepos = append(orderedRepos, repoMap[path])
	}
	return orderedRepos, nil
}
