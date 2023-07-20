package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	dct "github.com/docker/cli/cli/compose/types"
	ggit "github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

// CLI

var envCmd = &cli.Command{
	Name:    "env",
	Aliases: []string{"environment"},
	Usage:   "Manages the local environment",
	Subcommands: []*cli.Command{
		{
			Name:      "clone",
			Category:  "Modify the environment",
			Usage:     "Initializes the local environment from a remote release",
			ArgsUsage: "[github_repository_path@release]",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "force",
					Aliases: []string{"f"},
					Usage:   "Deletes the local environment if it already exists",
				},
			},
			Action: envCloneAction,
		},
		{
			Name:     "fetch",
			Category: "Modify the environment",
			Usage:    "Updates information about the remote release",
			Action:   envFetchAction,
		},
		{
			Name:     "pull",
			Category: "Modify the environment",
			Usage:    "Fast-forwards the local environment to match the remote release",
			Action:   envPullAction,
		},
		// {
		// 	Name:  "push",
		// 	Category:  "Modify the environment",
		// 	Usage: "Updates the remote release from the local environment",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("pushing to remote origin")
		// 		return nil
		// 	},
		// },
		{
			Name:     "rm",
			Aliases:  []string{"remove"},
			Category: "Modify the environment",
			Usage:    "Removes the local environment",
			Action:   envRmAction,
		},
		// envRemoteCmd,
		{
			Name:     "cache-repo",
			Aliases:  []string{"c-r", "cache-repositories"},
			Category: "Use the environment",
			Usage:    "Updates the cache with the repositories available in the local environment",
			Action:   envCacheRepoAction,
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"c-i", "cache-images"},
			Category: "Use the environment",
			Usage:    "Pre-downloads the Docker container images required by the local environment",
			Action:   envCacheImgAction,
		},
		{
			Name:     "check",
			Aliases:  []string{"c"},
			Category: "Use the environment",
			Usage:    "Checks whether the local environment's resource constraints are satisfied",
			Action:   envCheckAction,
		},
		{
			Name:     "plan",
			Aliases:  []string{"p"},
			Category: "Use the environment",
			Usage: "Determines the changes needed to update the Docker Swarm to match the deployments " +
				"specified by the local environment",
			Action: envPlanAction,
		},
		{
			Name:     "apply",
			Aliases:  []string{"a"},
			Category: "Use the environment",
			Usage: "Updates the Docker Swarm to match the deployments specified by the " +
				"local environment",
			Action: envApplyAction,
		},
		{
			Name:     "show",
			Aliases:  []string{"s"},
			Category: "Query the environment",
			Usage:    "Describes the local environment",
			Action:   envShowAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"ls-r", "list-repositories"},
			Category: "Query the environment",
			Usage:    "Lists repositories available in the local environment",
			Action:   envLsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"s-r", "show-repository"},
			Category:  "Query the environment",
			Usage:     "Describes a repository available in the local environment",
			ArgsUsage: "repository_path",
			Action:    envShowRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"ls-p", "list-packages"},
			Category: "Query the environment",
			Usage:    "Lists packages available in the local environment",
			Action:   envLsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"s-p", "show-package"},
			Category:  "Query the environment",
			Usage:     "Describes a package available in the local environment",
			ArgsUsage: "package_path",
			Action:    envShowPkgAction,
		},
		{
			Name:     "ls-depl",
			Aliases:  []string{"ls-d", "list-deployments"},
			Category: "Query the environment",
			Usage:    "Lists package deployments specified by the local environment",
			Action:   envLsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"s-d", "show-deployment"},
			Category:  "Query the environment",
			Usage:     "Describes a package deployment specified by the local environment",
			ArgsUsage: "package_path",
			Action:    envShowDeplAction,
		},
		// {
		// 	Name:      "add-repo",
		// 	Aliases:   []string{"add-r", "add-repositories"},
		// 	Category:  "Modify the environment",
		// 	Usage:     "Adds repositories to the environment, tracking specified versions or branches",
		// 	ArgsUsage: "[pallet_repository_path@version_query]...",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding repositories", c.Args())
		// 		// TODO: implement version queries - see https://go.dev/ref/mod#vcs-branch
		// 		return nil
		// 	},
		// },
		// TODO: add an upgrade-repo action
		// {
		// 	Name:      "rm-repo",
		// 	Aliases:   []string{"rm-r", "remove-repositories"},
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
		// 	Aliases:   []string{"add-d", "add-deployments"},
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

// var envRemoteCmd = &cli.Command{
// 	Name:  "remote",
// 	Usage: "Manages the local environment's relationship to the remote source",
// 	Subcommands: []*cli.Command{
// 		{
// 			Name:  "set",
// 			Usage: "Sets the remote source for the local environment",
// 			Action: func(c *cli.Context) error {
// 				fmt.Println("setting remote source to", c.Args().First())
// 				return nil
// 			},
// 		},
// 	},
// }

