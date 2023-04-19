package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift/env"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var defaultWorkspaceBase, _ = os.UserHomeDir()

var app = &cli.App{
	Name:    "forklift",
	Version: "v0.1.0",
	Usage:   "Manages Pallet repositories and package deployments",
	Commands: []*cli.Command{
		envCmd,
		repoCmd,
		pkgCmd,
		deplCmd,
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workspace",
			Aliases: []string{"w"},
			Value:   filepath.Join(defaultWorkspaceBase, ".forklift"),
			Usage:   "Path of the forklift workspace",
			EnvVars: []string{"FORKLIFT_WORKSPACE"},
		},
	},
	Suggest: true,
}

// Env

func downloadRepo(palletsPath string, repo env.Repo) error {
	path := filepath.Join(palletsPath, repo.VCSRepoRelease())
	if workspace.Exists(path) {
		// TODO: perform a disk checksum
		return nil
	}

	fmt.Printf("Downloading %s...\n", repo.VCSRepoRelease())
	gitRepo, err := git.Clone(repo.VCSRepoPath, path)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't clone repo %s to %s", repo.VCSRepoPath, path,
		)
	}
	if repo.Lock.Commit == "" {
		return errors.Errorf("pallet repository %s is not locked at a commit!", repo.Path())
	}
	if err = git.Checkout(gitRepo, repo.Lock.Commit); err != nil {
		return errors.Wrapf(
			err, "couldn't check out commit %s", repo.Lock.Commit,
		)
	}
	if err = os.RemoveAll(filepath.Join(path, ".git")); err != nil {
		return errors.Wrap(err, "couldn't detach from git")
	}
	return nil
}

var envCmd = &cli.Command{
	Name:    "env",
	Aliases: []string{"environment"},
	Usage:   "Manages the local environment",
	Subcommands: []*cli.Command{
		{
			Name:      "clone",
			Usage:     "Initializes the local environment from a remote release",
			ArgsUsage: "[github_repository_path@release]",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "force",
					Aliases: []string{"f"},
					Usage:   "Deletes the local environment if it already exists",
				},
			},
			Action: func(c *cli.Context) error {
				wpath := c.String("workspace")
				if !workspace.Exists(wpath) {
					fmt.Printf("Making a new workspace at %s...", wpath)
				}
				if err := workspace.EnsureExists(wpath); err != nil {
					return errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
				}
				remoteRelease := c.Args().First()
				remote, release, err := git.ParseRemoteRelease(remoteRelease)
				if err != nil {
					return errors.Wrapf(err, "couldn't parse remote release %s", remoteRelease)
				}
				local := workspace.LocalEnvPath(wpath)
				fmt.Printf("Cloning environment %s to %s...\n", remote, local)
				repo, err := git.Clone(remote, local)
				if err != nil {
					if !errors.Is(err, git.ErrRepositoryAlreadyExists) {
						return errors.Wrapf(
							err, "couldn't clone environment %s at release %s to %s", remote, release, local,
						)
					}
					if !c.Bool("force") {
						return errors.Wrap(
							err,
							"you need to first delete your local environment with `forklift env rm` before "+
								"cloning another remote release to it",
						)
					}
					fmt.Printf(
						"Removing local environment from workspace %s, because it already exists and the "+
							"command's --force flag was enabled...\n",
						wpath,
					)
					if err = workspace.RemoveLocalEnv(wpath); err != nil {
						return errors.Wrap(err, "couldn't remove local environment")
					}
					fmt.Printf("Cloning environment %s to %s...\n", remote, local)
					if repo, err = git.Clone(remote, local); err != nil {
						return errors.Wrapf(
							err, "couldn't clone environment %s at release %s to %s", remote, release, local,
						)
					}
				}
				fmt.Printf("Checking out release %s...\n", release)
				if err = git.Checkout(repo, release); err != nil {
					return errors.Wrapf(
						err, "couldn't check out release %s at %s", release, local,
					)
				}
				return nil
			},
		},
		{
			Name:  "fetch",
			Usage: "Updates information about the remote release",
			Action: func(c *cli.Context) error {
				fmt.Println("fetching")
				return nil
			},
		},
		{
			Name:  "pull",
			Usage: "Updates the local environment from the remote release",
			Action: func(c *cli.Context) error {
				fmt.Println("pulling to environment")
				return nil
			},
		},
		{
			Name:  "push",
			Usage: "Updates the remote release from the local environment",
			Action: func(c *cli.Context) error {
				fmt.Println("pushing to remote origin")
				return nil
			},
		},
		{
			Name:    "rm",
			Aliases: []string{"remove"},
			Usage:   "Removes the local environment",
			Action: func(c *cli.Context) error {
				wpath := c.String("workspace")
				fmt.Printf("Removing local environment from workspace %s...\n", wpath)
				return errors.Wrap(workspace.RemoveLocalEnv(wpath), "couldn't remove local environment")
			},
		},
		{
			Name:  "apply",
			Usage: "Updates the Docker Swarm to exactly match the local environment",
			Action: func(c *cli.Context) error {
				fmt.Println("applying local environment")
				return nil
			},
		},
		envRemoteCmd,
	},
}

