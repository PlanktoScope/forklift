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
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

var errMissingEnv = errors.Errorf(
	"you first need to set up a local environment with `forklift env clone`",
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
	fmt.Println("Done! Next, you'll probably want to run `forklift env cache`.")
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
	return printEnvInfo(envPath)
}

func printEnvInfo(envPath string) error {
	fmt.Printf("Environment: %s\n", envPath)
	config, err := forklift.LoadEnvConfig(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't load the environment config")
	}
	fmt.Printf("  Description: %s\n", config.Environment.Description)

	ref, err := git.Head(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its HEAD", envPath)
	}
	fmt.Printf("  Currently on: %s\n", git.StringifyRef(ref))
	// TODO: report any divergence between head and remotes
	if err := printUncommittedChanges(envPath); err != nil {
		return err
	}

	fmt.Println()
	if err := printLocalRefsInfo(envPath); err != nil {
		return err
	}
	fmt.Println()
	if err := printRemotesInfo(envPath); err != nil {
		return err
	}
	return nil
}

func printRemotesInfo(envPath string) error {
	remotes, err := git.Remotes(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its remotes", envPath)
	}

	fmt.Printf("  Remotes:")
	if len(remotes) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, remote := range remotes {
		config := remote.Config()
		fmt.Printf("    %s:\n", config.Name)
		fmt.Printf("      URLs:")
		if len(config.URLs) == 0 {
			fmt.Print(" (none)")
		}
		fmt.Println()
		for i, url := range config.URLs {
			fmt.Printf("        %s: ", url)
			if i == 0 {
				fmt.Print("fetch, ")
			}
			fmt.Println("push")
		}

		fmt.Printf("      Up-to-date references:")
		refs, err := remote.List(git.EmptyListOptions())
		if err != nil {
			fmt.Printf(" (couldn't retrieve references: %s)\n", err)
			continue
		}
		if len(refs) == 0 {
			fmt.Print(" (none)")
		}
		fmt.Println()
		for _, ref := range refs {
			fmt.Printf("        %s\n", git.StringifyRef(ref))
		}
	}
	return nil
}