// Errors

var errMissingEnv = errors.Errorf(
	"you first need to set up a local environment with `forklift env clone`",
)

var errMissingCache = errors.Errorf(
	"you first need to cache the repos specified by your environment with " +
		"`forklift env cache-repo` or `forklift dev env cache-repo`",
)

// clone

func envCloneAction(c *cli.Context) error {
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
	gitRepo, err := git.Clone(remote, local)
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
		if gitRepo, err = git.Clone(remote, local); err != nil {
			return errors.Wrapf(
				err, "couldn't clone environment %s at release %s to %s", remote, release, local,
			)
		}
	}
	fmt.Printf("Checking out release %s...\n", release)
	if err = gitRepo.Checkout(release); err != nil {
		return errors.Wrapf(
			err, "couldn't check out release %s at %s", release, local,
		)
	}
	fmt.Println("Done! Next, you'll probably want to run `forklift env cache-repo`.")
	return nil
}

// fetch

func envFetchAction(c *cli.Context) error {
	wpath := c.String("workspace")
	envPath := workspace.LocalEnvPath(wpath)
	if !workspace.Exists(envPath) {
		return errMissingEnv
	}

	fmt.Println("Fetching updates...")
	updated, err := git.Fetch(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't fetch changes from the remote release")
	}
	if !updated {
		fmt.Println("No updates from the remote release.")
	}
	// TODO: display changes
	return nil
}

// pull

func envPullAction(c *cli.Context) error {
	wpath := c.String("workspace")
	envPath := workspace.LocalEnvPath(wpath)
	if !workspace.Exists(envPath) {
		return errMissingEnv
	}

	fmt.Println("Attempting to fast-forward the local environment...")
	updated, err := git.Pull(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't fast-forward the local environment")
	}
	if !updated {
		fmt.Println("No changes from the remote release.")
	}
	// TODO: display changes
	return nil
}

// rm

func envRmAction(c *cli.Context) error {
	wpath := c.String("workspace")
	fmt.Printf("Removing local environment from workspace %s...\n", wpath)
	// TODO: return an error if there are uncommitted or unpushed changes to be removed - in which
	// case require a --force flag
	return errors.Wrap(workspace.RemoveLocalEnv(wpath), "couldn't remove local environment")
}

// info

func envShowAction(c *cli.Context) error {
	wpath := c.String("workspace")
	envPath := workspace.LocalEnvPath(wpath)
	if !workspace.Exists(envPath) {
		return errMissingEnv
	}
	return printEnvInfo(0, envPath)
}

func printEnvInfo(indent int, envPath string) error {
	fcli.IndentedPrintf(indent, "Environment: %s\n", envPath)
	config, err := forklift.LoadEnvConfig(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't load the environment config")
	}

	fcli.IndentedPrintf(indent, "Description: %s\n", config.Environment.Description)

	ref, err := git.Head(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its HEAD", envPath)
	}
	fcli.IndentedPrintf(indent, "Currently on: %s\n", git.StringifyRef(ref))
	// TODO: report any divergence between head and remotes
	if err := printUncommittedChanges(indent+1, envPath); err != nil {
		return err
	}

	fmt.Println()
	if err := printLocalRefsInfo(indent, envPath); err != nil {
		return err
	}
	fmt.Println()
	if err := printRemotesInfo(indent, envPath); err != nil {
		return err
	}
	return nil
}

func printRemotesInfo(indent int, envPath string) error {
	remotes, err := git.Remotes(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its remotes", envPath)
	}

	fcli.IndentedPrintf(indent, "Remotes:")
	if len(remotes) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, remote := range remotes {
		printRemoteInfo(indent, remote)
	}
	return nil
}

