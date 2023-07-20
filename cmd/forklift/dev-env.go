package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/app/forklift/dev"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

// CLI

var devEnvCmd = &cli.Command{
	Name:    "env",
	Aliases: []string{"environment"},
	Usage:   "Facilitates development and maintenance of a Forklift environment",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "repo",
			Aliases: []string{"r"},
			Usage: "Replaces version-locked repos from the cache with the corresponding repos in " +
				"the specified directory paths",
		},
	},
	Subcommands: []*cli.Command{
		{
			Name:     "cache-repo",
			Aliases:  []string{"c-r", "cache-repositories"},
			Category: "Use the environment",
			Usage:    "Updates the cache with the repositories available in the development environment",
			Action:   devEnvCacheRepoAction,
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"c-i", "cache-images"},
			Category: "Use the environment",
			Usage:    "Pre-downloads the Docker container images required by the development environment",
			Action:   devEnvCacheImgAction,
		},
		{
			Name:     "check",
			Aliases:  []string{"c"},
			Category: "Use the environment",
			Usage:    "Checks whether the development environment's resource constraints are satisfied",
			Action:   devEnvCheckAction,
		},
		{
			Name:     "plan",
			Aliases:  []string{"p"},
			Category: "Use the environment",
			Usage: "Determines the changes needed to update the Docker Swarm to match the deployments " +
				"specified by the local environment",
			Action: devEnvPlanAction,
		},
		{
			Name:     "apply",
			Aliases:  []string{"a"},
			Category: "Use the environment",
			Usage: "Updates the Docker Swarm to match the deployments specified by the " +
				"development environment",
			Action: devEnvApplyAction,
		},
		{
			Name:     "show",
			Aliases:  []string{"s"},
			Category: "Query the environment",
			Usage:    "Describes the development environment",
			Action:   devEnvShowAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"ls-r", "list-repositories"},
			Category: "Query the environment",
			Usage:    "Lists repositories specified by the environment",
			Action:   devEnvLsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"s-r", "show-repository"},
			Category:  "Query the environment",
			Usage:     "Describes a repository available in the development environment",
			ArgsUsage: "repository_path",
			Action:    devEnvShowRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"ls-p", "list-packages"},
			Category: "Query the environment",
			Usage:    "Lists packages available in the development environment",
			Action:   devEnvLsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"s-p", "show-package"},
			Category:  "Query the environment",
			Usage:     "Describes a package available in the development environment",
			ArgsUsage: "package_path",
			Action:    devEnvShowPkgAction,
		},
		{
			Name:    "ls-depl",
			Aliases: []string{"ls-d", "list-deployments"},
			Usage:   "Lists package deployments specified by the development environment",
			Action:  devEnvLsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"s-d", "show-deployment"},
			Category:  "Query the environment",
			Usage:     "Describes a package deployment specified by the development environment",
			ArgsUsage: "package_path",
			Action:    devEnvShowDeplAction,
		},
		{
			Name:      "add-repo",
			Aliases:   []string{"add-r", "add-repositories"},
			Category:  "Query the environment",
			Usage:     "Adds repositories to the environment, tracking specified versions or branches",
			ArgsUsage: "[pallet_repository_path@version_query]...",
			Action:    devEnvAddRepoAction,
		},
		// TODO: add an upgrade-repo action?
		// {
		// 	Name:      "rm-repo",
		// 	Aliases:   []string{"rm-r", "remove-repositories},
		// 	Category:  "Modify the environment",
		// 	Usage:     "Removes a repository from the environment",
		// 	ArgsUsage: "repository_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing repository", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:  "add-depl",
		// 	Aliases:   []string{"add-d, "add-deployments"},
		// 	Category:  "Modify the environment",
		// 	Usage: "Adds a package deployment to the environment",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding package deployment", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:      "rm-depl",
		// 	Aliases:   []string{"rm-d", "remove-deployments"},
		// 	Category:  "Modify the environment",
		// 	Usage:     "Removes a package deployment from the environment",
		// 	ArgsUsage: "package_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing package deployment", c.Args().First())
		// 		return nil
		// 	},
		// },
	},
}

// show

func devEnvShowAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	return printEnvInfo(0, envPath)
}

// cache-repo

func devEnvCacheRepoAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	wpath := c.String("workspace")

	fmt.Printf("Downloading Pallet repositories specified by the development environment...\n")
	changed, err := downloadRepos(0, envPath, workspace.CachePath(wpath))
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Done! No further actions are needed at this time.")
		return nil
	}

	// TODO: download all Docker images used by packages in the repo - either by inspecting the
	// Docker stack definitions or by allowing packages to list Docker images used.
	fmt.Println(
		// TODO: add a command to check if resource constraints are all satisfied, and show the command
		// to run for that (`forklift dev env check`)
		"Done! Next, you might want to run `sudo -E forklift dev env apply`.",
	)
	return nil
}

// cache-img

func devEnvCacheImgAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	replacementRepos, err := loadReplacementRepos(c.StringSlice("repo"))
	if err != nil {
		return err
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) && len(replacementRepos) == 0 {
		return errMissingCache
	}

	fmt.Println("Downloading Docker container images specified by the development environment...")
	if err := downloadImages(0, envPath, workspace.CachePath(wpath), replacementRepos); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift dev env apply`.")
	return nil
}

func loadReplacementRepos(fsPaths []string) (repos map[string]forklift.ExternalRepo, err error) {
	repos = make(map[string]forklift.ExternalRepo)
	for _, path := range fsPaths {
		replacementPath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
		}
		if !workspace.Exists(replacementPath) {
			return nil, errors.Errorf("couldn't find repository replacement path %s", replacementPath)
		}
		externalRepos, err := forklift.ListExternalRepos(os.DirFS(replacementPath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't list replacement repos in path %s", replacementPath)
		}
		if len(externalRepos) == 0 {
			return nil, errors.Errorf("no replacement repos found in path %s", replacementPath)
		}
		for _, repo := range externalRepos {
			repo.ConfigPath = fmt.Sprintf("%s/%s", replacementPath, repo.ConfigPath)
			repoPath := repo.Path()
			repos[repoPath] = forklift.ExternalRepo{
				FS:   os.DirFS(repo.ConfigPath),
				Repo: repo,
			}
		}
	}
	return repos, nil
}

// check

func devEnvCheckAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	replacementRepos, err := loadReplacementRepos(c.StringSlice("repo"))
	if err != nil {
		return err
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) && len(replacementRepos) == 0 {
		return errMissingCache
	}

	if err := checkEnv(0, envPath, workspace.CachePath(wpath), replacementRepos); err != nil {
		return err
	}
	return nil
}

// plan

func devEnvPlanAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	replacementRepos, err := loadReplacementRepos(c.StringSlice("repo"))
	if err != nil {
		return err
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) && len(replacementRepos) == 0 {
		return errMissingCache
	}

	if err := planEnv(0, envPath, workspace.CachePath(wpath), replacementRepos); err != nil {
		return err
	}
	return nil
}

// apply

func devEnvApplyAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	replacementRepos, err := loadReplacementRepos(c.StringSlice("repo"))
	if err != nil {
		return err
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) && len(replacementRepos) == 0 {
		return errMissingCache
	}

	if err := applyEnv(0, envPath, workspace.CachePath(wpath), replacementRepos); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}

// ls-repo

func devEnvLsRepoAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}

	return printEnvRepos(0, envPath)
}

// show-repo

func devEnvShowRepoAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	replacementRepos, err := loadReplacementRepos(c.StringSlice("repo"))
	if err != nil {
		return err
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) && len(replacementRepos) == 0 {
		return errMissingCache
	}

	repoPath := c.Args().First()
	return printRepoInfo(0, envPath, workspace.CachePath(wpath), replacementRepos, repoPath)
}

// ls-pkg

func devEnvLsPkgAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	replacementRepos, err := loadReplacementRepos(c.StringSlice("repo"))
	if err != nil {
		return err
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) && len(replacementRepos) == 0 {
		return errMissingCache
	}

	return printEnvPkgs(0, envPath, workspace.CachePath(wpath), replacementRepos)
}

// show-pkg

func devEnvShowPkgAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	replacementRepos, err := loadReplacementRepos(c.StringSlice("repo"))
	if err != nil {
		return err
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) && len(replacementRepos) == 0 {
		return errMissingCache
	}

	pkgPath := c.Args().First()
	return printPkgInfo(0, envPath, workspace.CachePath(wpath), replacementRepos, pkgPath)
}

// ls-depl

func devEnvLsDeplAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	replacementRepos, err := loadReplacementRepos(c.StringSlice("repo"))
	if err != nil {
		return err
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) && len(replacementRepos) == 0 {
		return errMissingCache
	}

	return printEnvDepls(0, envPath, workspace.CachePath(wpath), replacementRepos)
}

// show-depl

func devEnvShowDeplAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	replacementRepos, err := loadReplacementRepos(c.StringSlice("repo"))
	if err != nil {
		return err
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) && len(replacementRepos) == 0 {
		return errMissingCache
	}

	deplName := c.Args().First()
	return printDeplInfo(0, envPath, workspace.CachePath(wpath), replacementRepos, deplName)
}

// add-repo

func devEnvAddRepoAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	wpath := c.String("workspace")
	cachePath := workspace.CachePath(wpath)

	remoteReleases := c.Args().Slice()
	if len(remoteReleases) == 0 {
		return errors.Errorf("at least one repository must be specified")
	}

	if err = validateRemoteReleases(remoteReleases); err != nil {
		return errors.Wrap(err, "one or more arguments is invalid")
	}
	fmt.Println("Updating local mirrors of remote Git repos...")
	if err = updateLocalRepoMirrors(remoteReleases, cachePath); err != nil {
		return errors.Wrap(err, "couldn't update local repo mirrors")
	}

	fmt.Println()
	fmt.Println("Resolving version queries...")
	palletRepoConfigs, err := determinePalletRepoConfigs(remoteReleases, cachePath)
	if err != nil {
		return errors.Wrap(err, "couldn't resolve version queries for pallet repos")
	}
	fmt.Println()
	fmt.Printf("Saving configurations to %s...\n", envPath)
	for _, remoteRelease := range remoteReleases {
		config, ok := palletRepoConfigs[remoteRelease]
		if !ok {
			return errors.Errorf("couldn't find configuration for %s", remoteRelease)
		}
		// TODO: write configs as files
		path := filepath.Join(
			envPath, "repositories", config.VCSRepoPath, config.RepoSubdir, "forklift-repo.yml",
		)
		marshaled, err := yaml.Marshal(config.Config)
		if err != nil {
			return errors.Wrapf(err, "couldn't marshal config for %s", path)
		}
		const perm = 0o644 // owner rw, group r, public r
		if err := os.WriteFile(path, marshaled, perm); err != nil {
			return errors.Wrapf(err, "couldn't save config to %s", path)
		}
	}
	fmt.Println("Done!")
	return nil
}

func validateRemoteReleases(remoteReleases []string) error {
	for _, remoteRelease := range remoteReleases {
		_, _, _, err := splitRemoteRepoRelease(remoteRelease)
		if err != nil {
			return errors.Wrapf(err, "'%s' is not a valid argument", remoteRelease)
		}
	}
	return nil
}

func splitRemoteRepoRelease(
	remoteRepoRelease string,
) (vcsRepoPath, repoSubdir, versionQuery string, err error) {
	remote, versionQuery, err := git.ParseRemoteRelease(remoteRepoRelease)
	if err != nil {
		return "", "", "", err
	}
	vcsRepoPath, repoSubdir, err = forklift.SplitRepoPathSubdir(remote)
	if err != nil {
		return "", "", "", err
	}
	return vcsRepoPath, repoSubdir, versionQuery, nil
}

func updateLocalRepoMirrors(remoteReleases []string, cachePath string) error {
	updatedRepos := make(map[string]struct{})
	for _, remoteRelease := range remoteReleases {
		vcsRepoPath, _, _, err := splitRemoteRepoRelease(remoteRelease)
		if err != nil {
			return err
		}
		if _, updated := updatedRepos[vcsRepoPath]; updated {
			continue
		}

		if err = updateLocalRepoMirror(
			vcsRepoPath, filepath.Join(cachePath, vcsRepoPath),
		); err != nil {
			return errors.Wrapf(
				err, "couldn't update local mirror of %s", vcsRepoPath,
			)
		}
		updatedRepos[vcsRepoPath] = struct{}{}
	}
	return nil
}

func updateLocalRepoMirror(remote, cachedPath string) error {
	if _, err := os.Stat(cachedPath); err == nil {
		fmt.Printf("Fetching updates for %s...\n", cachedPath)
		if _, err = git.Fetch(cachedPath); err == nil {
			return err
		}
		fmt.Printf(
			"Warning: couldn't fetch updates in local mirror, will try to re-clone instead: %e\n", err,
		)
		if err = os.RemoveAll(cachedPath); err != nil {
			return errors.Wrapf(
				err, "couldn't remove %s in order to re-clone %s", cachedPath, remote,
			)
		}
	}

	fmt.Printf("Cloning %s to %s...\n", remote, cachedPath)
	_, err := git.CloneMirrored(remote, cachedPath)
	return err
}

func determinePalletRepoConfigs(
	remoteReleases []string, cachePath string,
) (map[string]forklift.VersionedRepo, error) {
	vcsRepoConfigs := make(map[string]forklift.VersionedRepo)
	palletRepoConfigs := make(map[string]forklift.VersionedRepo)
	for _, remoteRelease := range remoteReleases {
		vcsRepoPath, repoSubdir, versionQuery, err := splitRemoteRepoRelease(remoteRelease)
		if err != nil {
			return nil, err
		}
		vcsRepoRelease := fmt.Sprintf("%s@%s", vcsRepoPath, versionQuery)
		if _, configured := vcsRepoConfigs[vcsRepoRelease]; !configured {
			if vcsRepoConfigs[vcsRepoRelease], err = resolveVCSRepoVersionQuery(
				cachePath, vcsRepoPath, versionQuery,
			); err != nil {
				return nil, errors.Wrapf(
					err, "couldn't resolve version query %s for pallet repo %s/%s",
					versionQuery, vcsRepoPath, repoSubdir,
				)
			}
		}

		config := vcsRepoConfigs[vcsRepoRelease]
		config.RepoSubdir = repoSubdir
		versionString, err := config.Config.Version()
		if err != nil {
			return nil, errors.Wrapf(err, "constructed invalid version string from %+v", config.Config)
		}
		fmt.Printf("Resolved %s as %s", remoteRelease, versionString)
		if config.Config.BaseVersion != "" {
			fmt.Printf(", version %s", config.Config.BaseVersion)
		}
		fmt.Println()
		palletRepoConfigs[remoteRelease] = config
	}
	return palletRepoConfigs, nil
}

func resolveVCSRepoVersionQuery(
	cachePath, vcsRepoPath, versionQuery string,
) (forklift.VersionedRepo, error) {
	repo := forklift.VersionedRepo{
		VCSRepoPath: vcsRepoPath,
	}
	if versionQuery == "" {
		return forklift.VersionedRepo{}, errors.New(
			"support for empty version queries is not yet implemented!",
		)
	}
	localPath := filepath.Join(cachePath, vcsRepoPath)
	gitRepo, err := git.Open(localPath)
	if err != nil {
		return forklift.VersionedRepo{}, errors.Wrapf(
			err, "couldn't open local mirror of %s", vcsRepoPath,
		)
	}
	commit, err := queryRefs(gitRepo, versionQuery)
	if err != nil {
		return forklift.VersionedRepo{}, err
	}
	if commit == "" {
		commit, err = gitRepo.GetCommitFullHash(versionQuery)
		if err != nil {
			commit = ""
		}
	}
	if commit == "" {
		return forklift.VersionedRepo{}, errors.Errorf(
			"couldn't find matching commit for '%s' in %s", versionQuery, localPath,
		)
	}
	if repo.Config, err = lockCommit(gitRepo, commit); err != nil {
		return forklift.VersionedRepo{}, err
	}
	return repo, nil
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

func lockCommit(gitRepo *git.Repo, commit string) (config forklift.RepoVersionConfig, err error) {
	config.Commit = commit
	if config.Timestamp, err = getCommitTimestamp(gitRepo, config.Commit); err != nil {
		return forklift.RepoVersionConfig{}, err
	}
	// FIXME: look for a version tagged on the commit, or the last version if it's a pseudoversion.
	// If there's a proper tagged version, write the tag as the base version and write the commit hash
	// but omit the timestamp.
	return config, nil
}
