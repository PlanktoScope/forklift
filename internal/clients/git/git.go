// Package git simplifies git operations
package git

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"
)

func AbbreviateHash(h plumbing.Hash) string {
	const shortHashLength = 7
	return h.String()[:shortHashLength]
}

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

func (r *Repo) makeCheckoutOptions(release string, remote string) git.CheckoutOptions {
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
	if remote != "" {
		return git.CheckoutOptions{
			Branch: plumbing.NewRemoteReferenceName(remote, release),
		}
	}
	return git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(release),
	}
}

func (r *Repo) Checkout(release string, remote string) error {
	worktree, err := r.repository.Worktree()
	if err != nil {
		return err
	}
	checkoutOptions := r.makeCheckoutOptions(release, remote)
	if err = worktree.Checkout(&checkoutOptions); err != nil {
		return err
	}
	return nil
}

func (r *Repo) Refs() (refs []*plumbing.Reference, err error) {
	refsIter, err := r.repository.References()
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't list refs")
	}
	defer refsIter.Close()

	refs = make([]*plumbing.Reference, 0)
	err = refsIter.ForEach(func(ref *plumbing.Reference) error {
		refs = append(refs, ref)
		return nil
	})
	return refs, err
}

func (r *Repo) GetCommitFullHash(commit string) (string, error) {
	hash, err := r.resolveCommit(commit)
	if err != nil {
		return "", errors.Wrapf(
			err, "couldn't resolve %s to a commit hash in the repo", commit,
		)
	}
	return hash.String(), nil
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

type Tag struct {
	Name string
	Hash plumbing.Hash
}

func (t Tag) GetName() string {
	return t.Name
}

func (r *Repo) GetTags() ([]Tag, error) {
	iter, err := r.repository.Tags()
	if err != nil {
		return nil, err
	}
	tags := make([]Tag, 0)
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tags = append(tags, Tag{
			Name: strings.TrimPrefix(string(ref.Name()), "refs/tags/"),
			Hash: ref.Hash(),
		})
		return nil
	})
	return tags, err
}

func (r *Repo) GetTagsAt(commit string) ([]Tag, error) {
	hash, err := r.resolveCommit(commit)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't resolve commit %s", commit)
	}

	allTags, err := r.GetTags()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't list tags on git repo")
	}
	matchingTags := make([]Tag, 0)
	for _, tag := range allTags {
		if tag.Hash != *hash {
			continue
		}
		matchingTags = append(matchingTags, tag)
	}
	return matchingTags, nil
}

type ancestralCommit struct {
	commit *object.Commit
	depth  int
}

type AncestralTag struct {
	Tag
	Depth int
}

func (r *Repo) GetAncestralTags(commit string) ([]AncestralTag, error) {
	tags, err := r.GetTags()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't list tags on git repo")
	}
	taggedCommits := make(map[plumbing.Hash][]Tag)
	for _, tag := range tags {
		taggedCommits[tag.Hash] = append(taggedCommits[tag.Hash], tag)
	}

	hash, err := r.resolveCommit(commit)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't resolve commit %s", commit)
	}
	commitObject, err := r.repository.CommitObject(*hash)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load commit %s", hash)
	}

	// Walk ancestor commits with a breadth-first search, accumulating tagged commits
	visitQueue := []ancestralCommit{{
		commit: commitObject,
		depth:  0,
	}}
	visited := make(map[plumbing.Hash]struct{})
	ancestralTags := make([]AncestralTag, 0)
	for len(visitQueue) > 0 && len(ancestralTags) < len(tags) {
		next := visitQueue[0]
		visitQueue = visitQueue[1:]
		if _, ok := visited[next.commit.Hash]; ok { // we already visited this, so don't revisit it
			continue
		}

		if tags, ok := taggedCommits[next.commit.Hash]; ok {
			for _, tag := range tags {
				ancestralTags = append(ancestralTags, AncestralTag{
					Tag:   tag,
					Depth: next.depth,
				})
			}
		}
		visited[next.commit.Hash] = struct{}{}
		for _, hash := range next.commit.ParentHashes {
			if _, ok := visited[hash]; ok { // we already visited this, so don't enqueue it
				continue
			}

			commitObject, cerr := r.repository.CommitObject(hash)
			if cerr != nil {
				return nil, errors.Wrapf(err, "couldn't load commit %s", hash)
			}
			visitQueue = append(visitQueue, ancestralCommit{
				commit: commitObject,
				depth:  next.depth + 1,
			})
		}
	}
	return ancestralTags, errors.Wrapf(err, "couldn't check for tags ancestral to commit %s", commit)
}

