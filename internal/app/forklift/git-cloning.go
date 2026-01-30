package forklift

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/exp/versioning"
	"github.com/forklift-run/forklift/internal/clients/git"
)

// Downloading to cache

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

func ValidateCommit(versionLock versioning.Lock, gitRepo *git.Repo) error {
	// Check commit time
	commitTimestamp, err := versioning.GetCommitTimestamp(gitRepo, versionLock.Decl.Commit)
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
