package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/compose/v2/pkg/api"
	ggit "github.com/go-git/go-git/v5"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/internal/clients/git"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintEnvInfo(indent int, env *forklift.FSEnv) error {
	IndentedPrintf(indent, "Environment: %s\n", env.FS.Path())
	IndentedPrintf(indent, "Description: %s\n", env.Def.Environment.Description)

	ref, err := git.Head(env.FS.Path())
	if err != nil {
		return errors.Wrapf(err, "couldn't query environment %s for its HEAD", env.FS.Path())
	}
	IndentedPrintf(indent, "Currently on: %s\n", git.StringifyRef(ref))
	// TODO: report any divergence between head and remotes
	if err := printUncommittedChanges(indent+1, env.FS.Path()); err != nil {
		return err
	}

	fmt.Println()
	if err := printLocalRefsInfo(indent, env.FS.Path()); err != nil {
		return err
	}
	fmt.Println()
	if err := printRemotesInfo(indent, env.FS.Path()); err != nil {
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

// Check

func CheckEnv(indent int, env *forklift.FSEnv, loader forklift.FSPkgLoader) error {
	depls, err := env.LoadDepls("**/*")
	if err != nil {
		return err
	}
	resolved, err := forklift.ResolveDepls(env, loader, depls)
	if err != nil {
		return err
	}

	conflicts, err := checkDeplConflicts(indent, resolved)
	if err != nil {
		return err
	}
	_, missingDeps, err := checkDeplDeps(indent, resolved)
	if err != nil {
		return err
	}
	if len(conflicts) > 0 || len(missingDeps) > 0 {
		return errors.New("environment failed resource constraint checks")
	}
	return nil
}

func checkDeplConflicts(
	indent int, depls []*forklift.ResolvedDepl,
) ([]forklift.DeplConflict, error) {
	conflicts, err := forklift.CheckDeplConflicts(depls)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't check for conflicts among deployments")
	}
	if len(conflicts) > 0 {
		IndentedPrintln(indent, "Found resource conflicts among deployments:")
	}
	for _, conflict := range conflicts {
		if err = printDeplConflict(1, conflict); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't print resource conflicts among deployments %s and %s",
				conflict.First.Name, conflict.Second.Name,
			)
		}
	}
	return conflicts, nil
}

func printDeplConflict(indent int, conflict forklift.DeplConflict) error {
	IndentedPrintf(indent, "Between %s and %s:\n", conflict.First.Name, conflict.Second.Name)
	indent++

	if conflict.HasNameConflict() {
		IndentedPrintln(indent, "Conflicting deployment names")
	}
	if conflict.HasListenerConflict() {
		IndentedPrintln(indent, "Conflicting host port listeners:")
		if err := printResConflicts(indent+1, conflict.Listeners); err != nil {
			return errors.Wrap(err, "couldn't print conflicting host port listeners")
		}
	}
	if conflict.HasNetworkConflict() {
		IndentedPrintln(indent, "Conflicting Docker networks:")
		if err := printResConflicts(indent+1, conflict.Networks); err != nil {
			return errors.Wrap(err, "couldn't print conflicting docker networks")
		}
	}
	if conflict.HasServiceConflict() {
		IndentedPrintln(indent, "Conflicting network services:")
		if err := printResConflicts(indent+1, conflict.Services); err != nil {
			return errors.Wrap(err, "couldn't print conflicting network services")
		}
	}
	return nil
}

func printResConflicts[Res any](
	indent int, conflicts []pallets.ResConflict[Res],
) error {
	for _, resourceConflict := range conflicts {
		if err := printResConflict(indent, resourceConflict); err != nil {
			return errors.Wrap(err, "couldn't print resource conflict")
		}
	}
	return nil
}

