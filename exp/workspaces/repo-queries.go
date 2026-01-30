package workspaces

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/exp/fs"
)

// A GitRepoQuery holds settings for upgrading the locked version of a Git repo from the
// refs (branches & tags) in its origin.
type GitRepoQuery struct {
	// Path is the path of the pallet or Forklift repo being queried
	// (e.g. github.com/openUC2/rpi-imswitch-os)
	Path string `yaml:"path"`
	// VersionQuery is the version query of the pallet or Forklift repo being queried
	// (e.g. edge or stable or v2024.0.0-beta.0)
	VersionQuery string `yaml:"version-query"`
}

// loadGitRepoQuery loads a GitRepoQuery from the specified file path in the
// provided base filesystem.
func loadGitRepoQuery(fsys ffs.PathedFS, filePath string) (GitRepoQuery, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return GitRepoQuery{}, errors.Wrapf(
			err, "couldn't read git repo query file %s/%s", fsys.Path(), filePath,
		)
	}
	query := GitRepoQuery{}
	if err = yaml.Unmarshal(bytes, &query); err != nil {
		return GitRepoQuery{}, errors.Wrap(err, "couldn't parse git repo query")
	}
	return query, nil
}

func (q GitRepoQuery) Write(outputPath string) error {
	marshaled, err := yaml.Marshal(q)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal stage store state")
	}
	const perm = 0o644 // owner rw, group r, public r
	if err = os.WriteFile(filepath.FromSlash(outputPath), marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save git repo query to %s", outputPath)
	}
	return nil
}

func (q GitRepoQuery) Complete() bool {
	return q.Path != "" && q.VersionQuery != ""
}

func (q GitRepoQuery) String() string {
	return fmt.Sprintf("%s@%s", q.Path, q.VersionQuery)
}

func (q GitRepoQuery) Overlay(r GitRepoQuery) GitRepoQuery {
	result := q
	if r.Path != "" {
		result.Path = r.Path
	}
	if r.VersionQuery != "" {
		result.VersionQuery = r.VersionQuery
	}
	return result
}
