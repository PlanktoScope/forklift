// Package dev provides subcommands for developing packages, repositories, and pallets
package dev

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/cmd/forklift/dev/plt"
)

var defaultWorkingDir, _ = os.Getwd()

func MakeCmd(toolVersion, repoMinVersion, palletMinVersion, newStageVersion string) *cli.Command {
	return &cli.Command{
		Name:    "dev",
		Aliases: []string{"development"},
		Usage:   "Facilitates development and maintenance in the current working directory",
		Subcommands: []*cli.Command{
			plt.MakeCmd(toolVersion, repoMinVersion, palletMinVersion, newStageVersion),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "cwd",
				Value:   defaultWorkingDir,
				Usage:   "Path of the current working directory",
				EnvVars: []string{"FORKLIFT_CWD"},
			},
		},
	}
}