func (r *Repo) MakeTrackingBranches(remoteName string) error {
	// Determine local branches (so we can skip them)
	branches := make(map[string]struct{})
	branchesIter, err := r.repository.References()
	if err != nil {
		return errors.Wrapf(err, "couldn't list refs")
	}
	defer branchesIter.Close()
	refPrefix := "refs/heads/"
	if err = branchesIter.ForEach(func(ref *plumbing.Reference) error {
		refName := ref.Name().String()
		if !strings.HasPrefix(refName, refPrefix) {
			return nil
		}
		branchName := strings.TrimPrefix(refName, refPrefix)
		branches[branchName] = struct{}{}
		return nil
	}); err != nil {
		return err
	}
	// we don't want to make a branch named "HEAD", either:
	branches[string(plumbing.HEAD)] = struct{}{}

	// Determine remote branches
	remote, err := r.repository.Remote(remoteName)
	if err != nil {
		return errors.Wrapf(err, "couldn't open remote %s", remoteName)
	}
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "couldn't list remote refs")
	}

	// Make any missing branches
	for _, ref := range refs {
		refName := ref.Name().String()
		if !strings.HasPrefix(refName, refPrefix) {
			continue
		}
		branchName := strings.TrimPrefix(refName, refPrefix)
		if _, ok := branches[branchName]; ok {
			continue
		}
		if err = r.repository.CreateBranch(&config.Branch{
			Name:   branchName,
			Remote: remoteName,
			Merge:  plumbing.NewBranchReferenceName(branchName),
		}); err != nil {
			return errors.Wrapf(err, "couldn't set up local branch to track remote branch %s", branchName)
		}
	}
	return nil
}

func (r *Repo) FetchAll() error {
	if err := r.repository.Fetch(&git.FetchOptions{
		Progress: os.Stdout,
		Tags:     git.AllTags,
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/heads/*",
		},
	}); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil
		}
		return errors.Wrapf(err, "couldn't fetch changes")
	}
	return nil
}

func (r *Repo) SetRemoteURLs(remoteName string, urls []string) error {
	remote, err := r.repository.Remote(remoteName)
	if err != nil {
		return errors.Wrapf(err, "couldn't open remote %s", remoteName)
	}
	config := remote.Config()
	config.URLs = urls
	if err = r.repository.DeleteRemote(remoteName); err != nil {
		return errors.Wrapf(err, "couldn't delete remote %s", remoteName)
	}
	if _, err = r.repository.CreateRemote(config); err != nil {
		return errors.Wrapf(err, "couldn't delete remote %s", remoteName)
	}
	return nil
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

func Open(local string) (*Repo, error) {
	repo, err := git.PlainOpen(local)
	return &Repo{
		repository: repo,
	}, errors.Wrapf(err, "couldn't open git repo at %s", local)
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

func CloneMirrored(remote, local string) (*Repo, error) {
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
		Mirror:   true,
	})
	return &Repo{
		repository: repo,
	}, errors.Wrapf(err, "couldn't clone git repo %s to %s as a mirror", remote, local)
}

func Status(local string) (status git.Status, err error) {
	repo, err := git.PlainOpen(local)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't open %s as git repo", local)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	return worktree.Status()
}

func Prune(local string) (updated bool, err error) {
	repo, err := git.PlainOpen(local)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't open %s as git repo", local)
	}
	if err = repo.Prune(git.PruneOptions{}); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return false, nil
		}
		return false, errors.Wrapf(err, "couldn't prune repo")
	}
	return true, nil
}

func Fetch(local string) (updated bool, err error) {
	repo, err := git.PlainOpen(local)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't open %s as git repo", local)
	}
	if err = repo.Fetch(&git.FetchOptions{
		Progress: os.Stdout,
		Tags:     git.AllTags,
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/remotes/origin/*",
		},
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

const (
	StatusUnmodified = git.Unmodified
	StatusRenamed    = git.Renamed
)

func EmptyListOptions() *git.ListOptions {
	return &git.ListOptions{}
}