var envRemoteCmd = &cli.Command{
	Name:  "remote",
	Usage: "Manages the local environment's relationship to the remote source",
	Subcommands: []*cli.Command{
		{
			Name:  "set",
			Usage: "Sets the remote source for the local environment",
			Action: func(c *cli.Context) error {
				fmt.Println("setting remote source to", c.Args().First())
				return nil
			},
		},
	},
}

// Repo

var errMissingEnv = errors.Errorf(
	"you first need to set up a local environment with `forklift env clone`",
)

var repoCmd = &cli.Command{
	Name:    "repo",
	Aliases: []string{"repository"},
	Usage:   "Manages Pallet repositories in the local environment",
	Subcommands: []*cli.Command{
		{
			Name:      "add",
			Usage:     "Adds repositories to the environment, tracking specified versions or branches",
			ArgsUsage: "[pallet_repository_path@release]...",
			Action: func(c *cli.Context) error {
				fmt.Println("adding repositories", c.Args())
				return nil
			},
		},
		{
			Name:    "ls",
			Aliases: []string{"list"},
			Usage:   "Lists repositories which have been added to the environment",
			Action: func(c *cli.Context) error {
				wpath := c.String("workspace")
				if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
					return errMissingEnv
				}
				repos, err := env.ListRepos(workspace.LocalEnvFS(wpath))
				if err != nil {
					return errors.Wrapf(err, "couldn't identify pallet repositories")
				}
				for _, repo := range repos {
					fmt.Printf(
						"%s@%s locked at %s\n", repo.Path(), repo.Config.Release, repo.Lock.Commit,
					)
				}
				return nil
			},
		},
		{
			Name:  "get",
			Usage: "Downloads the repositories specified by the environment",
			Action: func(c *cli.Context) error {
				wpath := c.String("workspace")
				if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
					return errMissingEnv
				}
				fmt.Printf("Getting pallet repositories...\n")
				repos, err := env.ListRepos(workspace.LocalEnvFS(wpath))
				if err != nil {
					return errors.Wrapf(err, "couldn't identify pallet repositories")
				}
				palletsPath := workspace.LocalPalletsPath(wpath)
				for _, repo := range repos {
					if err = downloadRepo(palletsPath, repo); err != nil {
						return errors.Wrapf(
							err, "couldn't download %s at commit %s", repo.Path(), repo.Lock.Commit,
						)
					}
				}
				return nil
			},
		},
		{
			Name:      "rm",
			Aliases:   []string{"remove"},
			Usage:     "Removes a repository from the environment",
			ArgsUsage: "repository_path",
			Action: func(c *cli.Context) error {
				fmt.Println("removing repository", c.Args().First())
				return nil
			},
		},
	},
}

// Pkg

var pkgCmd = &cli.Command{
	Name:    "pkg",
	Aliases: []string{"package"},
	Usage:   "Provides information about Pallet packages available in the local environment",
	Subcommands: []*cli.Command{
		{
			Name:    "ls",
			Aliases: []string{"list"},
			Usage:   "Lists known packages",
			Action: func(c *cli.Context) error {
				fmt.Println("packages:")
				return nil
			},
		},
		{
			Name:      "info",
			Usage:     "Describes a package",
			ArgsUsage: "package_path",
			Action: func(c *cli.Context) error {
				fmt.Println("package", c.Args().First())
				return nil
			},
		},
	},
}

// Depl

var deplCmd = &cli.Command{
	Name:    "depl",
	Aliases: []string{"deployment"},
	Usage:   "Manages Pallet package deployments in the local environment",
	Subcommands: []*cli.Command{
		{
			Name:  "add",
			Usage: "Adds a package deployment to the environment",
			Action: func(c *cli.Context) error {
				fmt.Println("adding package deployment", c.Args().First())
				return nil
			},
		},
		{
			Name:    "ls",
			Aliases: []string{"list"},
			Usage:   "Lists package deployments in the environment",
			Action: func(c *cli.Context) error {
				fmt.Println("package deployments:")
				return nil
			},
		},
		{
			Name:      "rm",
			Aliases:   []string{"remove"},
			Usage:     "Removes a package deployment from the environment",
			ArgsUsage: "package_path",
			Action: func(c *cli.Context) error {
				fmt.Println("removing package deployment", c.Args().First())
				return nil
			},
		},
	},
}
