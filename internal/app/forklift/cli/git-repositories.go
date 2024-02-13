package cli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

func DownloadGitRepos(
	indent int, cachePath string, queries []string,
) (changed bool, err error) {
	if err = ValidateGitRepoQueries(queries); err != nil {
		return false, errors.Wrap(err, "one or more arguments is invalid")
	}
	IndentedPrintln(indent, "Updating local mirrors of remote Git repos...")
	if err = UpdateLocalGitRepoMirrors(indent, queries, cachePath); err != nil {
		return false, errors.Wrap(err, "couldn't update local Git repo mirrors")
	}

	fmt.Println()
	IndentedPrintln(indent, "Resolving version queries...")
	reqs, err := ResolveGitRepoQueries(queries, cachePath)
	if err != nil {
		return false, errors.Wrap(err, "couldn't resolve version queries for repos")
	}
	changed = false
	for _, req := range reqs {
		downloaded, err := DownloadGitRepo(indent, cachePath, req.Path(), req.VersionLock)
		if err != nil {
			return changed, errors.Wrapf(
				err, "couldn't download %s@%s as commit %s",
				req.Path(), req.VersionLock.Version, req.VersionLock.Def.ShortCommit(),
			)
		}
		if !downloaded {
			IndentedPrintf(
				indent, "Didn't download %s@%s because it already exists\n",
				req.Path(), req.VersionLock.Version,
			)
		}
		changed = changed || downloaded
	}
	return changed, nil
}

func ValidateGitRepoQueries(queries []string) error {
	if len(queries) == 0 {
		return errors.Errorf("at least one query must be specified")
	}
	for _, query := range queries {
		if _, _, ok := strings.Cut(query, "@"); !ok {
			return errors.Errorf("couldn't parse query '%s' as path@version", query)
		}
	}
	return nil
}

func UpdateLocalGitRepoMirrors(indent int, queries []string, cachePath string) error {
	allUpdated := make(map[string]struct{})
	for _, query := range queries {
		p, _, ok := strings.Cut(query, "@")
		if !ok {
			return errors.Errorf("couldn't parse query '%s' as path@version", query)
		}
		if _, updated := allUpdated[p]; updated {
			continue
		}

		if err := updateLocalGitRepoMirror(indent, p, path.Join(cachePath, p)); err != nil {
			return errors.Wrapf(err, "couldn't update local mirror of %s", p)
		}
		allUpdated[p] = struct{}{}
	}
	return nil
}

func updateLocalGitRepoMirror(indent int, remote, cachedPath string) error {
	remote = filepath.FromSlash(remote)
	cachedPath = filepath.FromSlash(cachedPath)
	if _, err := os.Stat(cachedPath); err == nil {
		IndentedPrintf(indent, "Fetching updates for %s...\n", cachedPath)
		if _, err = git.Fetch(cachedPath); err == nil {
			return err
		}
		IndentedPrintf(
			indent, "Warning: couldn't fetch updates in local mirror, will try to re-clone instead: %e\n",
			err,
		)
		if err = os.RemoveAll(cachedPath); err != nil {
			return errors.Wrapf(err, "couldn't remove %s in order to re-clone %s", cachedPath, remote)
		}
	}

	IndentedPrintf(indent, "Cloning %s to %s...\n", remote, cachedPath)
	_, err := git.CloneMirrored(remote, cachedPath)
	return err
}

func ResolveGitRepoQueries(
	queries []string, cachePath string,
) (resolved map[string]forklift.GitRepoReq, err error) {
	resolved = make(map[string]forklift.GitRepoReq)
	for _, query := range queries {
		if _, ok := resolved[query]; ok {
			continue
		}
		gitRepoPath, versionQuery, ok := strings.Cut(query, "@")
		if !ok {
			return nil, errors.Errorf("couldn't parse '%s' as git_repo_path@version", query)
		}
		req := forklift.GitRepoReq{
			RequiredPath: gitRepoPath,
		}
		if req.VersionLock, err = resolveVersionQuery(
			cachePath, gitRepoPath, versionQuery,
		); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't resolve version query %s for git repo %s", versionQuery, gitRepoPath,
			)
		}

		fmt.Printf("Resolved %s as %+v", query, req.VersionLock.Version)
		fmt.Println()
		resolved[query] = req
	}
	return resolved, nil
}