func printRemoteInfo(indent int, remote *ggit.Remote) {
	config := remote.Config()
	fcli.IndentedPrintf(indent, "%s:\n", config.Name)
	indent++

	fcli.IndentedPrintf(indent, "URLs:")
	if len(config.URLs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for i, url := range config.URLs {
		fcli.BulletedPrintf(indent+1, "%s: ", url)
		if i == 0 {
			fmt.Print("fetch, ")
		}
		fmt.Println("push")
	}

	fcli.IndentedPrintf(indent, "Up-to-date references:")
	refs, err := remote.List(git.EmptyListOptions())
	if err != nil {
		fmt.Printf(" (couldn't retrieve references: %s)\n", err)
		return
	}

	if len(refs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, ref := range refs {
		fcli.BulletedPrintf(indent+1, "%s\n", git.StringifyRef(ref))
	}
}

func printLocalRefsInfo(indent int, envPath string) error {
	refs, err := git.Refs(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its refs", envPath)
	}

	fcli.IndentedPrintf(indent, "References:")
	if len(refs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, ref := range refs {
		fcli.BulletedPrintf(indent, "%s\n", git.StringifyRef(ref))
	}

	return nil
}

func printUncommittedChanges(indent int, envPath string) error {
	status, err := git.Status(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query the environment %s for its status", envPath)
	}
	fcli.IndentedPrint(indent, "Uncommitted changes:")
	if len(status) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for file, status := range status {
		if status.Staging == git.StatusUnmodified && status.Worktree == git.StatusUnmodified {
			continue
		}
		if status.Staging == git.StatusRenamed {
			file = fmt.Sprintf("%s -> %s", file, status.Extra)
		}
		fcli.BulletedPrintf(indent, "%c%c %s\n", status.Staging, status.Worktree, file)
	}
	return nil
}

// cache-repo

func envCacheRepoAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		return errMissingEnv
	}

	fmt.Println("Downloading Pallet repositories specified by the local environment...")
	changed, err := downloadRepos(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath))
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Done! No further actions are needed at this time.")
		return nil
	}
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift env apply`.")
	return nil
}

func downloadRepos(indent int, envPath, cachePath string) (changed bool, err error) {
	repos, err := forklift.ListVersionedRepos(os.DirFS(envPath))
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	changed = false
	for _, repo := range repos {
		downloaded, err := downloadRepo(indent, cachePath, repo)
		changed = changed || downloaded
		if err != nil {
			return false, errors.Wrapf(
				err, "couldn't download %s at commit %s", repo.Path(), repo.Config.ShortCommit(),
			)
		}
	}
	return changed, nil
}

func downloadRepo(
	indent int, palletsPath string, repo forklift.VersionedRepo,
) (downloaded bool, err error) {
	if !repo.Config.IsCommitLocked() {
		return false, errors.Errorf(
			"the local environment's versioning config for repository %s has no commit lock", repo.Path(),
		)
	}
	vcsRepoPath := repo.VCSRepoPath
	version, err := repo.Config.Version()
	if err != nil {
		return false, errors.Wrapf(err, "couldn't determine version for %s", vcsRepoPath)
	}
	path := filepath.Join(palletsPath, fmt.Sprintf("%s@%s", repo.VCSRepoPath, version))
	if workspace.Exists(path) {
		// TODO: perform a disk checksum
		return false, nil
	}

	fcli.IndentedPrintf(indent, "Downloading %s@%s...\n", repo.VCSRepoPath, version)
	gitRepo, err := git.Clone(vcsRepoPath, path)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't clone repo %s to %s", vcsRepoPath, path)
	}

	// Validate commit
	shortCommit := repo.Config.ShortCommit()
	if err = validateCommit(repo, gitRepo); err != nil {
		if cerr := os.RemoveAll(path); cerr != nil {
			fcli.IndentedPrintf(
				indent,
				"Error: couldn't clean up %s after failed validation! You'll need to delete it yourself.\n",
				path,
			)
		}
		return false, errors.Wrapf(
			err, "commit %s for github repo %s failed repo version validation", shortCommit, vcsRepoPath,
		)
	}

	// Checkout commit
	if err = gitRepo.Checkout(repo.Config.Commit); err != nil {
		if cerr := os.RemoveAll(path); cerr != nil {
			fcli.IndentedPrintf(
				indent, "Error: couldn't clean up %s! You will need to delete it yourself.\n", path,
			)
		}
		return false, errors.Wrapf(err, "couldn't check out commit %s", shortCommit)
	}
	if err = os.RemoveAll(filepath.Join(path, ".git")); err != nil {
		return false, errors.Wrap(err, "couldn't detach from git")
	}
	return true, nil
}

func validateCommit(versionedRepo forklift.VersionedRepo, gitRepo *git.Repo) error {
	// Check commit time
	commitTimestamp, err := getCommitTimestamp(gitRepo, versionedRepo.Config.Commit)
	if err != nil {
		return err
	}
	versionedTimestamp := versionedRepo.Config.Timestamp
	if commitTimestamp != versionedTimestamp {
		return errors.Errorf(
			"commit %s was made at %s, while the repository versioning config file expects it to have "+
				"been made at %s",
			versionedRepo.Config.ShortCommit(), commitTimestamp, versionedTimestamp,
		)
	}

	// TODO: implement remaining checks specified in https://go.dev/ref/mod#pseudo-versions
	// (if base version is specified, there must be a corresponding semantic version tag that is an
	// ancestor of the revision described by the pseudo-version; and the revision must be an ancestor
	// of one of the module repository's branches or tags)
	return nil
}

func getCommitTimestamp(gitRepo *git.Repo, hash string) (string, error) {
	commitTime, err := gitRepo.GetCommitTime(hash)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't check time of commit %s", forklift.ShortCommit(hash))
	}
	return forklift.ToTimestamp(commitTime), nil
}

// cache-img

func envCacheImgAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		return errMissingEnv
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		return errMissingCache
	}

	fmt.Println("Downloading Docker container images specified by the local environment...")
	if err := downloadImages(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil,
	); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift env apply`.")
	return nil
}

