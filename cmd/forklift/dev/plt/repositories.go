package plt

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

// cache-repo

func cacheRepoAction(c *cli.Context) error {
	pallet, cache, _, err := processFullBaseArgs(c, false)
	if err != nil {
		return err
	}

	fmt.Printf("Downloading repos specified by the development pallet...\n")
	changed, err := fcli.DownloadRepos(0, pallet, cache.Underlay)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Done! No further actions are needed at this time.")
		return nil
	}

	fmt.Println("Done! Next, you might want to run `sudo -E forklift dev plt apply`.")
	return nil
}

// ls-repo

func lsRepoAction(c *cli.Context) error {
	pallet, err := getPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintPalletRepos(0, pallet)
}

// show-repo

func showRepoAction(c *cli.Context) error {
	pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
		return err
	}

	repoPath := c.Args().First()
	return fcli.PrintRepoInfo(0, pallet, cache, repoPath)
}

// add-repo

func addRepoAction(c *cli.Context) error {
	pallet, cache, _, err := processFullBaseArgs(c, false)
	if err != nil {
		return err
	}

	repoQueries := c.Args().Slice()
	if err = validateRepoQueries(repoQueries); err != nil {
		return errors.Wrap(err, "one or more arguments is invalid")
	}
	fmt.Println("Updating local mirrors of remote Git repos...")
	if err = updateLocalRepoMirrors(repoQueries, cache.Underlay.Path()); err != nil {
		return errors.Wrap(err, "couldn't update local repo mirrors")
	}

	fmt.Println()
	fmt.Println("Resolving version queries...")
	repoReqs, err := resolveRepoQueries(repoQueries, cache.Underlay.Path())
	if err != nil {
		return errors.Wrap(err, "couldn't resolve version queries for repos")
	}
	fmt.Println()
	fmt.Printf("Saving configurations to %s...\n", pallet.FS.Path())
	for _, repoQuery := range repoQueries {
		repoReq, ok := repoReqs[repoQuery]
		if !ok {
			return errors.Errorf("couldn't find configuration for %s", repoQuery)
		}
		reqsReposFS, err := pallet.GetRepoReqsFS()
		if err != nil {
			return err
		}
		repoReqPath := path.Join(reqsReposFS.Path(), repoReq.Path(), forklift.VersionLockDefFile)
		marshaled, err := yaml.Marshal(repoReq.VersionLock.Def)
		if err != nil {
			return errors.Wrapf(err, "couldn't marshal repo requirement from %s", repoReqPath)
		}
		if err := forklift.EnsureExists(filepath.FromSlash(path.Dir(repoReqPath))); err != nil {
			return errors.Wrapf(
				err, "couldn't make directory %s", filepath.FromSlash(path.Dir(repoReqPath)),
			)
		}
		const perm = 0o644 // owner rw, group r, public r
		if err := os.WriteFile(filepath.FromSlash(repoReqPath), marshaled, perm); err != nil {
			return errors.Wrapf(
				err, "couldn't save repo requirement to %s", filepath.FromSlash(repoReqPath),
			)
		}
	}
	fmt.Println("Done!")
	return nil
}

func validateRepoQueries(repoQueries []string) error {
	if len(repoQueries) == 0 {
		return errors.Errorf("at least one repo must be specified")
	}
	for _, repoQuery := range repoQueries {
		if _, _, ok := strings.Cut(repoQuery, "@"); !ok {
			return errors.Errorf("couldn't parse '%s' as repo_path@version", repoQuery)
		}
	}
	return nil
}

func updateLocalRepoMirrors(repoQueries []string, cachePath string) error {
	updatedRepos := make(map[string]struct{})
	for _, repoQuery := range repoQueries {
		repoPath, _, ok := strings.Cut(repoQuery, "@")
		if !ok {
			return errors.Errorf("couldn't parse '%s' as repo_path@version", repoQuery)
		}
		if _, updated := updatedRepos[repoPath]; updated {
			continue
		}

		if err := updateLocalRepoMirror(repoPath, path.Join(cachePath, repoPath)); err != nil {
			return errors.Wrapf(err, "couldn't update local mirror of %s", repoPath)
		}
		updatedRepos[repoPath] = struct{}{}
	}
	return nil
}

