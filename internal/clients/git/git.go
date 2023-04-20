// Package git simplifies git operations
package git

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"
)

type Repo struct {
	repository *git.Repository
}

func (r *Repo) resolveCommit(commit string) (*plumbing.Hash, error) {
	hash, err := r.repository.ResolveRevision(plumbing.Revision(commit))
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't resolve %s as a commit in the repo", commit)
	}
	if strings.HasPrefix(hash.String(), commit) {
		return hash, nil
	}
	return nil, errors.Errorf("%s appears to be a non-commit revision name", commit)
}

func (r *Repo) makeCheckoutOptions(release string) git.CheckoutOptions {
	if plumbing.IsHash(release) {
		return git.CheckoutOptions{
			Hash: plumbing.NewHash(release),
		}
	}
	if hash, err := r.resolveCommit(release); err == nil {
		return git.CheckoutOptions{
			Hash: *hash,
		}
	}
	if _, err := semver.Parse(
		strings.TrimPrefix(release, "v"),
	); strings.HasPrefix(release, "v") && err == nil {
		return git.CheckoutOptions{
			Branch: plumbing.NewTagReferenceName(release),
		}
	}
	return git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(release),
	}
}

func (r *Repo) Checkout(release string) error {
	worktree, err := r.repository.Worktree()
	if err != nil {
		return err
	}
	checkoutOptions := r.makeCheckoutOptions(release)
	if err = worktree.Checkout(&checkoutOptions); err != nil {
		return err
	}
	return nil
}

func (r *Repo) GetCommitTime(commit string) (time.Time, error) {
	hash, err := r.resolveCommit(commit)
	if err != nil {
		return time.Time{}, errors.Wrapf(
			err, "couldn't resolve %s to a commit hash in the repo", commit,
		)
	}
	object, err := r.repository.CommitObject(*hash)
	if err != nil {
		return time.Time{}, errors.Wrapf(
			err, "couldn't find commit object with hash %s", hash.String(),
		)
	}
	return object.Committer.When, nil
}

var ErrRepositoryAlreadyExists = git.ErrRepositoryAlreadyExists

func ParseRemoteRelease(remoteRelease string) (remote, release string, err error) {
	remote, release, ok := strings.Cut(remoteRelease, "@")
	if !ok {
		return "", "", errors.Errorf(
			"remote release %s needs to be of format git_repository_path@release", remoteRelease,
		)
	}
	if remote == "" {
		return "", "", errors.Errorf(
			"remote release %s is missing a remote git repository path", remoteRelease,
		)
	}
	if release == "" {
		return "", "", errors.Errorf(
			"remote release %s is missing a release (a version, branch name, full commit hash, or "+
				"abbreviated commit hash)",
			remoteRelease,
		)
	}
	return remote, release, nil
}

func Clone(remote, local string) (*Repo, error) {
	u, err := url.Parse(remote)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse %s as a url", remote)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	remote = u.String()
	repo, err := git.PlainClone(local, false, &git.CloneOptions{
		URL:      remote,
		Progress: os.Stdout,
	})
	return &Repo{
		repository: repo,
	}, errors.Wrapf(err, "couldn't clone git repo %s to %s", remote, local)
}

func Fetch(local string) (updated bool, err error) {
	repo, err := git.PlainOpen(local)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't open %s as git repo", local)
	}
	if err = repo.Fetch(&git.FetchOptions{
		Progress: os.Stdout,
		Tags:     git.AllTags,
	}); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return false, nil
		}
		return false, errors.Wrapf(err, "couldn't fetch changes")
	}
	return true, nil
}

func Pull(local string) (updated bool, err error) {
	repo, err := git.PlainOpen(local)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't open %s as git repo", local)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return false, err
	}
	if err = worktree.Pull(&git.PullOptions{
		Progress: os.Stdout,
	}); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return false, nil
		}
		return false, errors.Wrapf(err, "couldn't fast-forward to remote")
	}
	return true, nil
}