func downloadImages(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
) error {
	orderedImages, err := listRequiredImages(indent, envPath, cachePath, replacementRepos)
	if err != nil {
		return errors.Wrap(err, "couldn't determine images required by package deployments")
	}

	dc, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}
	for _, image := range orderedImages {
		fmt.Println()
		fcli.IndentedPrintf(indent, "Downloading %s...\n", image)
		pulled, err := dc.PullImage(context.Background(), image, docker.NewOutStream(os.Stdout))
		if err != nil {
			return errors.Wrapf(err, "couldn't download %s", image)
		}
		fcli.IndentedPrintf(
			indent, "Downloaded %s from %s\n", pulled.Reference(), pulled.RepoInfo().Name,
		)
	}
	return nil
}

func listRequiredImages(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
) ([]string, error) {
	cacheFS := os.DirFS(cachePath)
	depls, err := forklift.ListDepls(os.DirFS(envPath), cacheFS, replacementRepos)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}
	orderedImages := make([]string, 0, len(depls))
	images := make(map[string]struct{})
	for _, depl := range depls {
		fcli.IndentedPrintf(
			indent, "Checking Docker container images used by package deployment %s...\n", depl.Name,
		)
		if depl.Pkg.Cached.Config.Deployment.DefinitionFile == "" {
			continue
		}
		pkgPath := depl.Pkg.Cached.ConfigPath
		var f fs.FS
		var definitionFilePath string
		if filepath.IsAbs(pkgPath) {
			f = os.DirFS(pkgPath)
			definitionFilePath = depl.Pkg.Cached.Config.Deployment.DefinitionFile
		} else {
			f = cacheFS
			definitionFilePath = filepath.Join(
				pkgPath, depl.Pkg.Cached.Config.Deployment.DefinitionFile,
			)
		}
		stackConfig, err := docker.LoadStackDefinition(f, definitionFilePath)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load Docker stack definition from %s", definitionFilePath,
			)
		}
		for _, service := range stackConfig.Services {
			fcli.BulletedPrintf(indent+1, "%s: %s\n", service.Name, service.Image)
			if _, ok := images[service.Image]; !ok {
				images[service.Image] = struct{}{}
				orderedImages = append(orderedImages, service.Image)
			}
		}
	}
	return orderedImages, nil
}

// check

func envCheckAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		return errMissingCache
	}

	if err := checkEnv(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil,
	); err != nil {
		return err
	}
	return nil
}

func checkEnv(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
) error {
	cacheFS := os.DirFS(cachePath)
	depls, err := forklift.ListDepls(os.DirFS(envPath), cacheFS, replacementRepos)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}

	conflicts, err := forklift.CheckDeplConflicts(depls)
	if err != nil {
		return errors.Wrap(err, "couldn't check for conflicts among deployments")
	}
	if len(conflicts) > 0 {
		fcli.IndentedPrintln(indent, "Found resource conflicts among deployments:")
	}
	for _, conflict := range conflicts {
		if err = printDeplConflict(1, conflict); err != nil {
			return errors.Wrapf(
				err, "couldn't print resource conflicts among deployments %s and %s",
				conflict.First.Name, conflict.Second.Name,
			)
		}
	}

	missingDeps, err := forklift.CheckDeplDependencies(depls)
	if err != nil {
		return errors.Wrap(err, "couldn't check for unmet dependencies among deployments")
	}
	if len(missingDeps) > 0 {
		fcli.IndentedPrintln(indent, "Found unmet resource dependencies among deployments:")
	}
	for _, missingDep := range missingDeps {
		if err := printMissingDeplDependency(1, missingDep); err != nil {
			return err
		}
	}

	if len(conflicts) > 0 || len(missingDeps) > 0 {
		return errors.New("environment failed constraint checks")
	}
	return nil
}

func printDeplConflict(indent int, conflict forklift.DeplConflict) error {
	fcli.IndentedPrintf(indent, "Between %s and %s:\n", conflict.First.Name, conflict.Second.Name)
	indent++

	if conflict.HasNameConflict() {
		fcli.IndentedPrintln(indent, "Conflicting deployment names")
	}
	if conflict.HasListenerConflict() {
		fcli.IndentedPrintln(indent, "Conflicting host port listeners:")
		if err := printResourceConflicts(indent+1, conflict.Listeners); err != nil {
			return errors.Wrap(err, "couldn't print conflicting host port listeners")
		}
	}
	if conflict.HasNetworkConflict() {
		fcli.IndentedPrintln(indent, "Conflicting Docker networks:")
		if err := printResourceConflicts(indent+1, conflict.Networks); err != nil {
			return errors.Wrap(err, "couldn't print conflicting docker networks")
		}
	}
	if conflict.HasServiceConflict() {
		fcli.IndentedPrintln(indent, "Conflicting network services:")
		if err := printResourceConflicts(indent+1, conflict.Services); err != nil {
			return errors.Wrap(err, "couldn't print conflicting network services")
		}
	}
	return nil
}

