package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/internal/clients/git"
)

// Resolving version query

func ResolveVersionQueryUsingRepo(
	localPath, versionQuery string,
) (lock forklift.VersionLock, err error) {
	if versionQuery == "" {
		return forklift.VersionLock{}, errors.New("empty version queries are not yet supported")
	}

	gitRepo, err := git.Open(localPath)
	if err != nil {
		return forklift.VersionLock{}, errors.Wrapf(err, "couldn't open %s as a git repo", localPath)
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
	if lock.Decl, err = lockCommit(gitRepo, commit); err != nil {
		return forklift.VersionLock{}, err
	}
	if lock.Version, err = lock.Decl.Version(); err != nil {
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

func lockCommit(gitRepo *git.Repo, commit string) (config forklift.VersionLockDecl, err error) {
	config.Commit = commit
	if config.Timestamp, err = forklift.GetCommitTimestamp(gitRepo, config.Commit); err != nil {
		return forklift.VersionLockDecl{}, err
	}

	// Attempt to lock as a tagged version
	tags, err := gitRepo.GetTagsAt(commit)
	if err != nil {
		return forklift.VersionLockDecl{}, errors.Wrapf(err, "couldn't lookup tags matching %s", commit)
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
		return forklift.VersionLockDecl{}, errors.Wrapf(
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

// Resolving multiple version queries

func ResolveQueriesUsingLocalMirrors(
	indent int, mirrorsPath string, queries []string, updateLocalMirror bool,
) (resolved map[string]forklift.GitRepoReq, err error) {
	IndentedFprintln(
		indent, os.Stderr, "Resolving version queries using local mirrors of remote Git repos...",
	)
	indent++
	resolved, err = resolveGitRepoQueriesUsingLocalMirrors(indent, queries, mirrorsPath)
	if err != nil {
		if !updateLocalMirror {
			return resolved, errors.Wrap(
				err, "couldn't resolve one or more version queries, and we're not updating local mirrors",
			)
		}
		IndentedFprintln(
			indent, os.Stderr,
			"Warning: couldn't resolve one or more version queries, so we'll update local mirrors of "+
				"remote Git repos and try again",
		)

		IndentedFprintln(indent, os.Stderr, "Updating local mirrors of remote Git repos...")
		if err = updateQueriedLocalGitRepoMirrors(indent+1, queries, mirrorsPath); err != nil {
			return nil, errors.Wrap(err, "couldn't update local Git repo mirrors")
		}
		IndentedFprintln(indent, os.Stderr, "Resolving version queries from updated local mirrors...")
		if resolved, err = resolveGitRepoQueriesUsingLocalMirrors(
			indent+1, queries, mirrorsPath,
		); err != nil {
			return nil, errors.Wrap(err, "couldn't resolve version queries for repos")
		}
		return resolved, err
	}

	if !updateLocalMirror {
		return resolved, nil
	}

	performOptionalLocalMirrorsUpdate(indent, queries, mirrorsPath)
	IndentedFprintln(indent, os.Stderr, "Resolving version queries from updated local mirrors...")
	newResolved, err := resolveGitRepoQueriesUsingLocalMirrors(indent, queries, mirrorsPath)
	if err != nil {
		IndentedFprintln(
			indent, os.Stderr,
			"Warning: couldn't resolve version query with updated local mirror, falling back to "+
				"previous value",
		)
	}
	return newResolved, nil
}

func updateQueriedLocalGitRepoMirrors(indent int, queries []string, mirrorsPath string) error {
	allUpdated := make(map[string]struct{})
	for _, query := range queries {
		p, _, ok := strings.Cut(query, "@")
		if !ok {
			return errors.Errorf("couldn't parse query '%s' as path@version", query)
		}
		if _, updated := allUpdated[p]; updated {
			continue
		}

		if err := updateLocalGitRepoMirror(indent, p, path.Join(mirrorsPath, p)); err != nil {
			return errors.Wrapf(err, "couldn't update local mirror of %s", p)
		}
		allUpdated[p] = struct{}{}
	}
	return nil
}

func updateLocalGitRepoMirror(indent int, remote, mirrorPath string) error {
	remote = filepath.FromSlash(remote)
	mirrorPath = filepath.FromSlash(mirrorPath)
	if _, err := os.Stat(mirrorPath); errors.Is(err, fs.ErrNotExist) {
		IndentedFprintf(indent, os.Stderr, "Cloning %s to local mirror...\n", remote)
		_, err := git.CloneMirrored(indent+1, remote, mirrorPath, os.Stderr)
		return err
	}
	gitRepo, err := git.Open(mirrorPath)
	if err != nil {
		return errors.Errorf("couldn't open local mirror of %s at %s", remote, mirrorPath)
	}
	return gitRepo.FetchAll(indent+1, os.Stdout)
}

func resolveGitRepoQueriesUsingLocalMirrors(
	indent int, queries []string, mirrorsPath string,
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
		if req.VersionLock, err = ResolveVersionQueryUsingRepo(
			filepath.FromSlash(path.Join(mirrorsPath, gitRepoPath)), versionQuery,
		); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't resolve version query %s for git repo %s", versionQuery, gitRepoPath,
			)
		}

		IndentedFprintf(indent, os.Stderr, "Resolved %s as %+v\n", query, req.VersionLock.Version)
		resolved[query] = req
	}
	return resolved, nil
}

func performOptionalLocalMirrorsUpdate(indent int, queries []string, mirrorsPath string) {
	IndentedFprintln(
		indent, os.Stderr,
		"Updating local mirrors of remote Git repos (even though it's not required)...",
	)
	indent++
	if err := updateQueriedLocalGitRepoMirrors(indent+1, queries, mirrorsPath); err != nil {
		IndentedFprintln(
			indent, os.Stderr,
			"Warning: couldn't update local mirrors (do you have internet access? does the remote repo "+
				"actually exist?):",
		)
		IndentedFprintln(indent+1, os.Stderr, err)
		IndentedFprintln(
			indent, os.Stderr,
			"We might not even need updates of our local mirrors, so we'll continue anyways!",
		)
	}
}

// Downloading to cache

func DownloadQueriedGitReposUsingLocalMirrors(
	indent int, mirrorsPath, cachePath string, queries []string,
) (resolved map[string]forklift.GitRepoReq, changed map[forklift.GitRepoReq]bool, err error) {
	if err = validateGitRepoQueries(queries); err != nil {
		return nil, nil, errors.Wrap(err, "one or more arguments is invalid")
	}
	if resolved, err = ResolveQueriesUsingLocalMirrors(
		indent, mirrorsPath, queries, true,
	); err != nil {
		return nil, nil, err
	}

	changed = make(map[forklift.GitRepoReq]bool)
	for _, req := range resolved {
		downloaded, err := cloneLockedGitRepoFromLocalMirror(
			indent, cachePath, mirrorsPath, req.Path(), req.VersionLock,
		)
		if err != nil {
			return resolved, nil, errors.Wrapf(
				err, "couldn't download %s@%s as commit %s",
				req.Path(), req.VersionLock.Version, req.VersionLock.Decl.ShortCommit(),
			)
		}
		if !downloaded {
			IndentedFprintf(
				indent, os.Stderr,
				"Didn't download %s@%s because it already exists\n", req.Path(), req.VersionLock.Version,
			)
		}
		changed[req] = true
	}
	return resolved, changed, nil
}

func validateGitRepoQueries(queries []string) error {
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

func DownloadLockedGitRepoUsingLocalMirror(
	indent int, mirrorsPath, cachePath, gitRepoPath string, lock forklift.VersionLock,
) (downloaded bool, err error) {
	if err := forklift.EnsureExists(mirrorsPath); err != nil {
		return false, errors.Wrap(err, "couldn't ensure existence of mirrors cache")
	}
	mirrorPath := filepath.Join(filepath.FromSlash(mirrorsPath), gitRepoPath)
	downloaded, err = cloneLockedGitRepoFromLocalMirror(
		indent, cachePath, mirrorsPath, gitRepoPath, lock,
	)
	if err != nil {
		indent++
		IndentedFprintln(
			indent, os.Stderr,
			"Couldn't clone from local mirror, so we'll update from the remote Git repo and try again...",
		)
		if err = updateLocalGitRepoMirror(indent, gitRepoPath, mirrorPath); err != nil {
			return false, errors.Wrap(err, "couldn't update local Git repo mirrors")
		}
		if downloaded, err = cloneLockedGitRepoFromLocalMirror(
			indent, cachePath, mirrorsPath, gitRepoPath, lock,
		); err != nil {
			return false, errors.Wrapf(
				err, "couldn't clone repo %s at version %s", gitRepoPath, lock.Version,
			)
		}
		return downloaded, nil
	}

	if !downloaded {
		IndentedFprintf(indent, os.Stderr, "%s@%s was already downloaded!\n", gitRepoPath, lock.Version)
	}
	performOptionalLocalMirrorsUpdate(indent, []string{gitRepoPath + "@" + lock.Version}, mirrorsPath)
	return downloaded, nil
}

func cloneLockedGitRepoFromLocalMirror(
	indent int, cachePath, mirrorsPath, gitRepoPath string, lock forklift.VersionLock,
) (downloaded bool, err error) {
	if !lock.Decl.IsCommitLocked() {
		return false, errors.Errorf(
			"the version lock definition for Git repo %s has no commit lock", gitRepoPath,
		)
	}
	version := lock.Version
	gitRepoCachePath := filepath.FromSlash(path.Join(
		cachePath, fmt.Sprintf("%s@%s", gitRepoPath, version),
	))
	if forklift.DirExists(gitRepoCachePath) {
		// TODO: perform a disk checksum
		return false, nil
	}

	mirrorCachePath := filepath.Join(filepath.FromSlash(mirrorsPath), filepath.FromSlash(gitRepoPath))
	IndentedFprintln(indent, os.Stderr, "Cloning from local mirror...")
	gitRepo, err := git.Clone(
		indent+1, fmt.Sprintf("file://%s", mirrorCachePath), gitRepoCachePath, io.Discard,
	)
	if err != nil {
		return false, errors.Wrapf(
			err, "couldn't clone git repo %s from %s to %s",
			gitRepoPath, mirrorCachePath, gitRepoCachePath,
		)
	}

	// Validate commit
	shortCommit := lock.Decl.ShortCommit()
	if err = validateCommit(lock, gitRepo); err != nil {
		// TODO: this should instead be a Clear method on a WritableFS at that path
		if cerr := os.RemoveAll(gitRepoCachePath); cerr != nil {
			IndentedFprintf(
				indent, os.Stderr,
				"Error: couldn't clean up %s after failed validation! You'll need to delete it yourself.\n",
				gitRepoCachePath,
			)
		}
		return false, errors.Wrapf(
			err, "commit %s for git repo %s failed version validation", shortCommit, gitRepoPath,
		)
	}

	// Checkout commit
	if err = gitRepo.Checkout(lock.Decl.Commit, ""); err != nil {
		if cerr := os.RemoveAll(gitRepoCachePath); cerr != nil {
			IndentedFprintf(
				indent, os.Stderr, "Error: couldn't clean up %s! You'll need to delete it yourself.\n",
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
	commitTimestamp, err := forklift.GetCommitTimestamp(gitRepo, versionLock.Decl.Commit)
	if err != nil {
		return err
	}
	versionedTimestamp := versionLock.Decl.Timestamp
	if commitTimestamp != versionedTimestamp {
		return errors.Errorf(
			"commit %s was made at %s, while the repo version lock definition expects it to have "+
				"been made at %s",
			versionLock.Decl.ShortCommit(), commitTimestamp, versionedTimestamp,
		)
	}

	// TODO: implement remaining checks specified in https://go.dev/ref/mod#pseudo-versions
	// (if base version is specified, there must be a corresponding semantic version tag that is an
	// ancestor of the revision described by the pseudo-version; and the revision must be an ancestor
	// of one of the module repository's branches or tags)
	return nil
}

// Cloning to local copy

const (
	OriginRemoteName              = "origin"
	ForkliftCacheMirrorRemoteName = "forklift-cache-mirror"
)

func CloneQueriedGitRepoUsingLocalMirror(
	indent int, mirrorsPath, gitRepoPath, versionQuery, destination string,
	updateLocalMirror bool,
) error {
	if _, err := ResolveQueriesUsingLocalMirrors(
		indent, mirrorsPath, []string{gitRepoPath + "@" + versionQuery}, updateLocalMirror,
	); err != nil {
		return err
	}

	if _, err := os.Stat(destination); err == nil || !errors.Is(err, fs.ErrNotExist) {
		return errors.Errorf("%s already exists!", destination)
	}

	mirrorCachePath := filepath.Join(filepath.FromSlash(mirrorsPath), filepath.FromSlash(gitRepoPath))
	IndentedFprintf(
		indent, os.Stderr, "Cloning %s to %s via local mirror...\n", gitRepoPath, destination,
	)
	gitRepo, err := git.Clone(
		indent+1, fmt.Sprintf("file://%s", mirrorCachePath), destination, os.Stderr,
	)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't clone git repo %s from %s to %s", gitRepoPath, mirrorCachePath, destination,
		)
	}
	if err = gitRepo.MakeTrackingBranches(OriginRemoteName); err != nil {
		return errors.Wrapf(err, "couldn't set up local branches to track the remote")
	}
	if err = gitRepo.FetchAll(indent+1, os.Stdout); err != nil {
		return errors.Wrapf(err, "couldn't fetch new local branches tracking the remote")
	}
	if err = gitRepo.SetRemoteURLs(
		OriginRemoteName, []string{fmt.Sprintf("https://%s", gitRepoPath)},
	); err != nil {
		return errors.Wrapf(err, "couldn't set the correct URL of the origin remote")
	}
	if err = gitRepo.CreateRemote(
		ForkliftCacheMirrorRemoteName, []string{mirrorCachePath},
	); err != nil {
		return errors.Wrapf(err, "couldn't add a remote for the local mirror")
	}

	IndentedFprintf(indent, os.Stderr, "Checking out %s in %s...\n", versionQuery, destination)
	if err = gitRepo.Checkout(versionQuery, ""); err != nil {
		if cerr := os.RemoveAll(destination); cerr != nil {
			IndentedFprintf(
				indent, os.Stderr,
				"Error: couldn't clean up %s! You'll need to delete it yourself.\n", destination,
			)
		}
		return errors.Wrapf(err, "couldn't check out version query %s", versionQuery)
	}
	return nil
}
