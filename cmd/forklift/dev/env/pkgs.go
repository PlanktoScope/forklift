package env

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-pkg

func lsPkgAction(c *cli.Context) error {
	env, cache, replacementRepos, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	return fcli.PrintEnvPkgs(0, env, cache, replacementRepos)
}

// show-pkg

func showPkgAction(c *cli.Context) error {
	env, cache, replacementRepos, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	pkgPath := c.Args().First()
	return fcli.PrintPkgInfo(0, env, cache, replacementRepos, pkgPath)
}