func printResourceConflicts[Resource any](
	indent int, conflicts []forklift.ResourceConflict[Resource],
) error {
	for _, resourceConflict := range conflicts {
		if err := printResourceConflict(indent, resourceConflict); err != nil {
			return errors.Wrap(err, "couldn't print resource conflict")
		}
	}
	return nil
}

func printResourceConflict[Resource any](
	indent int, conflict forklift.ResourceConflict[Resource],
) error {
	fcli.BulletedPrintf(indent, "Conflicting resource from %s:\n", conflict.First.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResourceSource(indent+1, conflict.First.Source[1:])
	if err := fcli.IndentedPrintYaml(resourceIndent+1, conflict.First.Resource); err != nil {
		return errors.Wrap(err, "couldn't print first resource")
	}
	fcli.IndentedPrintf(indent, "Conflicting resource from %s:\n", conflict.Second.Source[0])
	resourceIndent = printResourceSource(indent+1, conflict.Second.Source[1:])
	if err := fcli.IndentedPrintYaml(resourceIndent+1, conflict.Second.Resource); err != nil {
		return errors.Wrap(err, "couldn't print second resource")
	}

	fcli.IndentedPrint(indent, "Resources are conflicting because of:")
	if len(conflict.Errs) == 0 {
		fmt.Print(" (unknown)")
	}
	fmt.Println()
	for _, err := range conflict.Errs {
		fcli.BulletedPrintf(indent+1, "%s\n", err)
	}
	return nil
}

func printResourceSource(indent int, source []string) (finalIndent int) {
	for i, line := range source {
		finalIndent = indent + i
		fcli.IndentedPrintf(finalIndent, "%s:", line)
		fmt.Println()
	}
	return finalIndent
}

func printMissingDeplDependency(indent int, deps forklift.MissingDeplDependencies) error {
	fcli.IndentedPrintf(indent, "For %s:\n", deps.Depl.Name)
	indent++

	if deps.HasMissingNetworkDependency() {
		fcli.IndentedPrintln(indent, "Missing Docker networks:")
		if err := printMissingDependencies(indent+1, deps.Networks); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet Docker network dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	if deps.HasMissingServiceDependency() {
		fcli.IndentedPrintln(indent, "Missing network services:")
		if err := printMissingDependencies(indent+1, deps.Services); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet network service dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	return nil
}

func printMissingDependencies[Resource any](
	indent int, missingDeps []forklift.MissingResourceDependency[Resource],
) error {
	for _, missingDep := range missingDeps {
		if err := printMissingDependency(indent, missingDep); err != nil {
			return errors.Wrap(err, "couldn't print unmet resource dependency")
		}
	}
	return nil
}

func printMissingDependency[Resource any](
	indent int, missingDep forklift.MissingResourceDependency[Resource],
) error {
	fcli.BulletedPrintf(indent, "Resource required by %s:\n", missingDep.Required.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResourceSource(indent+1, missingDep.Required.Source[1:])
	if err := fcli.IndentedPrintYaml(resourceIndent+1, missingDep.Required.Resource); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}
	fcli.IndentedPrintln(indent, "Best candidates to meet requirement:")
	indent++

	for _, candidate := range missingDep.BestCandidates {
		if err := printDependencyCandidate(indent, candidate); err != nil {
			return errors.Wrap(err, "couldn't print dependency candidate")
		}
	}
	return nil
}

func printDependencyCandidate[Resource any](
	indent int, candidate forklift.ResourceDependencyCandidate[Resource],
) error {
	fcli.BulletedPrintf(indent, "Candidate resource from %s:\n", candidate.Provided.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResourceSource(indent+1, candidate.Provided.Source[1:])
	if err := fcli.IndentedPrintYaml(resourceIndent+1, candidate.Provided.Resource); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}

	fcli.IndentedPrintln(indent, "Candidate doesn't meet requirement because of:")
	indent++
	for _, err := range candidate.Errs {
		fcli.BulletedPrintf(indent, "%s\n", err)
	}
	return nil
}

// plan

func envPlanAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		return errMissingCache
	}

	if err := planEnv(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil,
	); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local environment (have you run `forklift env cache` recently?)",
		)
	}
	return nil
}

func planEnv(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
) error {
	cacheFS := os.DirFS(cachePath)
	depls, err := forklift.ListDepls(os.DirFS(envPath), cacheFS, replacementRepos)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}
	dc, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}
	stacks, err := dc.ListStacks(context.Background())
	if err != nil {
		return errors.Wrapf(err, "couldn't list active Docker stacks")
	}

	fcli.IndentedPrintln(indent, "Determining package deployment changes...")
	changes := planReconciliation(depls, stacks)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Name < changes[j].Name
	})
	for _, change := range changes {
		fcli.IndentedPrintf(indent, "Will %s %s\n", strings.ToLower(change.Type), change.Name)
	}
	return nil
}

