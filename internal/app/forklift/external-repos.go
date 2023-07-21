package forklift

import (
	"fmt"
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

func ListExternalRepos(fsys pallets.PathedFS) ([]pallets.FSRepo, error) {
	repoConfigFiles, err := doublestar.Glob(fsys, fmt.Sprintf("**/%s", pallets.RepoSpecFile))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for cached repo configs")
	}

	repoPaths := make([]string, 0, len(repoConfigFiles))
	repoMap := make(map[string]pallets.FSRepo)
	for _, repoConfigFilePath := range repoConfigFiles {
		if filepath.Base(repoConfigFilePath) != pallets.RepoSpecFile {
			continue
		}
		repo, err := loadExternalRepo(fsys, filepath.Dir(repoConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached repo from %s", repoConfigFilePath)
		}

		repoPath := repo.Config.Repository.Path
		if prevRepo, ok := repoMap[repoPath]; ok {
			if prevRepo.FromSameVCSRepo(repo.Repo) && prevRepo.FS.Path() != repo.FS.Path() {
				return nil, errors.Errorf(
					"repository repeatedly defined in the same version of the same Github repo: %s, %s",
					prevRepo.FS.Path(), repo.FS.Path(),
				)
			}
		}
		repoPaths = append(repoPaths, repoPath)
		repoMap[repoPath] = repo
	}

	orderedRepos := make([]pallets.FSRepo, 0, len(repoPaths))
	for _, path := range repoPaths {
		orderedRepos = append(orderedRepos, repoMap[path])
	}
	return orderedRepos, nil
}

func loadExternalRepo(fsys pallets.PathedFS, subdirPath string) (pallets.FSRepo, error) {
	repo, err := pallets.LoadFSRepo(fsys, subdirPath)
	if err != nil {
		return pallets.FSRepo{}, errors.Wrapf(
			err, "couldn't load external repo config from %s", subdirPath,
		)
	}
	// TODO: move SplitRepoPathSubdir into pkg/pallets
	if repo.VCSRepoPath, repo.Subdir, err = SplitRepoPathSubdir(
		repo.Config.Repository.Path,
	); err != nil {
		return pallets.FSRepo{}, errors.Wrapf(
			err, "couldn't parse path of Pallet repo %s", repo.Config.Repository.Path,
		)
	}
	return repo, nil
}