func updateLocalRepoMirror(remote, cachedPath string) error {
	remote = filepath.FromSlash(remote)
	cachedPath = filepath.FromSlash(cachedPath)
	if _, err := os.Stat(cachedPath); err == nil {
		fmt.Printf("Fetching updates for %s...\n", cachedPath)
		if _, err = git.Fetch(cachedPath); err == nil {
			return err
		}
		fmt.Printf(
			"Warning: couldn't fetch updates in local mirror, will try to re-clone instead: %e\n", err,
		)
		if err = os.RemoveAll(cachedPath); err != nil {
			return errors.Wrapf(err, "couldn't remove %s in order to re-clone %s", cachedPath, remote)
		}
	}

	fmt.Printf("Cloning %s to %s...\n", remote, cachedPath)
	_, err := git.CloneMirrored(remote, cachedPath)
	return err
}

func resolveRepoQueries(
	repoQueries []string, cachePath string,
) (map[string]forklift.RepoReq, error) {
	resolved := make(map[string]forklift.RepoReq)
	for _, repoQuery := range repoQueries {
		if _, ok := resolved[repoQuery]; ok {
			continue
		}
		repoPath, versionQuery, ok := strings.Cut(repoQuery, "@")
		if !ok {
			return nil, errors.Errorf("couldn't parse '%s' as repo_path@version", repoQuery)
		}
		repoReq, err := resolveRepoVersionQuery(cachePath, repoPath, versionQuery)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't resolve version query %s for repo %s", versionQuery, repoPath,
			)
		}

		fmt.Printf("Resolved %s as %+v", repoQuery, repoReq.VersionLock.Version)
		if repoReq.VersionLock.Def.BaseVersion != "" {
			fmt.Printf(", version %s", repoReq.VersionLock.Def.BaseVersion)
		}
		fmt.Println()
		resolved[repoQuery] = repoReq
	}
	return resolved, nil
}

func resolveRepoVersionQuery(cachePath, repoPath, versionQuery string) (forklift.RepoReq, error) {
	req := forklift.RepoReq{
		RepoPath: repoPath,
	}
	if versionQuery == "" {
		return forklift.RepoReq{}, errors.New("empty version queries are not yet supported")
	}
	localPath := filepath.FromSlash(path.Join(cachePath, repoPath))
	gitRepo, err := git.Open(localPath)
	if err != nil {
		return forklift.RepoReq{}, errors.Wrapf(err, "couldn't open local mirror of %s", repoPath)
	}
	commit, err := queryRefs(gitRepo, versionQuery)
	if err != nil {
		return forklift.RepoReq{}, err
	}
	if commit == "" {
		commit, err = gitRepo.GetCommitFullHash(versionQuery)
		if err != nil {
			commit = ""
		}
	}
	if commit == "" {
		return forklift.RepoReq{}, errors.Errorf(
			"couldn't find matching commit for '%s' in %s", versionQuery, localPath,
		)
	}
	if req.VersionLock.Def, err = lockCommit(gitRepo, commit); err != nil {
		return forklift.RepoReq{}, err
	}
	if req.VersionLock.Version, err = req.VersionLock.Def.Version(); err != nil {
		return forklift.RepoReq{}, err
	}
	return req, nil
}

func queryRefs(gitRepo *git.Repo, versionQuery string) (commit string, err error) {
	refs, err := gitRepo.Refs()
	if err != nil {
		return "", err
	}
	for _, ref := range refs {
		if ref.Name().Short() != versionQuery {
			continue
		}

		if ref.Type() != git.HashReference {
			return "", errors.New("only hash references are supported")
		}
		return ref.Hash().String(), nil
	}
	return "", nil
}

func lockCommit(gitRepo *git.Repo, commit string) (config forklift.VersionLockDef, err error) {
	config.Commit = commit
	if config.Timestamp, err = forklift.GetCommitTimestamp(gitRepo, config.Commit); err != nil {
		return forklift.VersionLockDef{}, err
	}
	// FIXME: look for a version tagged on the commit, or the last version if it's a pseudoversion.
	// If there's a proper tagged version, write the tag as the base version and write the commit hash
	// but omit the timestamp.
	return config, nil
}
