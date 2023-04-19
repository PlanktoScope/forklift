// Package git simplifies git operations
package git

import (
	"net/url"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
)

func makeCheckoutOptions(repo *git.Repository, release string) git.CheckoutOptions {
	if plumbing.IsHash(release) {
		return git.CheckoutOptions{
			Hash: plumbing.Hash([]byte(release)),
		}
	}
	if hash, err := repo.ResolveRevision(plumbing.Revision(release)); err != nil {
		if strings.HasPrefix(hash.String(), release) {
			return git.CheckoutOptions{
				Hash: *hash,
			}
		}
	}
	if semver.IsValid(release) {
		return git.CheckoutOptions{
			Branch: plumbing.NewTagReferenceName(release),
		}
	}
	return git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(release),
	}
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

func Clone(remote, local string) (*git.Repository, error) {
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
	return repo, errors.Wrapf(err, "couldn't clone git repo %s to %s", remote, local)
}

func Checkout(repo *git.Repository, release string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}
	checkoutOptions := makeCheckoutOptions(repo, release)
	if err = worktree.Checkout(&checkoutOptions); err != nil {
		return err
	}
	return nil
}