func printResConflict[Res any](
	indent int, conflict pallets.ResConflict[Res],
) error {
	BulletedPrintf(indent, "Conflicting resource from %s:\n", conflict.First.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResSource(indent+1, conflict.First.Source[1:])
	if err := IndentedPrintYaml(resourceIndent+1, conflict.First.Res); err != nil {
		return errors.Wrap(err, "couldn't print first resource")
	}
	IndentedPrintf(indent, "Conflicting resource from %s:\n", conflict.Second.Source[0])
	resourceIndent = printResSource(indent+1, conflict.Second.Source[1:])
	if err := IndentedPrintYaml(resourceIndent+1, conflict.Second.Res); err != nil {
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

func printResSource(indent int, source []string) (finalIndent int) {
	for i, line := range source {
		finalIndent = indent + i
		IndentedPrintf(finalIndent, "%s:", line)
		fmt.Println()
	}
	return finalIndent
}

func checkDeplDeps(
	indent int, depls []*forklift.ResolvedDepl,
) (satisfied []forklift.SatisfiedDeplDeps, missing []forklift.MissingDeplDeps, err error) {
	if satisfied, missing, err = forklift.CheckDeplDeps(depls); err != nil {
		return nil, nil, errors.Wrap(err, "couldn't check dependencies among deployments")
	}
	if len(missing) > 0 {
		IndentedPrintln(indent, "Found unmet resource dependencies among deployments:")
	}
	for _, missingDep := range missing {
		if err := printMissingDeplDep(1, missingDep); err != nil {
			return nil, nil, err
		}
	}
	return satisfied, missing, nil
}

func printMissingDeplDep(indent int, deps forklift.MissingDeplDeps) error {
	IndentedPrintf(indent, "For %s:\n", deps.Depl.Name)
	indent++

	if deps.HasMissingNetworkDep() {
		IndentedPrintln(indent, "Missing Docker networks:")
		if err := printMissingDeps(indent+1, deps.Networks); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet Docker network dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	if deps.HasMissingServiceDep() {
		IndentedPrintln(indent, "Missing network services:")
		if err := printMissingDeps(indent+1, deps.Services); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet network service dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	return nil
}

func printMissingDeps[Res any](indent int, missingDeps []pallets.MissingResDep[Res]) error {
	for _, missingDep := range missingDeps {
		if err := printMissingDep(indent, missingDep); err != nil {
			return errors.Wrap(err, "couldn't print unmet resource dependency")
		}
	}
	return nil
}

func printMissingDep[Res any](indent int, missingDep pallets.MissingResDep[Res]) error {
	BulletedPrintf(indent, "Resource required by %s:\n", missingDep.Required.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResSource(indent+1, missingDep.Required.Source[1:])
	if err := IndentedPrintYaml(resourceIndent+1, missingDep.Required.Res); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}
	IndentedPrintln(indent, "Best candidates to meet requirement:")
	indent++

	for _, candidate := range missingDep.BestCandidates {
		if err := printDepCandidate(indent, candidate); err != nil {
			return errors.Wrap(err, "couldn't print dependency candidate")
		}
	}
	return nil
}

func printDepCandidate[Res any](indent int, candidate pallets.ResDepCandidate[Res]) error {
	BulletedPrintf(indent, "Candidate resource from %s:\n", candidate.Provided.Source[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResSource(indent+1, candidate.Provided.Source[1:])
	if err := IndentedPrintYaml(resourceIndent+1, candidate.Provided.Res); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}

	IndentedPrintln(indent, "Candidate doesn't meet requirement because of:")
	indent++
	for _, err := range candidate.Errs {
		BulletedPrintf(indent, "%s\n", err)
	}
	return nil
}

// Plan

func PlanEnv(indent int, env *forklift.FSEnv, loader forklift.FSPkgLoader) error {
	_, _, err := computePlan(indent, env, loader)
	if err != nil {
		return errors.Wrap(err, "couldn't compute plan for changes")
	}
	return nil
}

const (
	addReconciliationChange    = "Add"
	removeReconciliationChange = "Remove"
	updateReconciliationChange = "Update"
)

type reconciliationChange struct {
	Name string
	Type string
	Depl *forklift.ResolvedDepl
	App  api.Stack
}

func computePlan(
	indent int, env *forklift.FSEnv, loader forklift.FSPkgLoader,
) ([]reconciliationChange, *docker.Client, error) {
	depls, err := env.LoadDepls("**/*")
	if err != nil {
		return nil, nil, err
	}
	resolved, err := forklift.ResolveDepls(env, loader, depls)
	if err != nil {
		return nil, nil, err
	}

	dc, err := docker.NewClient()
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't make Docker API client")
	}
	apps, err := dc.ListApps(context.Background())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't list active Docker Compose apps")
	}

	conflicts, err := checkDeplConflicts(indent, resolved)
	if err != nil {
		return nil, nil, err
	}
	satisfiedDeps, missingDeps, err := checkDeplDeps(indent, resolved)
	if err != nil {
		return nil, nil, err
	}
	if len(conflicts) > 0 || len(missingDeps) > 0 {
		return nil, nil, errors.New("environment failed resource constraint checks")
	}

	IndentedPrintln(indent, "Resolving resource dependencies among package deployments...")
	deps := resolveDeps(satisfiedDeps)
	IndentedPrintln(indent, "Direct dependencies:")
	printDigraph(indent+1, deps, "directly depends on")
	IndentedPrintln(indent, "(In)direct dependencies:")
	deps = computeTransitiveClosure(deps)
	printDigraph(indent+1, deps, "(in)directly depends on")

	// TODO: warn about any circular dependencies, until we can make a reconciliation plan where
	// relevant resources (i.e. Docker networks) are created simultaneously so that circular
	// dependencies don't prevent successful application. We can safely assume that clients of
	// services can handle a transiently missing service (i.e. missing while only part of the circular
	// dependency has been created so far).

	fmt.Println()
	IndentedPrintln(indent, "Determining package deployment changes...")
	changes := planReconciliation(resolved, deps, apps)
	for _, change := range changes {
		printReconciliationChange(indent, change)
	}
	return changes, dc, nil
}

// resolveDeps returns a map of sets, where each key is the name of a deployment and the
// the value is the set of deployments providing its required resources.
func resolveDeps(satisfiedDeps []forklift.SatisfiedDeplDeps) map[string]map[string]struct{} {
	deps := make(map[string]map[string]struct{})
	for _, satisfied := range satisfiedDeps {
		providers := make(map[string]struct{})
		for _, network := range satisfied.Networks {
			provider := strings.TrimPrefix(network.Provided.Source[0], "deployment ")
			if provider == satisfied.Depl.Name { // i.e. the deployment requires a resource it provides
				continue
			}
			providers[provider] = struct{}{}
		}
		for _, service := range satisfied.Services {
			provider := strings.TrimPrefix(service.Provided.Source[0], "deployment ")
			if provider == satisfied.Depl.Name { // i.e. the deployment requires a resource it provides
				continue
			}
			providers[provider] = struct{}{}
		}
		deps[satisfied.Depl.Name] = providers
	}
	return deps
}

func printDigraph(
	indent int, digraph map[string]map[string]struct{}, edgeType string,
) {
	sortedNodes := make([]string, 0, len(digraph))
	for node := range digraph {
		sortedNodes = append(sortedNodes, node)
	}
	sort.Strings(sortedNodes)
	for _, node := range sortedNodes {
		upstreamNodes := make([]string, 0, len(digraph[node]))
		for dep := range digraph[node] {
			upstreamNodes = append(upstreamNodes, dep)
		}
		sort.Strings(upstreamNodes)
		if len(upstreamNodes) > 0 {
			IndentedPrintf(indent, "%s %s: %+v\n", node, edgeType, upstreamNodes)
		}
	}
}

// computeTransitiveClosure returns, given a set of direct dependencies for every deployment, a set
// of all direct and indirect dependencies for every deployment. This is just the transitive closure
// of the relation expressed by the digraph. Iff the digraph isn't a DAG (i.e. iff it has cycles),
// then each node in the cycle will have an edge directed to itself.
func computeTransitiveClosure(
	digraph map[string]map[string]struct{},
) map[string]map[string]struct{} {
	// Seed the transitive closure with the initial digraph
	closure := make(map[string]map[string]struct{})
	prevChangedNodes := make(map[string]bool)
	changedNodes := make(map[string]bool)
	for node, upstreamNodes := range digraph {
		closure[node] = make(map[string]struct{})
		for upstreamNode := range upstreamNodes {
			closure[node][upstreamNode] = struct{}{}
		}
		prevChangedNodes[node] = true
		changedNodes[node] = true
	}
	// This algorithm is very asymptotically inefficient when long paths exist between nodes, but it's
	// easy to understand, and performance is good enough for a typical use case in dependency
	// resolution where dependency trees should be kept relatively shallow.
	for {
		converged := true
		for node, upstreamNodes := range closure {
			initial := len(upstreamNodes)
			for upstreamNode := range upstreamNodes {
				if !prevChangedNodes[upstreamNode] { // this is just a performance optimization
					continue
				}
				// Add the dependency's own dependencies to the set of dependencies
				transitiveNodes := closure[upstreamNode]
				for transitiveNode := range transitiveNodes {
					upstreamNodes[transitiveNode] = struct{}{}
				}
			}
			final := len(upstreamNodes)
			changedNodes[node] = initial != final
			if changedNodes[node] {
				converged = false
			}
		}
		if converged {
			return closure
		}
		prevChangedNodes = changedNodes
		changedNodes = make(map[string]bool)
	}
}

// invertDeps produces a map associating every deployment to the set of deployments
// depending on it. In other words, it reverses the edges of the DAG of dependencies among
// deployments.
func invertDeps(deps map[string]map[string]struct{}) map[string]map[string]struct{} {
	dependents := make(map[string]map[string]struct{})
	for depl, deps := range deps {
		for dependency := range deps {
			if _, ok := dependents[dependency]; !ok {
				dependents[dependency] = make(map[string]struct{})
			}
			dependents[dependency][depl] = struct{}{}
		}
	}
	return dependents
}

// planReconciliation produces a list of changes to make on the Docker host based on the desired
// list of deployments, a transitive closure of dependencies among those deployments, a transitive
// closure of the deployments depending on each deployment, and a list of
// Docker Compose apps describing the current complete state of the Docker host.
func planReconciliation(
	depls []*forklift.ResolvedDepl, deps map[string]map[string]struct{},
	apps []api.Stack,
) []reconciliationChange {
	deplSet := make(map[string]*forklift.ResolvedDepl)
	for _, depl := range depls {
		deplSet[depl.Name] = depl
	}
	appSet := make(map[string]api.Stack)
	for _, app := range apps {
		appSet[app.Name] = app
	}
	appDeplNames := make(map[string]string)

	changes := make([]reconciliationChange, 0, len(deplSet)+len(appSet))
	for name, depl := range deplSet {
		appDeplNames[getAppName(name)] = name
		definesApp := depl.Pkg.Def.Deployment.DefinesApp()
		app, ok := appSet[getAppName(name)]
		if !ok {
			if definesApp {
				changes = append(changes, reconciliationChange{
					Name: getAppName(name),
					Type: addReconciliationChange,
					Depl: depl,
				})
			}
			continue
		}
		if definesApp {
			changes = append(changes, reconciliationChange{
				Name: getAppName(name),
				Type: updateReconciliationChange,
				Depl: depl,
				App:  app,
			})
		}
	}
	for name, app := range appSet {
		if deplName, ok := appDeplNames[name]; ok {
			if depl, ok := deplSet[deplName]; ok && depl.Pkg.Def.Deployment.DefinesApp() {
				continue
			}
		}
		changes = append(changes, reconciliationChange{
			Name: name,
			Type: removeReconciliationChange,
			App:  app,
		})
	}

	dependents := invertDeps(deps)
	// Sequence the changes such that they can (hopefully) be carried out successfully
	sort.Slice(changes, func(i, j int) bool {
		return compareChanges(changes[i], changes[j], deps, dependents) == pallets.CompareLT
	})
	return changes
}

func getAppName(deplName string) string {
	return strings.ReplaceAll(deplName, "/", "_")
}

// compareChanges returns a comparison for generating a total ordering of reconciliation changes
// so that they are applied in a way that will (hopefully) succeed for all changes. Deps
// should be a transitive closure of dependencies, and dependents should be a transitive closure of
// dependents.
func compareChanges(
	r, s reconciliationChange, deps, dependents map[string]map[string]struct{},
) int {
	// Remove old resources first, in case additions/updates would add overlapping resources.
	if result := compareReconciliationChangesByType(r, s); result != pallets.CompareEQ {
		return result
	}
	// Now r.Depl and s.Depl are either both nil or both non-nil
	if r.Depl == nil && s.Depl == nil {
		return compareDeplNames(r.Name, s.Name)
	}

	// Now r and s are either both removals or both changes/additions
	if result := compareReconciliationChangesByDeplDeps(r, s, deps); result != pallets.CompareEQ {
		return result
	}
	// Now r and s either are in a circular dependency or have no dependency relationships
	if result := compareDeplsByDepCounts(
		r.Depl.Name, s.Depl.Name, deps, dependents,
	); result != pallets.CompareEQ {
		return result
	}
	return compareDeplNames(r.Depl.Name, s.Depl.Name)
}

func compareReconciliationChangesByType(r, s reconciliationChange) int {
	if r.Type == removeReconciliationChange && s.Type != removeReconciliationChange {
		return pallets.CompareLT
	}
	if r.Type != removeReconciliationChange && s.Type == removeReconciliationChange {
		return pallets.CompareGT
	}
	return pallets.CompareEQ
}

func compareReconciliationChangesByDeplDeps(
	r, s reconciliationChange, deps map[string]map[string]struct{},
) int {
	rDependsOnS := false
	if rDeps, ok := deps[r.Depl.Name]; ok {
		_, rDependsOnS = rDeps[s.Depl.Name]
	}
	sDependsOnR := false
	if sDeps, ok := deps[s.Depl.Name]; ok {
		_, sDependsOnR = sDeps[r.Depl.Name]
	}
	if rDependsOnS && !sDependsOnR {
		if r.Type == removeReconciliationChange { // i.e. r and s are both removals
			return pallets.CompareLT // removal r goes before removal s
		}
		return pallets.CompareGT // addition/update r goes after addition/update s
	}
	if !rDependsOnS && sDependsOnR {
		if s.Type == removeReconciliationChange { // i.e. r and s are both removals
			return pallets.CompareGT // removal r goes after removal s
		}
		return pallets.CompareLT // addition/update r goes before addition/update s
	}
	return pallets.CompareEQ
}

func compareDeplsByDepCounts(r, s string, deps, dependents map[string]map[string]struct{}) int {
	// Deployments with greater numbers of dependents go first (needed for correct ordering among
	// unrelated deployments sorted by sort.Slice).
	if len(dependents[r]) > len(dependents[s]) {
		return pallets.CompareLT
	}
	if len(dependents[r]) < len(dependents[s]) {
		return pallets.CompareGT
	}
	// Deployments with greater numbers of deps go first (for aesthetic reasons)
	if len(deps[r]) > len(deps[s]) {
		return pallets.CompareLT
	}
	if len(deps[r]) < len(deps[s]) {
		return pallets.CompareGT
	}
	return pallets.CompareEQ
}

func compareDeplNames(r, s string) int {
	if r < s {
		return pallets.CompareLT
	}
	if r > s {
		return pallets.CompareGT
	}
	return pallets.CompareEQ
}

func printReconciliationChange(indent int, change reconciliationChange) {
	if change.Depl == nil {
		IndentedPrintf(
			indent, "Will %s Compose app %s (from unknown deployment)\n",
			strings.ToLower(change.Type), change.Name,
		)
		return
	}
	IndentedPrintf(
		indent, "Will %s deployment %s as Compose app %s\n",
		strings.ToLower(change.Type), change.Depl.Name, change.Name,
	)
}

// Apply

func ApplyEnv(indent int, env *forklift.FSEnv, loader forklift.FSPkgLoader) error {
	changes, dc, err := computePlan(indent, env, loader)
	if err != nil {
		return errors.Wrap(err, "couldn't compute plan for changes")
	}

	for _, change := range changes {
		if err := applyReconciliationChange(0, change, dc); err != nil {
			return errors.Wrapf(
				err, "couldn't apply '%s' change to Compose app %s", change.Type, change.Name,
			)
		}
	}
	return nil
}

func applyReconciliationChange(
	indent int, change reconciliationChange, dc *docker.Client,
) error {
	fmt.Println()
	switch change.Type {
	default:
		return errors.Errorf("unknown change type '%s'", change.Type)
	case addReconciliationChange:
		IndentedPrintf(
			indent, "Adding package deployment %s as Compose app %s...\n", change.Depl.Name, change.Name,
		)
		if err := deployApp(indent+1, change.Depl, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	case removeReconciliationChange:
		// Note: removeReconciliationChange has a nil Depl field
		IndentedPrintf(indent, "Removing Compose app %s (unknown deployment)...\n", change.Name)
		if err := dc.RemoveApps(context.Background(), []string{change.Name}); err != nil {
			return errors.Wrapf(err, "couldn't remove %s", change.Name)
		}
		return nil
	case updateReconciliationChange:
		IndentedPrintf(
			indent, "Updating package deployment %s as Compose app %s...\n",
			change.Depl.Name, change.Name,
		)
		if err := deployApp(indent+1, change.Depl, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	}
}

func deployApp(indent int, depl *forklift.ResolvedDepl, name string, dc *docker.Client) error {
	if !depl.Pkg.Def.Deployment.DefinesApp() {
		IndentedPrintln(indent, "No Docker Compose app to deploy!")
		return nil
	}

	appDef, err := docker.LoadAppDefinition(
		depl.Pkg.FS, name, depl.Pkg.Def.Deployment.DefinitionFiles, nil,
	)
	if err != nil {
		return err
	}
	if err = dc.DeployApp(context.Background(), appDef, 0); err != nil {
		return errors.Wrapf(err, "couldn't deploy Compose app '%s'", name)
	}
	return nil
}