func resolveVersionQuery(
	cachePath, gitRepoPath, versionQuery string,
) (lock forklift.VersionLock, err error) {
	if versionQuery == "" {
		return forklift.VersionLock{}, errors.New("empty version queries are not yet supported")
	}
	localPath := filepath.FromSlash(path.Join(cachePath, gitRepoPath))
	gitRepo, err := git.Open(localPath)
	if err != nil {
		return forklift.VersionLock{}, errors.Wrapf(
			err, "couldn't open local mirror of %s", gitRepoPath,
		)
	}
	commit, err := queryRefs(gitRepo, versionQuery)
	if err != nil {
		return forklift.VersionLock{}, err
	}
	if commit == "" {
		commit, err = gitRepo.GetCommitFullHash(versionQuery)
		if err != nil {
			commit = ""
		}
	}
	if commit == "" {
		return forklift.VersionLock{}, errors.Errorf(
			"couldn't find matching commit for '%s' in %s", versionQuery, localPath,
		)
	}
	if lock.Def, err = lockCommit(gitRepo, commit); err != nil {
		return forklift.VersionLock{}, err
	}
	if lock.Version, err = lock.Def.Version(); err != nil {
		return forklift.VersionLock{}, err
	}
	return lock, nil
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

	// Attempt to lock as a tagged version
	tags, err := gitRepo.GetTagsAt(commit)
	if err != nil {
		return forklift.VersionLockDef{}, errors.Wrapf(err, "couldn't lookup tags matching %s", commit)
	}
	tags = filterTags(tags)
	sort.Slice(tags, func(i, j int) bool {
		return semver.Compare(tags[i].Name, tags[j].Name) > 0
	})
	if len(tags) > 0 {
		config.Tag = tags[0].Name
		config.Type = forklift.LockTypeVersion
		return config, nil
	}

	// Lock as a pseudoversion
	config.Type = forklift.LockTypePseudoversion
	ancestralTags, err := gitRepo.GetAncestralTags(commit)
	if err != nil {
		return forklift.VersionLockDef{}, errors.Wrapf(
			err, "couldn't determine tagged ancestors of %s", commit,
		)
	}
	ancestralTags = filterTags(ancestralTags)
	sort.Slice(ancestralTags, func(i, j int) bool {
		return semver.Compare(ancestralTags[i].Name, ancestralTags[j].Name) > 0
	})
	if len(ancestralTags) > 0 {
		config.Tag = ancestralTags[0].Name
	}
	return config, nil
}

type nameGetter interface {
	GetName() string
}

func filterTags[T nameGetter](tags []T) []T {
	filtered := make([]T, 0, len(tags))
	for _, tag := range tags {
		if !semver.IsValid(tag.GetName()) {
			continue
		}
		filtered = append(filtered, tag)
	}
	return filtered
}

func DownloadGitRepo(
	indent int, cachePath string, gitRepoPath string, lock forklift.VersionLock,
) (downloaded bool, err error) {
	if !lock.Def.IsCommitLocked() {
		return false, errors.Errorf(
			"the version lock definition for Git repo %s has no commit lock", gitRepoPath,
		)
	}
	version := lock.Version
	gitRepoCachePath := filepath.Join(
		filepath.FromSlash(cachePath), fmt.Sprintf("%s@%s", filepath.FromSlash(gitRepoPath), version),
	)
	if forklift.Exists(gitRepoCachePath) {
		// TODO: perform a disk checksum
		return false, nil
	}

	IndentedPrintf(indent, "Downloading %s@%s...\n", gitRepoPath, version)
	gitRepo, err := git.Clone(gitRepoPath, gitRepoCachePath)
	if err != nil {
		return false, errors.Wrapf(
			err, "couldn't clone git repo %s to %s", gitRepoPath, gitRepoCachePath,
		)
	}

	// Validate commit
	shortCommit := lock.Def.ShortCommit()
	if err = validateCommit(lock, gitRepo); err != nil {
		// TODO: this should instead be a Clear method on a WritableFS at that path
		if cerr := os.RemoveAll(gitRepoCachePath); cerr != nil {
			IndentedPrintf(
				indent,
				"Error: couldn't clean up %s after failed validation! You'll need to delete it yourself.\n",
				gitRepoCachePath,
			)
		}
		return false, errors.Wrapf(
			err, "commit %s for git repo %s failed version validation", shortCommit, gitRepoPath,
		)
	}

	// Checkout commit
	if err = gitRepo.Checkout(lock.Def.Commit, ""); err != nil {
		if cerr := os.RemoveAll(gitRepoCachePath); cerr != nil {
			IndentedPrintf(
				indent, "Error: couldn't clean up %s! You'll need to delete it yourself.\n",
				gitRepoCachePath,
			)
		}
		return false, errors.Wrapf(err, "couldn't check out commit %s", shortCommit)
	}
	if err = os.RemoveAll(filepath.Join(gitRepoCachePath, ".git")); err != nil {
		return false, errors.Wrap(err, "couldn't detach from git")
	}
	return true, nil
}

func validateCommit(versionLock forklift.VersionLock, gitRepo *git.Repo) error {
	// Check commit time
	commitTimestamp, err := forklift.GetCommitTimestamp(gitRepo, versionLock.Def.Commit)
	if err != nil {
		return err
	}
	versionedTimestamp := versionLock.Def.Timestamp
	if commitTimestamp != versionedTimestamp {
		return errors.Errorf(
			"commit %s was made at %s, while the repo version lock definition expects it to have "+
				"been made at %s",
			versionLock.Def.ShortCommit(), commitTimestamp, versionedTimestamp,
		)
	}

	// TODO: implement remaining checks specified in https://go.dev/ref/mod#pseudo-versions
	// (if base version is specified, there must be a corresponding semantic version tag that is an
	// ancestor of the revision described by the pseudo-version; and the revision must be an ancestor
	// of one of the module repository's branches or tags)
	return nil
}