const (
	addReconciliationChange    = "Add"
	removeReconciliationChange = "Remove"
	updateReconciliationChange = "Update"
)

type reconciliationChange struct {
	Name  string
	Type  string
	Depl  forklift.Depl
	Stack docker.Stack
}

func planReconciliation(depls []forklift.Depl, stacks []docker.Stack) []reconciliationChange {
	deplSet := make(map[string]forklift.Depl)
	for _, depl := range depls {
		deplSet[depl.Name] = depl
	}
	stackSet := make(map[string]docker.Stack)
	for _, stack := range stacks {
		stackSet[stack.Name] = stack
	}

	changes := make([]reconciliationChange, 0, len(deplSet)+len(stackSet))
	for name, depl := range deplSet {
		definesStack := depl.Pkg.Cached.Config.Deployment.DefinesStack()
		stack, ok := stackSet[name]
		if !ok {
			if definesStack {
				changes = append(changes, reconciliationChange{
					Name: name,
					Type: addReconciliationChange,
					Depl: depl,
				})
			}
			continue
		}
		if definesStack {
			changes = append(changes, reconciliationChange{
				Name:  name,
				Type:  updateReconciliationChange,
				Depl:  depl,
				Stack: stack,
			})
		}
	}
	for name, stack := range stackSet {
		if depl, ok := deplSet[name]; ok && depl.Pkg.Cached.Config.Deployment.DefinesStack() {
			continue
		}
		changes = append(changes, reconciliationChange{
			Name:  name,
			Type:  removeReconciliationChange,
			Stack: stack,
		})
	}

	// TODO: reorder reconciliation actions based on dependencies
	return changes
}

// apply

func envApplyAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		return errMissingCache
	}

	if err := applyEnv(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil,
	); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local environment (have you run `forklift env cache` recently?)",
		)
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}

func applyEnv(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
) error {
	cacheFS := os.DirFS(cachePath)
	depls, err := forklift.ListDepls(os.DirFS(envPath), cacheFS, replacementRepos)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}
	dc, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}
	stacks, err := dc.ListStacks(context.Background())
	if err != nil {
		return errors.Wrapf(err, "couldn't list active Docker stacks")
	}

	fcli.IndentedPrintln(indent, "Determining package deployment changes...")
	changes := planReconciliation(depls, stacks)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Name < changes[j].Name
	})
	for _, change := range changes {
		fcli.IndentedPrintf(indent, "Will %s %s\n", strings.ToLower(change.Type), change.Name)
	}

	for _, change := range changes {
		if err := applyReconciliationChange(0, cacheFS, change, dc); err != nil {
			return errors.Wrapf(err, "couldn't apply '%s' change to stack %s", change.Type, change.Name)
		}
	}
	return nil
}

func applyReconciliationChange(
	indent int, cacheFS fs.FS, change reconciliationChange, dc *docker.Client,
) error {
	fmt.Println()
	switch change.Type {
	default:
		return errors.Errorf("unknown change type '%s'", change.Type)
	case addReconciliationChange:
		fcli.IndentedPrintf(indent, "Adding package deployment %s...\n", change.Name)
		if err := deployStack(indent+1, cacheFS, change.Depl.Pkg.Cached, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	case removeReconciliationChange:
		fcli.IndentedPrintf(indent, "Removing package deployment %s...\n", change.Name)
		if err := dc.RemoveStacks(context.Background(), []string{change.Name}); err != nil {
			return errors.Wrapf(err, "couldn't remove %s", change.Name)
		}
		return nil
	case updateReconciliationChange:
		fcli.IndentedPrintf(indent, "Updating package deployment %s...\n", change.Name)
		if err := deployStack(indent+1, cacheFS, change.Depl.Pkg.Cached, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	}
}

func deployStack(
	indent int, cacheFS fs.FS, cachedPkg forklift.CachedPkg, name string, dc *docker.Client,
) error {
	pkgDeplSpec := cachedPkg.Config.Deployment
	if !pkgDeplSpec.DefinesStack() {
		fcli.IndentedPrintln(indent, "No Docker stack to deploy!")
		return nil
	}
	definitionFilePath := filepath.Join(cachedPkg.ConfigPath, pkgDeplSpec.DefinitionFile)
	stackConfig, err := docker.LoadStackDefinition(cacheFS, definitionFilePath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load Docker stack definition from %s", definitionFilePath,
		)
	}
	if err = dc.DeployStack(
		context.Background(), name, stackConfig, docker.NewOutStream(os.Stdout),
	); err != nil {
		return errors.Wrapf(err, "couldn't deploy stack '%s'", name)
	}
	return nil
}

// ls-repo

func envLsRepoAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}

	return printEnvRepos(0, workspace.LocalEnvPath(wpath))
}

