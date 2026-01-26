package versioning

import (
	"time"

	"github.com/pkg/errors"
)

func ToTimestamp(t time.Time) string {
	const Timestamp = "20060102150405"
	return t.UTC().Format(Timestamp)
}

type CommitTimeGetter interface {
	GetCommitTime(hash string) (time.Time, error)
}

func GetCommitTimestamp(c CommitTimeGetter, hash string) (string, error) {
	commitTime, err := c.GetCommitTime(hash)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't check time of commit %s", ShortCommit(hash))
	}
	return ToTimestamp(commitTime), nil
}

func ShortCommit(commit string) string {
	const truncatedLength = 12
	return commit[:truncatedLength]
}
