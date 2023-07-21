package forklift

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Loading

func FindExternalRepoOfPkg(
	repos map[string]ExternalRepo, pkgPath string,
) (repo ExternalRepo, ok bool) {
	repoCandidatePath := filepath.Dir(pkgPath)
	for repoCandidatePath != "." {
		if repo, ok = repos[repoCandidatePath]; ok {
			return repo, true
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
	}
	return ExternalRepo{}, false
}

// Listing

func ListExternalRepos(dir fs.FS) ([]CachedRepo, error) {
	repoConfigFiles, err := doublestar.Glob(dir, fmt.Sprintf("**/%s", pallets.RepoSpecFile))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for cached repo configs")
	}

	repoPaths := make([]string, 0, len(repoConfigFiles))
	repoMap := make(map[string]CachedRepo)
	for _, repoConfigFilePath := range repoConfigFiles {
		if filepath.Base(repoConfigFilePath) != pallets.RepoSpecFile {
			continue
		}
		repo, err := loadExternalRepo(dir, filepath.Dir(repoConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached repo from %s", repoConfigFilePath)
		}

		repoPath := repo.Config.Repository.Path
		if prevRepo, ok := repoMap[repoPath]; ok {
			if prevRepo.FromSameVCSRepo(repo) && prevRepo.FSPath != repo.FSPath {
				return nil, errors.Errorf(
					"repository repeatedly defined in the same version of the same Github repo: %s, %s",
					prevRepo.FSPath, repo.FSPath,
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

func loadExternalRepo(dirFS fs.FS, repoConfigPath string) (CachedRepo, error) {
	fsRepo, err := pallets.LoadFSRepo(dirFS, "", repoConfigPath)
	if err != nil {
		return CachedRepo{}, errors.Wrapf(
			err, "couldn't load external repo config from %s", repoConfigPath,
		)
	}
	repo := CachedRepo{
		FSRepo: fsRepo,
	}
	if repo.VCSRepoPath, repo.Subdir, err = SplitRepoPathSubdir(
		repo.Config.Repository.Path,
	); err != nil {
		return CachedRepo{}, errors.Wrapf(
			err, "couldn't parse path of Pallet repo %s", repo.Config.Repository.Path,
		)
	}
	return repo, nil
}
