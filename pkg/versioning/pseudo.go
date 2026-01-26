package versioning

import (
	"fmt"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
)

func Pseudoversion(tag, timestamp, commitHash string) (string, error) {
	// This implements the specification described at https://go.dev/ref/mod#pseudo-versions
	if commitHash == "" {
		return "", errors.Errorf("pseudoversion missing commit hash")
	}
	if timestamp == "" {
		return "", errors.Errorf("pseudoversion missing commit timestamp")
	}
	revisionID := ShortCommit(commitHash)
	if tag == "" {
		return fmt.Sprintf("v0.0.0-%s-%s", timestamp, revisionID), nil
	}
	parsed, err := parseTag(tag)
	if err != nil {
		return "", err
	}
	parsed.Build = nil
	if len(parsed.Pre) > 0 {
		return fmt.Sprintf("v%s.0.%s-%s", parsed.String(), timestamp, revisionID), nil
	}
	return fmt.Sprintf(
		"v%d.%d.%d-0.%s-%s", parsed.Major, parsed.Minor, parsed.Patch+1, timestamp, revisionID,
	), nil
}

func parseTag(v string) (parsed semver.Version, err error) {
	if !strings.HasPrefix(v, "v") {
		return parsed, errors.Errorf("invalid tag `%s` doesn't start with `v`", v)
	}
	if parsed, err = semver.Parse(strings.TrimPrefix(v, "v")); err != nil {
		return parsed, errors.Errorf("tag `%s` couldn't be parsed as a semantic version", v)
	}
	return parsed, nil
}

// Timestamps

func GetCommitTimestamp(c CommitTimeGetter, hash string) (string, error) {
	commitTime, err := c.GetCommitTime(hash)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't check time of commit %s", ShortCommit(hash))
	}
	return ToTimestamp(commitTime), nil
}

type CommitTimeGetter interface {
	GetCommitTime(hash string) (time.Time, error)
}

func ToTimestamp(t time.Time) string {
	const Timestamp = "20060102150405"
	return t.UTC().Format(Timestamp)
}

// Commit hashes

func ShortCommit(commit string) string {
	const truncatedLength = 12
	return commit[:truncatedLength]
}