func printEnvRepos(indent int, envPath string) error {
	repos, err := forklift.ListVersionedRepos(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	sort.Slice(repos, func(i, j int) bool {
		return forklift.CompareVersionedRepos(repos[i], repos[j]) < 0
	})
	for _, repo := range repos {
		fcli.IndentedPrintf(indent, "%s\n", repo.Path())
	}
	return nil
}

// show-repo

func envShowRepoAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		return errMissingCache
	}

	repoPath := c.Args().First()
	return printRepoInfo(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil, repoPath)
}

func printRepoInfo(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
	repoPath string,
) error {
	reposFS, err := forklift.VersionedReposFS(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(
			err, "couldn't open directory for Pallet repositories in environment %s", envPath,
		)
	}
	versionedRepo, err := forklift.LoadVersionedRepo(reposFS, repoPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load Pallet repo versioning config %s from environment %s", repoPath, envPath,
		)
	}
	// TODO: maybe the version should be computed and error-handled when the repo is loaded, so that
	// we don't need error-checking for every subsequent access of the version
	version, err := versionedRepo.Config.Version()
	if err != nil {
		return errors.Wrapf(err, "couldn't determine configured version of Pallet repo %s", repoPath)
	}
	printVersionedRepo(indent, versionedRepo)
	fmt.Println()

	var cachedRepo forklift.CachedRepo
	replacementRepo, ok := replacementRepos[repoPath]
	if ok {
		cachedRepo = replacementRepo.Repo
	} else {
		if cachedRepo, err = forklift.FindCachedRepo(
			os.DirFS(cachePath), repoPath, version,
		); err != nil {
			return errors.Wrapf(
				err,
				"couldn't find Pallet repository %s@%s in cache, please update the local cache of repos",
				repoPath, version,
			)
		}
	}
	if filepath.IsAbs(cachedRepo.ConfigPath) {
		fcli.IndentedPrint(indent+1, "External path (replacing cached package): ")
	} else {
		fcli.IndentedPrint(indent+1, "Path in cache: ")
	}
	fmt.Println(cachedRepo.ConfigPath)
	fcli.IndentedPrintf(indent+1, "Description: %s\n", cachedRepo.Config.Repository.Description)
	// TODO: show the README file
	return nil
}

func printVersionedRepo(indent int, repo forklift.VersionedRepo) {
	fcli.IndentedPrintf(indent, "Pallet repository: %s\n", repo.Path())
	indent++
	version, _ := repo.Config.Version() // assume that the validity of the version was already checked
	fcli.IndentedPrintf(indent, "Locked version: %s\n", version)
	fcli.IndentedPrintf(indent, "Provided by Git repository: %s\n", repo.VCSRepoPath)
}

// ls-pkg

func envLsPkgAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		return errMissingCache
	}

	return printEnvPkgs(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil)
}

func printEnvPkgs(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
) error {
	repos, err := forklift.ListVersionedRepos(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories in environment %s", envPath)
	}
	pkgs, err := forklift.ListVersionedPkgs(os.DirFS(cachePath), replacementRepos, repos)
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet packages")
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return forklift.CompareCachedPkgs(pkgs[i], pkgs[j]) < 0
	})
	for _, pkg := range pkgs {
		fcli.IndentedPrintf(indent, "%s\n", pkg.Path)
	}
	return nil
}

// show-pkg

func envShowPkgAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		return errMissingCache
	}

	pkgPath := c.Args().First()
	return printPkgInfo(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil, pkgPath)
}

func printPkgInfo(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
	pkgPath string,
) error {
	reposFS, err := forklift.VersionedReposFS(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(
			err, "couldn't open directory for Pallet repositories in environment %s", envPath,
		)
	}

	var pkg forklift.VersionedPkg
	repo, ok := forklift.FindExternalRepoOfPkg(replacementRepos, pkgPath)
	if ok {
		externalPkg, perr := forklift.FindExternalPkg(repo, pkgPath)
		if perr != nil {
			return errors.Wrapf(
				err, "couldn't find external package %s from replacement repo %s",
				pkgPath, repo.Repo.ConfigPath,
			)
		}
		pkg = forklift.AsVersionedPkg(externalPkg)
	} else if pkg, err = forklift.LoadVersionedPkg(reposFS, os.DirFS(cachePath), pkgPath); err != nil {
		return errors.Wrapf(
			err, "couldn't look up information about package %s in environment %s", pkgPath, envPath,
		)
	}

	printVersionedPkg(indent, pkg)
	return nil
}

func printVersionedPkg(indent int, pkg forklift.VersionedPkg) {
	fcli.IndentedPrintf(indent, "Pallet package: %s\n", pkg.Path)
	indent++

	printVersionedPkgRepo(indent, pkg)
	if filepath.IsAbs(pkg.Cached.ConfigPath) {
		fcli.IndentedPrint(indent, "External path (replacing cached package): ")
	} else {
		fcli.IndentedPrint(indent, "Path in cache: ")
	}
	fmt.Println(pkg.Cached.ConfigPath)
	fmt.Println()

	printPkgSpec(indent, pkg.Cached.Config.Package)
	fmt.Println()
	printDeplSpec(indent, pkg.Cached.Config.Deployment)
	fmt.Println()
	printFeatureSpecs(indent, pkg.Cached.Config.Features)
}

