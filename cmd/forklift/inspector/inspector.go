package inspector

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/forklift-run/forklift/internal/app/forklift"
	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// resolve-git-repo

func resolveGitRepoAction(c *cli.Context) error {
	workspace, err := ensureWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}

	query := c.Args().First()
	resolved, err := fcli.ResolveQueriesUsingLocalMirrors(
		0, workspace.GetMirrorCachePath(), []string{query}, true,
	)
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve query %s", query)
	}
	fmt.Println(resolved[query].VersionLock.Version)
	return nil
}

func ensureWorkspace(wpath string) (*forklift.FSWorkspace, error) {
	if !ffs.DirExists(wpath) {
		fmt.Fprintf(os.Stderr, "Making a new workspace at %s...", wpath)
	}
	if err := ffs.EnsureExists(wpath); err != nil {
		return nil, errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
	}
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	if err = ffs.EnsureExists(workspace.GetDataPath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", workspace.GetDataPath())
	}
	return workspace, nil
}
