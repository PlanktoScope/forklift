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
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

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
	indentedPrintf(indent, "Environment: %s\n", envPath)
	config, err := forklift.LoadEnvConfig(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't load the environment config")
	}

	indentedPrintf(indent, "Description: %s\n", config.Environment.Description)

	ref, err := git.Head(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its HEAD", envPath)
	}
	indentedPrintf(indent, "Currently on: %s\n", git.StringifyRef(ref))
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

	indentedPrintf(indent, "Remotes:")
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
	indentedPrintf(indent, "%s:\n", config.Name)
	indent++

	indentedPrintf(indent, "URLs:")
	if len(config.URLs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for i, url := range config.URLs {
		bulletedPrintf(indent+1, "%s: ", url)
		if i == 0 {
			fmt.Print("fetch, ")
		}
		fmt.Println("push")
	}

	indentedPrintf(indent, "Up-to-date references:")
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
		bulletedPrintf(indent+1, "%s\n", git.StringifyRef(ref))
	}
}

func printLocalRefsInfo(indent int, envPath string) error {
	refs, err := git.Refs(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its refs", envPath)
	}

	indentedPrintf(indent, "References:")
	if len(refs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, ref := range refs {
		bulletedPrintf(indent, "%s\n", git.StringifyRef(ref))
	}

	return nil
}

func printUncommittedChanges(indent int, envPath string) error {
	status, err := git.Status(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query the environment %s for its status", envPath)
	}
	indentedPrint(indent, "Uncommitted changes:")
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
		bulletedPrintf(indent, "%c%c %s\n", status.Staging, status.Worktree, file)
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

	indentedPrintf(indent, "Downloading %s@%s...\n", repo.VCSRepoPath, version)
	gitRepo, err := git.Clone(vcsRepoPath, path)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't clone repo %s to %s", vcsRepoPath, path)
	}

	// Validate commit
	shortCommit := repo.Config.ShortCommit()
	if err = validateCommit(repo, gitRepo); err != nil {
		if cerr := os.RemoveAll(path); cerr != nil {
			indentedPrintf(
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
			indentedPrintf(
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
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath),
	); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift env apply`.")
	return nil
}

func downloadImages(indent int, envPath, cachePath string) error {
	orderedImages, err := listRequiredImages(indent, envPath, cachePath)
	if err != nil {
		return errors.Wrap(err, "couldn't determine images required by package deployments")
	}

	dc, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}
	for _, image := range orderedImages {
		fmt.Println()
		indentedPrintf(indent, "Downloading %s...\n", image)
		pulled, err := dc.PullImage(context.Background(), image, docker.NewOutStream(os.Stdout))
		if err != nil {
			return errors.Wrapf(err, "couldn't download %s", image)
		}
		indentedPrintf(indent, "Downloaded %s from %s\n", pulled.Reference(), pulled.RepoInfo().Name)
	}
	return nil
}

func listRequiredImages(indent int, envPath, cachePath string) ([]string, error) {
	cacheFS := os.DirFS(cachePath)
	depls, err := forklift.ListDepls(os.DirFS(envPath), cacheFS)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}
	orderedImages := make([]string, 0, len(depls))
	images := make(map[string]struct{})
	for _, depl := range depls {
		indentedPrintf(
			indent, "Checking Docker container images used by package deployment %s...\n", depl.Name,
		)
		if depl.Pkg.Cached.Config.Deployment.DefinitionFile == "" {
			continue
		}
		definitionFilePath := filepath.Join(
			depl.Pkg.Cached.ConfigPath, depl.Pkg.Cached.Config.Deployment.DefinitionFile,
		)
		stackConfig, err := docker.LoadStackDefinition(cacheFS, definitionFilePath)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load Docker stack definition from %s", definitionFilePath,
			)
		}
		for _, service := range stackConfig.Services {
			bulletedPrintf(indent+1, "%s: %s\n", service.Name, service.Image)
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

	if err := checkEnv(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath)); err != nil {
		return err
	}
	return nil
}

func checkEnv(indent int, envPath, cachePath string) error {
	cacheFS := os.DirFS(cachePath)
	depls, err := forklift.ListDepls(os.DirFS(envPath), cacheFS)
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
		indentedPrintln(indent, "Found resource conflicts among deployments:")
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
		indentedPrintln(indent, "Found unmet resource dependencies among deployments:")
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
	indentedPrintf(indent, "Between %s and %s:\n", conflict.First.Name, conflict.Second.Name)
	indent++

	if conflict.HasNameConflict() {
		indentedPrintln(indent, "Conflicting deployment names")
	}
	if conflict.HasListenerConflict() {
		indentedPrintln(indent, "Conflicting host port listeners:")
		if err := printResourceConflicts(indent+1, conflict.Listeners); err != nil {
			return errors.Wrap(err, "couldn't print conflicting host port listeners")
		}
	}
	if conflict.HasNetworkConflict() {
		indentedPrintln(indent, "Conflicting Docker networks:")
		if err := printResourceConflicts(indent+1, conflict.Networks); err != nil {
			return errors.Wrap(err, "couldn't print conflicting docker networks")
		}
	}
	if conflict.HasServiceConflict() {
		indentedPrintln(indent, "Conflicting network services:")
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
	bulletedPrintf(indent, "Conflicting resource from %s:\n", conflict.First.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResourceSource(indent+1, conflict.First.Source[1:])
	if err := indentedPrintYaml(resourceIndent+1, conflict.First.Resource); err != nil {
		return errors.Wrap(err, "couldn't print first resource")
	}
	indentedPrintf(indent, "Conflicting resource from %s:\n", conflict.Second.Source[0])
	resourceIndent = printResourceSource(indent+1, conflict.Second.Source[1:])
	if err := indentedPrintYaml(resourceIndent+1, conflict.Second.Resource); err != nil {
		return errors.Wrap(err, "couldn't print second resource")
	}

	indentedPrint(indent, "Resources are conflicting because of:")
	if len(conflict.Errs) == 0 {
		fmt.Print(" (unknown)")
	}
	fmt.Println()
	for _, err := range conflict.Errs {
		bulletedPrintf(indent+1, "%s\n", err)
	}
	return nil
}

func printResourceSource(indent int, source []string) (finalIndent int) {
	for i, line := range source {
		finalIndent = indent + i
		indentedPrintf(finalIndent, "%s:", line)
		fmt.Println()
	}
	return finalIndent
}

func printMissingDeplDependency(indent int, deps forklift.MissingDeplDependencies) error {
	indentedPrintf(indent, "For %s:\n", deps.Depl.Name)
	indent++

	if deps.HasMissingNetworkDependency() {
		indentedPrintln(indent, "Missing Docker networks:")
		if err := printMissingDependencies(indent+1, deps.Networks); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet Docker network dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	if deps.HasMissingServiceDependency() {
		indentedPrintln(indent, "Missing network services:")
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
	bulletedPrintf(indent, "Resource required by %s:\n", missingDep.Required.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResourceSource(indent+1, missingDep.Required.Source[1:])
	if err := indentedPrintYaml(resourceIndent+1, missingDep.Required.Resource); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}
	indentedPrintln(indent, "Best candidates to meet requirement:")
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
	bulletedPrintf(indent, "Candidate resource from %s:\n", candidate.Provided.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResourceSource(indent+1, candidate.Provided.Source[1:])
	if err := indentedPrintYaml(resourceIndent+1, candidate.Provided.Resource); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}

	indentedPrintln(indent, "Candidate doesn't meet requirement because of:")
	indent++
	for _, err := range candidate.Errs {
		bulletedPrintf(indent, "%s\n", err)
	}
	return nil
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

	if err := applyEnv(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath)); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local environment (have you run `forklift env cache` recently?)",
		)
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}

func applyEnv(indent int, envPath, cachePath string) error {
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

	indentedPrintln(indent, "Determining package deployment changes...")
	changes := planReconciliation(depls, stacks)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Name < changes[j].Name
	})
	for _, change := range changes {
		indentedPrintf(indent, "Will %s %s\n", strings.ToLower(change.Type), change.Name)
	}

	if err != nil {
		return errors.Wrap(err, "couldn't make Docker swarm client")
	}
	for _, change := range changes {
		if err := applyReconciliationChange(0, cacheFS, change, dc); err != nil {
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
	indent int, cacheFS fs.FS, change reconciliationChange, dc *docker.Client,
) error {
	fmt.Println()
	switch change.Type {
	default:
		return errors.Errorf("unknown change type '%s'", change.Type)
	case addReconciliationChange:
		indentedPrintf(indent, "Adding package deployment %s...\n", change.Name)
		if err := deployStack(indent+1, cacheFS, change.Depl.Pkg.Cached, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	case removeReconciliationChange:
		indentedPrintf(indent, "Removing package deployment %s...\n", change.Name)
		if err := dc.RemoveStacks(context.Background(), []string{change.Name}); err != nil {
			return errors.Wrapf(err, "couldn't remove %s", change.Name)
		}
		return nil
	case updateReconciliationChange:
		indentedPrintf(indent, "Updating package deployment %s...\n", change.Name)
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
		indentedPrintln(indent, "No Docker stack to deploy!")
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
		indentedPrintf(indent, "%s\n", repo.Path())
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
	return printRepoInfo(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), repoPath)
}

func printRepoInfo(indent int, envPath, cachePath, repoPath string) error {
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

	cachedRepo, err := forklift.FindCachedRepo(os.DirFS(cachePath), repoPath, version)
	if err != nil {
		return errors.Wrapf(
			err,
			"couldn't find Pallet repository %s@%s in cache, please run `forklift env cache-repo` again",
			repoPath, version,
		)
	}
	indentedPrintf(indent+1, "Path in cache: %s\n", cachedRepo.ConfigPath)
	indentedPrintf(indent+1, "Description: %s\n", cachedRepo.Config.Repository.Description)
	return nil
}

func printVersionedRepo(indent int, repo forklift.VersionedRepo) {
	indentedPrintf(indent, "Pallet repository: %s\n", repo.Path())
	indent++
	version, _ := repo.Config.Version() // assume that the validity of the version was already checked
	indentedPrintf(indent, "Locked version: %s\n", version)
	indentedPrintf(indent, "Provided by Git repository: %s\n", repo.VCSRepoPath)
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

	return printEnvPkgs(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath))
}

func printEnvPkgs(indent int, envPath, cachePath string) error {
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
		indentedPrintf(indent, "%s\n", pkg.Path)
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
	return printPkgInfo(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), pkgPath)
}

func printPkgInfo(indent int, envPath, cachePath, pkgPath string) error {
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
	printVersionedPkg(indent, pkg)
	return nil
}

func printVersionedPkg(indent int, pkg forklift.VersionedPkg) {
	indentedPrintf(indent, "Pallet package: %s\n", pkg.Path)
	indent++

	printVersionedPkgRepo(indent, pkg)
	indentedPrintf(indent, "Path in cache: %s\n", pkg.Cached.ConfigPath)
	fmt.Println()

	printPkgSpec(indent, pkg.Cached.Config.Package)
	fmt.Println()
	printDeplSpec(indent, pkg.Cached.Config.Deployment)
	fmt.Println()
	printFeatureSpecs(indent, pkg.Cached.Config.Features)
}

func printVersionedPkgRepo(indent int, pkg forklift.VersionedPkg) {
	indentedPrintf(indent, "Provided by Pallet repository: %s\n", pkg.Repo.Path())
	indent++

	indentedPrintf(indent, "Version: %s\n", pkg.Cached.Repo.Version)
	indentedPrintf(indent, "Description: %s\n", pkg.Cached.Repo.Config.Repository.Description)
	indentedPrintf(indent, "Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
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
	return printEnvDepls(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath))
}

func printEnvDepls(indent int, envPath, cachePath string) error {
	depls, err := forklift.ListDepls(os.DirFS(envPath), os.DirFS(cachePath))
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}
	for _, depl := range depls {
		indentedPrintf(indent, "%s\n", depl.Name)
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
	return printDeplInfo(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), deplName)
}

func printDeplInfo(indent int, envPath, cachePath, deplName string) error {
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
	printDepl(indent, depl)
	indent++

	cachedPkg := depl.Pkg.Cached
	pkgDeplSpec := cachedPkg.Config.Deployment
	if pkgDeplSpec.DefinesStack() {
		fmt.Println()
		indentedPrintln(indent, "Deploys with Docker stack:")
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
	indentedPrintf(indent, "Pallet package deployment: %s\n", depl.Name)
	indent++

	printDeplPkg(indent, depl)

	enabledFeatures, err := depl.EnabledFeatures()
	if err != nil {
		indentedPrintf(indent, "Warning: couldn't determine enabled features: %s\n", err.Error())
	}
	indentedPrint(indent, "Enabled features:")
	if len(enabledFeatures) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	printFeatures(indent+1, enabledFeatures)

	disabledFeatures := depl.DisabledFeatures()
	indentedPrint(indent, "Disabled features:")
	if len(disabledFeatures) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	printFeatures(indent+1, disabledFeatures)
}

func printDeplPkg(indent int, depl forklift.Depl) {
	indentedPrintf(indent, "Deploys Pallet package: %s\n", depl.Config.Package)
	indent++

	indentedPrintf(indent, "Description: %s\n", depl.Pkg.Cached.Config.Package.Description)
	printDeplPkgRepo(indent, depl.Pkg)
}

func printDeplPkgRepo(indent int, pkg forklift.VersionedPkg) {
	indentedPrintf(indent, "Provided by Pallet repository: %s\n", pkg.Repo.Path())
	indent++

	indentedPrintf(indent, "Version: %s\n", pkg.Cached.Repo.Version)
	indentedPrintf(indent, "Description: %s\n", pkg.Cached.Repo.Config.Repository.Description)
	indentedPrintf(indent, "Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
}

func printFeatures(indent int, features map[string]forklift.PkgFeatureSpec) {
	orderedNames := make([]string, 0, len(features))
	for name := range features {
		orderedNames = append(orderedNames, name)
	}
	sort.Strings(orderedNames)
	for _, name := range orderedNames {
		if description := features[name].Description; description != "" {
			indentedPrintf(indent, "%s: %s\n", name, description)
			continue
		}
		indentedPrintf(indent, "%s\n", name)
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
	indentedPrint(indent, "Services:")
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
	if len(services) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, service := range services {
		indentedPrintf(indent, "%s: %s\n", service.Name, service.Image)
	}
}