func printLocalRefsInfo(envPath string) error {
	refs, err := git.Refs(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its refs", envPath)
	}

	fmt.Printf("  References:")
	if len(refs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, ref := range refs {
		fmt.Printf("    %s\n", git.StringifyRef(ref))
	}

	return nil
}

func printUncommittedChanges(envPath string) error {
	status, err := git.Status(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query the environment %s for its status", envPath)
	}
	fmt.Print("  Uncommitted changes:")
	if len(status) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for file, status := range status {
		if status.Staging == git.StatusUnmodified && status.Worktree == git.StatusUnmodified {
			continue
		}
		if status.Staging == git.StatusRenamed {
			file = fmt.Sprintf("%s -> %s", file, status.Extra)
		}
		fmt.Printf("    %c%c %s\n", status.Staging, status.Worktree, file)
	}
	return nil
}

// cache

func envCacheAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		return errMissingEnv
	}

	fmt.Printf("Downloading Pallet repositories specified by the local environment...\n")
	changed, err := downloadRepos(workspace.LocalEnvPath(wpath), workspace.CachePath(wpath))
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Done! No further actions are needed at this time.")
		return nil
	}

	// TODO: download all Docker images used by packages in the repo - either by inspecting the
	// Docker stack definitions or by allowing packages to list Docker images used.
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift env deploy`.")
	return nil
}

func downloadRepos(envPath, cachePath string) (changed bool, err error) {
	repos, err := forklift.ListVersionedRepos(os.DirFS(envPath))
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	changed = false
	for _, repo := range repos {
		downloaded, err := downloadRepo(cachePath, repo)
		changed = changed || downloaded
		if err != nil {
			return false, errors.Wrapf(
				err, "couldn't download %s at commit %s", repo.Path(), repo.Config.ShortCommit(),
			)
		}
	}
	return changed, nil
}

func downloadRepo(palletsPath string, repo forklift.VersionedRepo) (downloaded bool, err error) {
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

	fmt.Printf("Downloading %s@%s...\n", repo.VCSRepoPath, version)
	gitRepo, err := git.Clone(vcsRepoPath, path)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't clone repo %s to %s", vcsRepoPath, path)
	}

	// Validate commit
	shortCommit := repo.Config.ShortCommit()
	if err = validateCommit(repo, gitRepo); err != nil {
		if cerr := os.RemoveAll(path); cerr != nil {
			fmt.Printf(
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
			fmt.Printf("Error: couldn't clean up %s! You will need to delete it yourself.\n", path)
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

// deploy

func envDeployAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Println("The cache is empty, please run `forklift env cache` first")
		return nil
	}

	if err := deployEnv(workspace.LocalEnvPath(wpath), workspace.CachePath(wpath)); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local environment (have you run `forklift env cache` recently?)",
		)
	}
	fmt.Println("Done!")
	return nil
}

func deployEnv(envPath, cachePath string) error {
	cacheFS := os.DirFS(cachePath)
	depls, err := forklift.ListDepls(os.DirFS(envPath), cacheFS)
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

	fmt.Println("Determining package deployment changes...")
	changes := planReconciliation(depls, stacks)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Name < changes[j].Name
	})
	for _, change := range changes {
		fmt.Printf("Will %s %s\n", strings.ToLower(change.Type), change.Name)
	}

	fmt.Println()
	fmt.Println("Applying package deployment changes...")
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker swarm client")
	}
	for _, change := range changes {
		if err := applyReconciliationChange(cacheFS, change, dc); err != nil {
			return errors.Wrapf(err, "couldn't apply '%s' change to stack %s", change.Type, change.Name)
		}
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

func applyReconciliationChange(
	cacheFS fs.FS, change reconciliationChange, dc *docker.Client,
) error {
	switch change.Type {
	default:
		return errors.Errorf("unknown change type '%s'", change.Type)
	case addReconciliationChange:
		fmt.Printf("Adding %s...\n", change.Name)
		if err := deployStack(cacheFS, change.Depl.Pkg.Cached, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		fmt.Println("  Done!")
		return nil
	case removeReconciliationChange:
		fmt.Printf("Removing %s...\n", change.Name)
		if err := dc.RemoveStacks(context.Background(), []string{change.Name}); err != nil {
			return errors.Wrapf(err, "couldn't remove %s", change.Name)
		}
		return nil
	case updateReconciliationChange:
		fmt.Printf("Updating %s...\n", change.Name)
		if err := deployStack(cacheFS, change.Depl.Pkg.Cached, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		fmt.Println("  Done!")
		return nil
	}
}

func deployStack(
	cacheFS fs.FS, cachedPkg forklift.CachedPkg, name string, dc *docker.Client,
) error {
	pkgDeplSpec := cachedPkg.Config.Deployment
	if !pkgDeplSpec.DefinesStack() {
		fmt.Println("  No Docker stack to deploy!")
		return nil
	}
	definitionFilePath := filepath.Join(cachedPkg.ConfigPath, pkgDeplSpec.DefinitionFile)
	stackConfig, err := docker.LoadStackDefinition(cacheFS, definitionFilePath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load Docker stack definition from %s", definitionFilePath,
		)
	}
	if err = dc.DeployStack(context.Background(), name, stackConfig); err != nil {
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

	return printEnvRepos(workspace.LocalEnvPath(wpath))
}

func printEnvRepos(envPath string) error {
	repos, err := forklift.ListVersionedRepos(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	sort.Slice(repos, func(i, j int) bool {
		return forklift.CompareVersionedRepos(repos[i], repos[j]) < 0
	})
	for _, repo := range repos {
		fmt.Printf("%s\n", repo.Path())
	}
	return nil
}

// info-repo

func envShowRepoAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}

	repoPath := c.Args().First()
	return printRepoInfo(workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), repoPath)
}

func printRepoInfo(envPath, cachePath, repoPath string) error {
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
	printVersionedRepo(versionedRepo)
	fmt.Println()

	cachedRepo, err := forklift.FindCachedRepo(os.DirFS(cachePath), repoPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find Pallet repository %s@%s in cache, please run `forklift env cache` again",
			repoPath, version,
		)
	}
	fmt.Printf("  Path in cache: %s\n", cachedRepo.ConfigPath)
	fmt.Printf("  Description: %s\n", cachedRepo.Config.Repository.Description)
	return nil
}

func printVersionedRepo(repo forklift.VersionedRepo) {
	fmt.Printf("Pallet repository: %s\n", repo.Path())
	version, _ := repo.Config.Version() // assume that the validity of the version was already checked
	fmt.Printf("  Locked version: %s\n", version)
	fmt.Printf("  Provided by Git repository: %s\n", repo.VCSRepoPath)
}

// ls-pkg

func envLsPkgAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Println("The cache is empty, please run `forklift env cache` first")
		return nil
	}

	return printEnvPkgs(workspace.LocalEnvPath(wpath), workspace.CachePath(wpath))
}

func printEnvPkgs(envPath, cachePath string) error {
	repos, err := forklift.ListVersionedRepos(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories in environment %s", envPath)
	}
	pkgs, err := forklift.ListVersionedPkgs(os.DirFS(cachePath), repos)
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet packages")
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return forklift.CompareCachedPkgs(pkgs[i], pkgs[j]) < 0
	})
	for _, pkg := range pkgs {
		fmt.Printf("%s\n", pkg.Path)
	}
	return nil
}

// info-pkg

func envShowPkgAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Println("The cache is empty, please run `forklift env cache` first")
		return nil
	}

	pkgPath := c.Args().First()
	return printPkgInfo(workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), pkgPath)
}

func printPkgInfo(envPath, cachePath, pkgPath string) error {
	reposFS, err := forklift.VersionedReposFS(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(
			err, "couldn't open directory for Pallet repositories in environment %s", envPath,
		)
	}
	pkg, err := forklift.LoadVersionedPkg(reposFS, os.DirFS(cachePath), pkgPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't look up information about package %s in environment %s", pkgPath, envPath,
		)
	}
	printVersionedPkg(pkg)
	return nil
}

func printVersionedPkg(pkg forklift.VersionedPkg) {
	fmt.Printf("Pallet package: %s\n", pkg.Path)
	fmt.Printf("  Provided by Pallet repository: %s\n", pkg.Repo.Path())
	fmt.Printf("    Version: %s\n", pkg.Cached.Repo.Version)
	fmt.Printf("    Description: %s\n", pkg.Cached.Repo.Config.Repository.Description)
	fmt.Printf("    Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
	fmt.Printf("  Path in cache: %s\n", pkg.Cached.ConfigPath)
	fmt.Println()
	printPkgSpec(pkg.Cached.Config.Package)
	fmt.Println()
	printDeplSpec(pkg.Cached.Config.Deployment)
	fmt.Println()
	printFeatureSpecs(pkg.Cached.Config.Features)
}

// ls-depl

func envLsDeplAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	return printEnvDepls(workspace.LocalEnvPath(wpath), workspace.CachePath(wpath))
}

func printEnvDepls(envPath, cachePath string) error {
	depls, err := forklift.ListDepls(os.DirFS(envPath), os.DirFS(cachePath))
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}
	for _, depl := range depls {
		fmt.Printf("%s\n", depl.Name)
	}
	return nil
}

// info-depl

func envShowDeplAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Println("The cache is empty, please run `forklift env cache` first")
		return nil
	}

	deplName := c.Args().First()
	return printDeplInfo(workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), deplName)
}

func printDeplInfo(envPath, cachePath, deplName string) error {
	cacheFS := os.DirFS(cachePath)
	depl, err := forklift.LoadDepl(os.DirFS(envPath), cacheFS, deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment specification %s in environment %s", deplName, envPath,
		)
	}
	if depl.Pkg.Cached.Config.Deployment.Name != deplName {
		return errors.Errorf(
			"package deployment name %s specified by environment %s doesn't match name %s specified "+
				"by package %s from repo %s",
			deplName, envPath, depl.Pkg.Cached.Config.Deployment.Name,
			depl.Pkg.Path, depl.Pkg.Repo.Path(),
		)
	}
	printDepl(depl)

	cachedPkg := depl.Pkg.Cached
	pkgDeplSpec := cachedPkg.Config.Deployment
	if pkgDeplSpec.DefinesStack() {
		fmt.Println()
		fmt.Println("  Deploys with Docker stack:")
		definitionFilePath := filepath.Join(cachedPkg.ConfigPath, pkgDeplSpec.DefinitionFile)
		stackConfig, err := docker.LoadStackDefinition(cacheFS, definitionFilePath)
		if err != nil {
			return errors.Wrapf(err, "couldn't load Docker stack definition from %s", definitionFilePath)
		}
		printDockerStackConfig(*stackConfig)
	}

	// TODO: print the state of the Docker stack associated with deplName - or maybe that should be
	// a `forklift depl info-d deplName` command instead?
	return nil
}

func printDepl(depl forklift.Depl) {
	fmt.Printf("Pallet package deployment: %s\n", depl.Name)
	fmt.Printf("  Deploys Pallet package: %s\n", depl.Config.Package)
	fmt.Printf("    Description: %s\n", depl.Pkg.Cached.Config.Package.Description)
	fmt.Printf("    Provided by Pallet repository: %s\n", depl.Pkg.Repo.Path())
	fmt.Printf("      Version: %s\n", depl.Pkg.Cached.Repo.Version)
	fmt.Printf("      Description: %s\n", depl.Pkg.Cached.Repo.Config.Repository.Description)
	fmt.Printf("      Provided by Git repository: %s\n", depl.Pkg.Repo.VCSRepoPath)

	enabledFeatures, err := depl.EnabledFeatures(depl.Pkg.Cached.Config.Features)
	if err != nil {
		fmt.Printf("Warning: couldn't determine enabled features: %s\n", err.Error())
	}
	fmt.Print("  Enabled features:")
	if len(enabledFeatures) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	printFeatures(enabledFeatures)

	disabledFeatures, err := depl.DisabledFeatures(depl.Pkg.Cached.Config.Features)
	if err != nil {
		fmt.Printf("Warning: couldn't determine disabled features: %s\n", err.Error())
	}
	fmt.Print("  Disabled features:")
	if len(disabledFeatures) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	printFeatures(disabledFeatures)
}

func printFeatures(features map[string]forklift.PkgFeatureSpec) {
	orderedNames := make([]string, 0, len(features))
	for name := range features {
		orderedNames = append(orderedNames, name)
	}
	sort.Strings(orderedNames)
	for _, name := range orderedNames {
		if description := features[name].Description; description != "" {
			fmt.Printf("    %s: %s\n", name, description)
			continue
		}
		fmt.Printf("    %s\n", name)
	}
}

func printDockerStackConfig(stackConfig dct.Config) {
	printDockerStackServices(stackConfig.Services)
	// TODO: also print networks, volumes, etc.
}

func printDockerStackServices(services []dct.ServiceConfig) {
	if len(services) == 0 {
		return
	}
	fmt.Print("    Services:")
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
	if len(services) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, service := range services {
		fmt.Printf("      %s: %s\n", service.Name, service.Image)
	}
}
