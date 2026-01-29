package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/internal/clients/git"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/versioning"
)

// Resolving version query

func ResolveVersionQueryUsingRepo(
	localPath, versionQuery string,
) (l versioning.Lock, err error) {
	if versionQuery == "" {
		return l, errors.New("empty version queries are not yet supported")
	}

	gitRepo, err := git.Open(localPath)
	if err != nil {
		return l, errors.Wrapf(err, "couldn't open %s as a git repo", localPath)
	}
	commit, err := queryRefs(gitRepo, versionQuery)
	if err != nil {
		return l, err
	}
	if commit == "" {
		commit, err = gitRepo.GetCommitFullHash(versionQuery)
		if err != nil {
			commit = ""
		}
	}
	if commit == "" {
		return l, errors.Errorf(
			"couldn't find matching commit for '%s' in %s", versionQuery, localPath,
		)
	}
	if l.Decl, err = forklift.LockCommit(gitRepo, commit); err != nil {
		return l, err
	}
	if l.Version, err = l.Decl.Version(); err != nil {
		return l, err
	}
	return l, nil
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

// Resolving multiple version queries

func ResolveQueriesUsingLocalMirrors(
	indent int, mirrorsPath string, queries []string, updateLocalMirror bool,
) (resolved map[string]fplt.GitRepoReq, err error) {
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
) (resolved map[string]fplt.GitRepoReq, err error) {
	resolved = make(map[string]fplt.GitRepoReq)
	for _, query := range queries {
		if _, ok := resolved[query]; ok {
			continue
		}
		gitRepoPath, versionQuery, ok := strings.Cut(query, "@")
		if !ok {
			return nil, errors.Errorf("couldn't parse '%s' as git_repo_path@version", query)
		}
		req := fplt.GitRepoReq{
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
) (resolved map[string]fplt.GitRepoReq, changed map[fplt.GitRepoReq]bool, err error) {
	if err = forklift.ValidateGitRepoQueries(queries); err != nil {
		return nil, nil, errors.Wrap(err, "one or more arguments is invalid")
	}
	if resolved, err = ResolveQueriesUsingLocalMirrors(
		indent, mirrorsPath, queries, true,
	); err != nil {
		return nil, nil, err
	}

	changed = make(map[fplt.GitRepoReq]bool)
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

func DownloadLockedGitRepoUsingLocalMirror(
	indent int, mirrorsPath, cachePath, gitRepoPath string, lock versioning.Lock,
) (downloaded bool, err error) {
	if err := ffs.EnsureExists(mirrorsPath); err != nil {
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
	indent int, cachePath, mirrorsPath, gitRepoPath string, lock versioning.Lock,
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
	if ffs.DirExists(gitRepoCachePath) {
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
	if err = forklift.ValidateCommit(lock, gitRepo); err != nil {
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

// Cloning to local copy

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
	if err = gitRepo.MakeTrackingBranches(forklift.OriginRemoteName); err != nil {
		return errors.Wrapf(err, "couldn't set up local branches to track the remote")
	}
	if err = gitRepo.FetchAll(indent+1, os.Stdout); err != nil {
		return errors.Wrapf(err, "couldn't fetch new local branches tracking the remote")
	}
	if err = gitRepo.SetRemoteURLs(
		forklift.OriginRemoteName, []string{fmt.Sprintf("https://%s", gitRepoPath)},
	); err != nil {
		return errors.Wrapf(err, "couldn't set the correct URL of the origin remote")
	}
	if err = gitRepo.CreateRemote(
		forklift.ForkliftCacheMirrorRemoteName, []string{mirrorCachePath},
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
