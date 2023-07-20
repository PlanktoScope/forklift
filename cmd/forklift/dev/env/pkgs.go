package env

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
)

// ls-pkg

func lsPkgAction(c *cli.Context) error {
	envPath, wpath, replacementRepos, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	return fcli.PrintEnvPkgs(0, envPath, workspace.CachePath(wpath), replacementRepos)
}

// show-pkg

func showPkgAction(c *cli.Context) error {
	envPath, wpath, replacementRepos, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	pkgPath := c.Args().First()
	return fcli.PrintPkgInfo(0, envPath, workspace.CachePath(wpath), replacementRepos, pkgPath)
}
