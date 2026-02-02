package forklift

import (
	"sort"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/forklift-run/forklift/exp/versioning"
	"github.com/forklift-run/forklift/internal/clients/git"
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
	if l.Decl, err = LockCommit(gitRepo, commit); err != nil {
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

func LockCommit(gitRepo *git.Repo, commit string) (d versioning.LockDecl, err error) {
	d.Commit = commit
	if d.Timestamp, err = versioning.GetCommitTimestamp(gitRepo, d.Commit); err != nil {
		return d, err
	}

	// Attempt to lock as a tagged version
	tags, err := gitRepo.GetTagsAt(commit)
	if err != nil {
		return d, errors.Wrapf(err, "couldn't lookup tags matching %s", commit)
	}
	tags = filterTags(tags)
	sort.Slice(tags, func(i, j int) bool {
		return semver.Compare(tags[i].Name, tags[j].Name) > 0
	})
	if len(tags) > 0 {
		d.Tag = tags[0].Name
		d.Type = versioning.LockTypeVersion
		return d, nil
	}

	// Lock as a pseudoversion
	d.Type = versioning.LockTypePseudoversion
	ancestralTags, err := gitRepo.GetAncestralTags(commit)
	if err != nil {
		return d, errors.Wrapf(
			err, "couldn't determine tagged ancestors of %s", commit,
		)
	}
	ancestralTags = filterTags(ancestralTags)
	sort.Slice(ancestralTags, func(i, j int) bool {
		return semver.Compare(ancestralTags[i].Name, ancestralTags[j].Name) > 0
	})
	if len(ancestralTags) > 0 {
		d.Tag = ancestralTags[0].Name
	}
	return d, nil
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
