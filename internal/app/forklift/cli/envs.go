package cli

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	ggit "github.com/go-git/go-git/v5"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/internal/clients/git"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// print

func PrintEnvInfo(indent int, envPath string) error {
	IndentedPrintf(indent, "Environment: %s\n", envPath)
	config, err := forklift.LoadEnvConfig(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't load the environment config")
	}

	IndentedPrintf(indent, "Description: %s\n", config.Environment.Description)

	ref, err := git.Head(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its HEAD", envPath)
	}
	IndentedPrintf(indent, "Currently on: %s\n", git.StringifyRef(ref))
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

func printUncommittedChanges(indent int, envPath string) error {
	status, err := git.Status(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query the environment %s for its status", envPath)
	}
	IndentedPrint(indent, "Uncommitted changes:")
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
		BulletedPrintf(indent, "%c%c %s\n", status.Staging, status.Worktree, file)
	}
	return nil
}

func printLocalRefsInfo(indent int, envPath string) error {
	refs, err := git.Refs(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its refs", envPath)
	}

	IndentedPrintf(indent, "References:")
	if len(refs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, ref := range refs {
		BulletedPrintf(indent, "%s\n", git.StringifyRef(ref))
	}

	return nil
}

func printRemotesInfo(indent int, envPath string) error {
	remotes, err := git.Remotes(envPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its remotes", envPath)
	}

	IndentedPrintf(indent, "Remotes:")
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
	IndentedPrintf(indent, "%s:\n", config.Name)
	indent++

	IndentedPrintf(indent, "URLs:")
	if len(config.URLs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for i, url := range config.URLs {
		BulletedPrintf(indent+1, "%s: ", url)
		if i == 0 {
			fmt.Print("fetch, ")
		}
		fmt.Println("push")
	}

	IndentedPrintf(indent, "Up-to-date references:")
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
		BulletedPrintf(indent+1, "%s\n", git.StringifyRef(ref))
	}
}

// check

func CheckEnv(
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
		IndentedPrintln(indent, "Found resource conflicts among deployments:")
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
		IndentedPrintln(indent, "Found unmet resource dependencies among deployments:")
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
	IndentedPrintf(indent, "Between %s and %s:\n", conflict.First.Name, conflict.Second.Name)
	indent++

	if conflict.HasNameConflict() {
		IndentedPrintln(indent, "Conflicting deployment names")
	}
	if conflict.HasListenerConflict() {
		IndentedPrintln(indent, "Conflicting host port listeners:")
		if err := printResourceConflicts(indent+1, conflict.Listeners); err != nil {
			return errors.Wrap(err, "couldn't print conflicting host port listeners")
		}
	}
	if conflict.HasNetworkConflict() {
		IndentedPrintln(indent, "Conflicting Docker networks:")
		if err := printResourceConflicts(indent+1, conflict.Networks); err != nil {
			return errors.Wrap(err, "couldn't print conflicting docker networks")
		}
	}
	if conflict.HasServiceConflict() {
		IndentedPrintln(indent, "Conflicting network services:")
		if err := printResourceConflicts(indent+1, conflict.Services); err != nil {
			return errors.Wrap(err, "couldn't print conflicting network services")
		}
	}
	return nil
}

func printResourceConflicts[Resource any](
	indent int, conflicts []pallets.ResourceConflict[Resource],
) error {
	for _, resourceConflict := range conflicts {
		if err := printResourceConflict(indent, resourceConflict); err != nil {
			return errors.Wrap(err, "couldn't print resource conflict")
		}
	}
	return nil
}

func printResourceConflict[Resource any](
	indent int, conflict pallets.ResourceConflict[Resource],
) error {
	BulletedPrintf(indent, "Conflicting resource from %s:\n", conflict.First.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResourceSource(indent+1, conflict.First.Source[1:])
	if err := IndentedPrintYaml(resourceIndent+1, conflict.First.Resource); err != nil {
		return errors.Wrap(err, "couldn't print first resource")
	}
	IndentedPrintf(indent, "Conflicting resource from %s:\n", conflict.Second.Source[0])
	resourceIndent = printResourceSource(indent+1, conflict.Second.Source[1:])
	if err := IndentedPrintYaml(resourceIndent+1, conflict.Second.Resource); err != nil {
		return errors.Wrap(err, "couldn't print second resource")
	}

	IndentedPrint(indent, "Resources are conflicting because of:")
	if len(conflict.Errs) == 0 {
		fmt.Print(" (unknown)")
	}
	fmt.Println()
	for _, err := range conflict.Errs {
		BulletedPrintf(indent+1, "%s\n", err)
	}
	return nil
}

func printResourceSource(indent int, source []string) (finalIndent int) {
	for i, line := range source {
		finalIndent = indent + i
		IndentedPrintf(finalIndent, "%s:", line)
		fmt.Println()
	}
	return finalIndent
}

func printMissingDeplDependency(indent int, deps forklift.MissingDeplDependencies) error {
	IndentedPrintf(indent, "For %s:\n", deps.Depl.Name)
	indent++

	if deps.HasMissingNetworkDependency() {
		IndentedPrintln(indent, "Missing Docker networks:")
		if err := printMissingDependencies(indent+1, deps.Networks); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet Docker network dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	if deps.HasMissingServiceDependency() {
		IndentedPrintln(indent, "Missing network services:")
		if err := printMissingDependencies(indent+1, deps.Services); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet network service dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	return nil
}

func printMissingDependencies[Resource any](
	indent int, missingDeps []pallets.MissingResourceDependency[Resource],
) error {
	for _, missingDep := range missingDeps {
		if err := printMissingDependency(indent, missingDep); err != nil {
			return errors.Wrap(err, "couldn't print unmet resource dependency")
		}
	}
	return nil
}

func printMissingDependency[Resource any](
	indent int, missingDep pallets.MissingResourceDependency[Resource],
) error {
	BulletedPrintf(indent, "Resource required by %s:\n", missingDep.Required.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResourceSource(indent+1, missingDep.Required.Source[1:])
	if err := IndentedPrintYaml(resourceIndent+1, missingDep.Required.Resource); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}
	IndentedPrintln(indent, "Best candidates to meet requirement:")
	indent++

	for _, candidate := range missingDep.BestCandidates {
		if err := printDependencyCandidate(indent, candidate); err != nil {
			return errors.Wrap(err, "couldn't print dependency candidate")
		}
	}
	return nil
}

func printDependencyCandidate[Resource any](
	indent int, candidate pallets.ResourceDependencyCandidate[Resource],
) error {
	BulletedPrintf(indent, "Candidate resource from %s:\n", candidate.Provided.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResourceSource(indent+1, candidate.Provided.Source[1:])
	if err := IndentedPrintYaml(resourceIndent+1, candidate.Provided.Resource); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}

	IndentedPrintln(indent, "Candidate doesn't meet requirement because of:")
	indent++
	for _, err := range candidate.Errs {
		BulletedPrintf(indent, "%s\n", err)
	}
	return nil
}

// plan

func PlanEnv(
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

	IndentedPrintln(indent, "Determining package deployment changes...")
	changes := planReconciliation(depls, stacks)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Name < changes[j].Name
	})
	for _, change := range changes {
		IndentedPrintf(indent, "Will %s %s\n", strings.ToLower(change.Type), change.Name)
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

func ApplyEnv(
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

	IndentedPrintln(indent, "Determining package deployment changes...")
	changes := planReconciliation(depls, stacks)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Name < changes[j].Name
	})
	for _, change := range changes {
		IndentedPrintf(indent, "Will %s %s\n", strings.ToLower(change.Type), change.Name)
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
		IndentedPrintf(indent, "Adding package deployment %s...\n", change.Name)
		if err := deployStack(indent+1, cacheFS, change.Depl.Pkg.Cached, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	case removeReconciliationChange:
		IndentedPrintf(indent, "Removing package deployment %s...\n", change.Name)
		if err := dc.RemoveStacks(context.Background(), []string{change.Name}); err != nil {
			return errors.Wrapf(err, "couldn't remove %s", change.Name)
		}
		return nil
	case updateReconciliationChange:
		IndentedPrintf(indent, "Updating package deployment %s...\n", change.Name)
		if err := deployStack(indent+1, cacheFS, change.Depl.Pkg.Cached, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	}
}

func deployStack(
	indent int, cacheFS fs.FS, cachedPkg forklift.CachedPkg, name string, dc *docker.Client,
) error {
	if !cachedPkg.Config.Deployment.DefinesStack() {
		IndentedPrintln(indent, "No Docker stack to deploy!")
		return nil
	}

	stackConfig, err := loadStackDefinition(cacheFS, cachedPkg)
	if err != nil {
		return err
	}
	if err = dc.DeployStack(
		context.Background(), name, stackConfig, docker.NewOutStream(os.Stdout),
	); err != nil {
		return errors.Wrapf(err, "couldn't deploy stack '%s'", name)
	}
	return nil
}