func printVersionedPkgRepo(indent int, pkg forklift.VersionedPkg) {
	fcli.IndentedPrintf(indent, "Provided by Pallet repository: %s\n", pkg.Repo.Path())
	indent++

	if filepath.IsAbs(pkg.Cached.Repo.ConfigPath) {
		fcli.IndentedPrintf(
			indent, "External path (replacing cached repository): %s\n", pkg.Cached.Repo.ConfigPath,
		)
	} else {
		fcli.IndentedPrintf(indent, "Version: %s\n", pkg.Cached.Repo.Version)
	}

	fcli.IndentedPrintf(indent, "Description: %s\n", pkg.Cached.Repo.Config.Repository.Description)
	fcli.IndentedPrintf(indent, "Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
}

// ls-depl

func envLsDeplAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		return errMissingCache
	}
	return printEnvDepls(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil)
}

func printEnvDepls(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
) error {
	depls, err := forklift.ListDepls(os.DirFS(envPath), os.DirFS(cachePath), replacementRepos)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}
	for _, depl := range depls {
		fcli.IndentedPrintf(indent, "%s\n", depl.Name)
	}
	return nil
}

// show-depl

func envShowDeplAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		return errMissingCache
	}

	deplName := c.Args().First()
	return printDeplInfo(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil, deplName)
}

func printDeplInfo(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
	deplName string,
) error {
	cacheFS := os.DirFS(cachePath)
	depl, err := forklift.LoadDepl(os.DirFS(envPath), cacheFS, replacementRepos, deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment specification %s in environment %s", deplName, envPath,
		)
	}
	printDepl(indent, depl)
	indent++

	cachedPkg := depl.Pkg.Cached
	pkgDeplSpec := cachedPkg.Config.Deployment
	if pkgDeplSpec.DefinesStack() {
		fmt.Println()
		fcli.IndentedPrintln(indent, "Deploys with Docker stack:")
		definitionFilePath := filepath.Join(cachedPkg.ConfigPath, pkgDeplSpec.DefinitionFile)
		stackConfig, err := docker.LoadStackDefinition(cacheFS, definitionFilePath)
		if err != nil {
			return errors.Wrapf(err, "couldn't load Docker stack definition from %s", definitionFilePath)
		}
		printDockerStackConfig(indent+1, *stackConfig)
	}

	// TODO: print the state of the Docker stack associated with deplName - or maybe that should be
	// a `forklift depl show-d deplName` command instead?
	return nil
}

func printDepl(indent int, depl forklift.Depl) {
	fcli.IndentedPrintf(indent, "Pallet package deployment: %s\n", depl.Name)
	indent++

	printDeplPkg(indent, depl)

	enabledFeatures, err := depl.EnabledFeatures()
	if err != nil {
		fcli.IndentedPrintf(indent, "Warning: couldn't determine enabled features: %s\n", err.Error())
	}
	fcli.IndentedPrint(indent, "Enabled features:")
	if len(enabledFeatures) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	printFeatures(indent+1, enabledFeatures)

	disabledFeatures := depl.DisabledFeatures()
	fcli.IndentedPrint(indent, "Disabled features:")
	if len(disabledFeatures) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	printFeatures(indent+1, disabledFeatures)
}

func printDeplPkg(indent int, depl forklift.Depl) {
	fcli.IndentedPrintf(indent, "Deploys Pallet package: %s\n", depl.Config.Package)
	indent++

	fcli.IndentedPrintf(indent, "Description: %s\n", depl.Pkg.Cached.Config.Package.Description)
	printVersionedPkgRepo(indent, depl.Pkg)
}

func printFeatures(indent int, features map[string]forklift.PkgFeatureSpec) {
	orderedNames := make([]string, 0, len(features))
	for name := range features {
		orderedNames = append(orderedNames, name)
	}
	sort.Strings(orderedNames)
	for _, name := range orderedNames {
		if description := features[name].Description; description != "" {
			fcli.IndentedPrintf(indent, "%s: %s\n", name, description)
			continue
		}
		fcli.IndentedPrintf(indent, "%s\n", name)
	}
}

func printDockerStackConfig(indent int, stackConfig dct.Config) {
	printDockerStackServices(indent, stackConfig.Services)
	// TODO: also print networks, volumes, etc.
}

func printDockerStackServices(indent int, services []dct.ServiceConfig) {
	if len(services) == 0 {
		return
	}
	fcli.IndentedPrint(indent, "Services:")
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
	if len(services) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, service := range services {
		fcli.IndentedPrintf(indent, "%s: %s\n", service.Name, service.Image)
	}
}
