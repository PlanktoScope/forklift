package cli

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/compose/v2/pkg/api"
	ggit "github.com/go-git/go-git/v5"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/internal/clients/git"
	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

func GetPalletCache(
	wpath string, pallet *forklift.FSPallet, ensureCache bool,
) (*forklift.FSPalletCache, error) {
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetPalletCache()
	if err != nil {
		return nil, err
	}

	if ensureCache && !cache.Exists() && pallet != nil {
		palletReqs, err := pallet.LoadFSPalletReqs("**")
		if err != nil {
			return nil, errors.Wrap(err, "couldn't check whether the pallet requires any pallets")
		}
		if len(palletReqs) > 0 {
			return nil, errors.New("you first need to cache the pallets specified by your pallet")
		}
	}
	return cache, nil
}

// Print

func PrintCachedPallet(indent int, cache core.Pather, pallet *forklift.FSPallet) error {
	IndentedPrintf(indent, "Cached pallet: %s\n", pallet.Path())
	indent++

	IndentedPrintf(indent, "Forklift version: %s\n", pallet.Def.ForkliftVersion)
	fmt.Println()

	IndentedPrintf(indent, "Version: %s\n", pallet.Version)
	IndentedPrintf(indent, "Path in cache: %s\n", core.GetSubdirPath(cache, pallet.FS.Path()))
	IndentedPrintf(indent, "Description: %s\n", pallet.Def.Pallet.Description)

	readme, err := pallet.LoadReadme()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load readme file for pallet %s@%s from cache", pallet.Path(), pallet.Version,
		)
	}
	IndentedPrintln(indent, "Readme:")
	const widthLimit = 100
	PrintReadme(indent+1, readme, widthLimit)
	return nil
}

func PrintPalletInfo(indent int, pallet *forklift.FSPallet) error {
	IndentedPrintf(indent, "Pallet: %s\n", pallet.Path())
	indent++

	IndentedPrintf(indent, "Forklift version: %s\n", pallet.Def.ForkliftVersion)
	fmt.Println()

	if pallet.Def.Pallet.Path != "" {
		IndentedPrintf(indent, "Path in filesystem: %s\n", pallet.FS.Path())
	}
	IndentedPrintf(indent, "Description: %s\n", pallet.Def.Pallet.Description)
	if pallet.Def.Pallet.ReadmeFile == "" {
		fmt.Println()
	} else {
		readme, err := pallet.LoadReadme()
		if err != nil {
			return errors.Wrapf(err, "couldn't load readme file for pallet %s", pallet.FS.Path())
		}
		IndentedPrintln(indent, "Readme:")
		const widthLimit = 100
		PrintReadme(indent+1, readme, widthLimit)
	}

	return printGitRepoInfo(indent, pallet.FS.Path())
}

func printGitRepoInfo(indent int, palletPath string) error {
	ref, err := git.Head(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query pallet %s for its HEAD", palletPath)
	}
	IndentedPrintf(indent, "Currently on: %s\n", git.StringifyRef(ref))
	// TODO: report any divergence between head and remotes
	if err := printUncommittedChanges(indent+1, palletPath); err != nil {
		return err
	}
	if err := printLocalRefsInfo(indent, palletPath); err != nil {
		return err
	}
	if err := printRemotesInfo(indent, palletPath); err != nil {
		return err
	}
	return nil
}

func printUncommittedChanges(indent int, palletPath string) error {
	status, err := git.Status(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query the pallet %s for its status", palletPath)
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

func printLocalRefsInfo(indent int, palletPath string) error {
	refs, err := git.Refs(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query pallet %s for its refs", palletPath)
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

func printRemotesInfo(indent int, palletPath string) error {
	remotes, err := git.Remotes(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query pallet %s for its remotes", palletPath)
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

// Download

func DownloadRequiredPallets(
	indent int, pallet *forklift.FSPallet, cachePath string,
) (changed bool, err error) {
	loadedPalletReqs, err := pallet.LoadFSPalletReqs("**")
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify pallets")
	}
	changed = false
	for _, req := range loadedPalletReqs {
		downloaded, err := DownloadLockedGitRepoUsingLocalMirror(
			indent, cachePath, req.Path(), req.VersionLock,
		)
		changed = changed || downloaded
		if err != nil {
			return false, errors.Wrapf(
				err, "couldn't download %s at commit %s", req.Path(), req.VersionLock.Def.ShortCommit(),
			)
		}
	}
	return changed, nil
}

// Cache

func CacheAllRequirements(
	pallet *forklift.FSPallet, repoCachePath string, pkgLoader forklift.FSPkgLoader,
	includeDisabled, parallel bool,
) (changed bool, err error) {
	changed, err = CacheStagingRequirements(pallet, repoCachePath)
	if err != nil {
		return false, err
	}
	fmt.Println()

	fmt.Println("Downloading Docker container images specified by the local pallet...")
	if err := DownloadImages(0, pallet, pkgLoader, includeDisabled, parallel); err != nil {
		return false, err
	}
	return changed, nil
}

func CacheStagingRequirements(
	pallet *forklift.FSPallet, repoCachePath string,
) (changed bool, err error) {
	// TODO: download required pallets, once we allow layering pallets; then merge the pallets into
	// a composite before downloading required repos

	fmt.Println("Downloading repos specified by the local pallet...")
	changed, err = DownloadRequiredRepos(0, pallet, repoCachePath)
	if err != nil {
		return false, err
	}

	// TODO: warn if any downloaded repo doesn't appear to be an actual repo, or if any repo's
	// forklift version is incompatible or ahead of the pallet version

	return changed, nil
}

// Check

type ResolvedDeplsLoader interface {
	forklift.PkgReqLoader
	LoadDepls(searchPattern string) ([]forklift.Depl, error)
}

// Check checks the resource constraints among package deployments in the pallet or bundle.
func Check(
	indent int, deplsLoader ResolvedDeplsLoader, pkgLoader forklift.FSPkgLoader,
) ([]*forklift.ResolvedDepl, []forklift.SatisfiedDeplDeps, error) {
	depls, err := deplsLoader.LoadDepls("**/*")
	if err != nil {
		return nil, nil, err
	}
	depls = forklift.FilterDeplsForEnabled(depls)
	resolved, err := forklift.ResolveDepls(deplsLoader, pkgLoader, depls)
	if err != nil {
		return nil, nil, err
	}

	conflicts, err := checkDeplConflicts(indent, resolved)
	if err != nil {
		return nil, nil, err
	}
	satisfied, missingDeps, err := checkDeplDeps(indent, resolved)
	if err != nil {
		return nil, nil, err
	}
	if len(conflicts)+len(missingDeps) > 0 {
		return nil, nil, errors.Errorf(
			"resource constraint checks failed (%d conflicts, %d missing dependencies)",
			len(conflicts), len(missingDeps),
		)
	}
	return resolved, satisfied, nil
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
	if conflict.HasFilesetConflict() {
		IndentedPrintln(indent, "Conflicting filesets:")
		if err := printResConflicts(indent+1, conflict.Filesets); err != nil {
			return errors.Wrap(err, "couldn't print conflicting filesets")
		}
	}
	if conflict.HasFileExportConflict() {
		IndentedPrintln(indent, "Conflicting file exports:")
		if err := printResConflicts(indent+1, conflict.FileExports); err != nil {
			return errors.Wrap(err, "couldn't print conflicting file exports")
		}
	}
	return nil
}

func printResConflicts[Res any](
	indent int, conflicts []core.ResConflict[Res],
) error {
	for _, resourceConflict := range conflicts {
		if err := printResConflict(indent, resourceConflict); err != nil {
			return errors.Wrap(err, "couldn't print resource conflict")
		}
	}
	return nil
}

func printResConflict[Res any](
	indent int, conflict core.ResConflict[Res],
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
	if deps.HasMissingFilesetDep() {
		IndentedPrintln(indent, "Missing filesets:")
		if err := printMissingDeps(indent+1, deps.Filesets); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet fileset dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	return nil
}

func printMissingDeps[Res any](indent int, missingDeps []core.MissingResDep[Res]) error {
	for _, missingDep := range missingDeps {
		if err := printMissingDep(indent, missingDep); err != nil {
			return errors.Wrap(err, "couldn't print unmet resource dependency")
		}
	}
	return nil
}

func printMissingDep[Res any](indent int, missingDep core.MissingResDep[Res]) error {
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

func printDepCandidate[Res any](indent int, candidate core.ResDepCandidate[Res]) error {
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

const (
	addReconciliationChange    = "add"
	removeReconciliationChange = "remove"
	updateReconciliationChange = "update"
)

type ReconciliationChange struct {
	Name string
	Type string
	Depl *forklift.ResolvedDepl // this is nil for an app to be removed
	App  api.Stack              // this is empty for an app which does not yet exist
}

func (c *ReconciliationChange) String() string {
	if c.Depl == nil {
		return fmt.Sprintf("(%s %s)", c.Type, c.Name)
	}
	return fmt.Sprintf("(%s %s)", c.Type, c.Depl.Name)
}

func (c *ReconciliationChange) PlanString() string {
	if c.Depl == nil {
		return fmt.Sprintf("%s Compose app %s (from unknown deployment)", c.Type, c.Name)
	}
	return fmt.Sprintf("%s deployment %s as Compose app %s", c.Type, c.Depl.Name, c.Name)
}

func newAddReconciliationChange(
	deplName string, depl *forklift.ResolvedDepl,
) *ReconciliationChange {
	return &ReconciliationChange{
		Name: getAppName(deplName),
		Type: addReconciliationChange,
		Depl: depl,
	}
}

func getAppName(deplName string) string {
	return strings.ReplaceAll(deplName, "/", "_")
}

func newUpdateReconciliationChange(
	deplName string, depl *forklift.ResolvedDepl, app api.Stack,
) *ReconciliationChange {
	return &ReconciliationChange{
		Name: getAppName(deplName),
		Type: updateReconciliationChange,
		Depl: depl,
		App:  app,
	}
}

func newRemoveReconciliationChange(appName string, app api.Stack) *ReconciliationChange {
	return &ReconciliationChange{
		Name: appName,
		Type: removeReconciliationChange,
		App:  app,
	}
}

// Plan builds a plan for changes to make to the Docker host in order to reconcile it with the
// desired state as expressed by the pallet or bundle. The plan is expressed as a dependency graph
// which can be used to build a partial ordering of the changes (where each change is a node in the
// graph) for concurrent execution, and - if serial execution is required either because the
// parallel arg is set to true or because a dependency cycle was detected - a total ordering of the
// changes for serial (rather than concurrent) execution.
func Plan(
	indent int, deplsLoader ResolvedDeplsLoader, pkgLoader forklift.FSPkgLoader, parallel bool,
) (
	changeDeps structures.Digraph[*ReconciliationChange], serialization []*ReconciliationChange,
	err error,
) {
	dc, err := docker.NewClient()
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't make Docker API client")
	}

	depls, satisfiedDeps, err := Check(indent, deplsLoader, pkgLoader)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't ensure validity")
	}
	// Always skip nonblocking dependency relationships - even for serial execution, they don't need
	// to be considered for a total ordering. And we don't want nonblocking dependency relationships
	// to count towards dependency cycles. And it's simpler to just have the same behavior (and the
	// same resulting dependency graph) regardless of serial vs. concurrent execution.
	deps := forklift.ResolveDeps(satisfiedDeps, true)

	IndentedPrintln(indent, "Determining and ordering package deployment changes...")
	apps, err := dc.ListApps(context.Background())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't list active Docker Compose apps")
	}
	changeDeps, cycles, serialization, err := planChanges(depls, deps, apps, !parallel)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't compute a plan for changes")
	}

	IndentedPrintln(indent, "Ordering relationships:")
	printDigraph(indent+1, changeDeps, "after")
	if len(cycles) > 0 {
		fmt.Println("Detected ordering cycles:")
		for _, cycle := range cycles {
			IndentedPrintf(indent+1, "cycle between: %s\n", cycle)
		}
		if parallel {
			return nil, nil, errors.Errorf(
				"concurrent plan would deadlock due to ordering cycles (try a serial plan instead): %+v",
				cycles,
			)
		}
	}
	if serialization == nil {
		return changeDeps, nil, nil
	}

	fmt.Println()
	IndentedPrintln(indent, "Serialized ordering of package deployment changes:")
	for _, change := range serialization {
		IndentedPrintln(indent+1, change.PlanString())
	}
	return changeDeps, serialization, nil
}

func printDigraph[Node comparable, Digraph structures.MapDigraph[Node]](
	indent int, digraph Digraph, edgeType string,
) {
	sortedNodes := make([]Node, 0, len(digraph))
	for node := range digraph {
		sortedNodes = append(sortedNodes, node)
	}
	sort.Slice(sortedNodes, func(i, j int) bool {
		return fmt.Sprintf("%v", sortedNodes[i]) < fmt.Sprintf("%v", sortedNodes[j])
	})
	for _, node := range sortedNodes {
		printNodeOutboundEdges(indent, digraph, node, edgeType)
	}
}

func printNodeOutboundEdges[Node comparable, Digraph structures.MapDigraph[Node]](
	indent int, digraph Digraph, node Node, edgeType string,
) {
	upstreamNodes := make([]Node, 0, len(digraph[node]))
	for dep := range digraph[node] {
		upstreamNodes = append(upstreamNodes, dep)
	}
	sort.Slice(upstreamNodes, func(i, j int) bool {
		return fmt.Sprintf("%v", upstreamNodes[i]) < fmt.Sprintf("%v", upstreamNodes[j])
	})
	if len(upstreamNodes) == 0 {
		IndentedPrintf(indent, "%v %s nothing", node, edgeType)
	} else {
		IndentedPrintf(indent, "%v %s: %+v", node, edgeType, upstreamNodes)
	}
	fmt.Println()
}

// planChanges builds a dependency graph of changes to make to the Docker host (as a plan for
// concurrent execution), for a given list of resolved package deployments, a precomputed graph of
// direct dependency relationships between them, and a list of currently active Compose apps.
// This function also identifies any cycles in the returned dependency graph.
// If the serialize arg is set to true, this function will also compute a non-nil sequential order
// for executing the changes serially (rather than concurrently); otherwise, a nil sequential order
// will be returned.
func planChanges(
	depls []*forklift.ResolvedDepl, deplDirectDeps structures.Digraph[string], apps []api.Stack,
	serialize bool,
) (
	changeDirectDeps structures.Digraph[*ReconciliationChange], cycles [][]*ReconciliationChange,
	serialization []*ReconciliationChange, err error,
) {
	// TODO: make a reconciliation plan where relevant resources (i.e. Docker networks) are created
	// simultaneously/independently so that circular dependencies for those resouces won't prevent
	// successful application.
	changes, err := identifyReconciliationChanges(depls, apps)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "couldn't identify the changes to make")
	}

	changeDirectDeps = computeChangeDeps(changes, deplDirectDeps)
	changeIndirectDeps := changeDirectDeps.ComputeTransitiveClosure()
	cycles = changeIndirectDeps.IdentifyCycles()
	if !serialize {
		return changeDirectDeps, cycles, nil, nil
	}

	// Serialize changes with a total ordering
	dependents := changeIndirectDeps.Invert()
	sort.Slice(changes, func(i, j int) bool {
		return compareChangesTotal(
			changes[i], changes[j], changeIndirectDeps, dependents,
		) == core.CompareLT
	})
	return changeDirectDeps, cycles, changes, nil
}

// identifyReconciliationChanges builds an arbitrarily-ordered list of changes to carry out to
// reconcile the desired list of deployments with the actual list of active Docker Compose apps.
func identifyReconciliationChanges(
	depls []*forklift.ResolvedDepl, apps []api.Stack,
) ([]*ReconciliationChange, error) {
	deplsByName := make(map[string]*forklift.ResolvedDepl)
	for _, depl := range depls {
		deplsByName[depl.Name] = depl
	}
	appsByName := make(map[string]api.Stack)
	for _, app := range apps {
		appsByName[app.Name] = app
	}
	composeAppDefinerSet, err := identifyComposeAppDefiners(deplsByName)
	if err != nil {
		return nil, err
	}

	appDeplNames := make(map[string]string)
	changes := make([]*ReconciliationChange, 0, len(depls)+len(apps))
	for name, depl := range deplsByName {
		appDeplNames[getAppName(name)] = name
		app, ok := appsByName[getAppName(name)]
		if !ok {
			if composeAppDefinerSet.Has(name) {
				changes = append(changes, newAddReconciliationChange(name, depl))
			}
			continue
		}
		if composeAppDefinerSet.Has(name) {
			changes = append(changes, newUpdateReconciliationChange(name, depl, app))
		}
	}
	for name, app := range appsByName {
		if deplName, ok := appDeplNames[name]; ok {
			if composeAppDefinerSet.Has(deplName) {
				continue
			}
		}
		changes = append(changes, newRemoveReconciliationChange(name, app))
	}
	return changes, nil
}

// identifyComposeAppDefiners builds a set of the names of deployments which define Compose apps.
func identifyComposeAppDefiners(
	depls map[string]*forklift.ResolvedDepl,
) (structures.Set[string], error) {
	composeAppDefinerSet := make(structures.Set[string])
	for _, depl := range depls {
		definesApp, err := depl.DefinesApp()
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine whether package deployment %s defines a Compose app", depl.Name,
			)
		}
		if definesApp {
			composeAppDefinerSet.Add(depl.Name)
		}
	}
	return composeAppDefinerSet, nil
}

// computeChangeDeps produces a dependency graph of changes to make on the Docker host based on the
// desired list of deployments, a graph of direct dependencies among those deployments, and a list
// of Docker Compose apps describing the current complete state of the Docker host. The returned
// dependency graph is a map between each reconciliation change and the respective set of any other
// reconciliation changes which must be completed first.
func computeChangeDeps(
	changes []*ReconciliationChange, directDeps structures.Digraph[string],
) structures.Digraph[*ReconciliationChange] {
	removalChanges := make(map[string]*ReconciliationChange)    // keyed by app name
	nonremovalChanges := make(map[string]*ReconciliationChange) // keyed by depl name
	graph := make(structures.Digraph[*ReconciliationChange])
	for _, change := range changes {
		graph.AddNode(change)
		if change.Type == removeReconciliationChange {
			removalChanges[change.Name] = change
			continue
		}
		nonremovalChanges[change.Depl.Name] = change
	}
	// FIXME: ideally we would order the removal changes based on dependency relationships between
	// the Compose apps, e.g. with networks. With removals we don't have deployments which would
	// tell us about Docker resource dependency relationships, so we'd need to determine this from
	// Docker. If app r depends on a resource provided by app s, then app r must be removed first -
	// so the removal of app s depends upon the removal of app r.
	// Remove old resources first, in case additions/updates would add overlapping resources.
	for _, change := range nonremovalChanges {
		for _, removalChange := range removalChanges {
			graph.AddEdge(change, removalChange)
		}
	}
	for _, dependent := range nonremovalChanges {
		for deplName := range directDeps[dependent.Depl.Name] {
			if dependency, ok := nonremovalChanges[deplName]; ok {
				graph.AddEdge(dependent, dependency)
			}
		}
	}

	return graph
}

// compareChangesTotal returns a comparison for generating a total ordering of reconciliation
// changes so that they are applied serially and sequentially in a way that will (hopefully) succeed
// for all changes. deps should be a transitive closure of dependencies between changes, and
// dependents should be the inverse of deps.
// This function returns -1 if r should occur before s and 1 if s should occur before r.
func compareChangesTotal(
	r, s *ReconciliationChange, deps, dependents structures.TransitiveClosure[*ReconciliationChange],
) int {
	// Apply the partial ordering from dependencies
	if result := compareReconciliationChangesByDeps(r, s, deps); result != core.CompareEQ {
		return result
	}

	// Now r and s either are in a circular dependency or have no dependency relationships
	if result := compareDeplsByDepCounts(r, s, deps, dependents); result != core.CompareEQ {
		return result
	}

	// Compare by names as a last resort
	if r.Depl != nil && s.Depl != nil {
		return compareDeplNames(r.Depl.Name, s.Depl.Name)
	}
	return compareDeplNames(r.Name, s.Name)
}

func compareReconciliationChangesByDeps(
	r, s *ReconciliationChange, deps structures.TransitiveClosure[*ReconciliationChange],
) int {
	rDependsOnS := deps.HasEdge(r, s)
	sDependsOnR := deps.HasEdge(s, r)
	if rDependsOnS && !sDependsOnR {
		return core.CompareGT
	}
	if !rDependsOnS && sDependsOnR {
		return core.CompareLT
	}
	return core.CompareEQ
}

func compareDeplsByDepCounts(
	r, s *ReconciliationChange, deps, dependents structures.TransitiveClosure[*ReconciliationChange],
) int {
	// Deployments with greater numbers of dependents go first (needed for correct ordering among
	// unrelated deployments sorted by sort.Slice).
	if len(dependents[r]) > len(dependents[s]) {
		return core.CompareLT
	}
	if len(dependents[r]) < len(dependents[s]) {
		return core.CompareGT
	}
	// Deployments with greater numbers of dependencies go first (for aesthetic reasons)
	if len(deps[r]) > len(deps[s]) {
		return core.CompareLT
	}
	if len(deps[r]) < len(deps[s]) {
		return core.CompareGT
	}
	return core.CompareEQ
}

func compareDeplNames(r, s string) int {
	if r < s {
		return core.CompareLT
	}
	if r > s {
		return core.CompareGT
	}
	return core.CompareEQ
}

// Stage

func StagePallet(
	pallet *forklift.FSPallet, stageStore *forklift.FSStageStore, repoCache forklift.PathedRepoCache,
	exportPath, toolVersion, bundleMinVersion, newBundleForkliftVersion string,
	skipImageCaching, parallel, ignoreToolVersion bool,
) (index int, err error) {
	index, err = stageStore.AllocateNew()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't allocate a directory for staging")
	}
	fmt.Printf("Bundling pallet as stage %d for staged application...\n", index)
	if err = buildBundle(
		pallet, repoCache, newBundleForkliftVersion,
		path.Join(stageStore.FS.Path(), fmt.Sprintf("%d", index)),
	); err != nil {
		return index, errors.Wrapf(err, "couldn't bundle pallet %s as stage %d", pallet.Path(), index)
	}
	if err = SetNextStagedBundle(
		stageStore, index, exportPath, toolVersion, bundleMinVersion,
		skipImageCaching, parallel, ignoreToolVersion,
	); err != nil {
		return index, errors.Wrapf(
			err, "couldn't prepare staged pallet bundle %d to be applied next", index,
		)
	}
	return index, nil
}

func buildBundle(
	pallet *forklift.FSPallet, repoCache forklift.PathedRepoCache,
	forkliftVersion, outputPath string,
) (err error) {
	outputBundle := forklift.NewFSBundle(outputPath)
	// TODO: once we can overlay pallets, save the result of overlaying the pallets to a `overlay`
	// subdir
	outputBundle.Manifest, err = newBundleManifest(pallet, repoCache, forkliftVersion)
	if err != nil {
		return errors.Wrapf(err, "couldn't create bundle manifest for %s", outputBundle.FS.Path())
	}

	depls, _, err := Check(0, pallet, repoCache)
	if err != nil {
		return errors.Wrap(err, "couldn't ensure pallet validity")
	}
	for _, depl := range depls {
		if err := outputBundle.AddResolvedDepl(depl); err != nil {
			return err
		}
	}

	if err := outputBundle.SetBundledPallet(pallet); err != nil {
		return err
	}
	if err = outputBundle.WriteRepoDefFile(); err != nil {
		return err
	}
	if err = outputBundle.WriteFileExports(); err != nil {
		return err
	}
	return outputBundle.WriteManifestFile()
}

func newBundleManifest(
	pallet *forklift.FSPallet, repoCache forklift.PathedRepoCache, forkliftVersion string,
) (forklift.BundleManifest, error) {
	desc := forklift.BundleManifest{
		ForkliftVersion: forkliftVersion,
		Pallet: forklift.BundlePallet{
			Path:        pallet.Path(),
			Description: pallet.Def.Pallet.Description,
		},
		Includes: forklift.BundleInclusions{
			Pallets: make(map[string]forklift.BundlePalletInclusion),
			Repos:   make(map[string]forklift.BundleRepoInclusion),
		},
		Deploys: make(map[string]forklift.DeplDef),
		Exports: make(map[string][]string),
	}
	desc.Pallet.Version, desc.Pallet.Clean = checkGitRepoVersion(pallet.FS.Path())
	palletReqs, err := pallet.LoadFSPalletReqs("**")
	if err != nil {
		return desc, errors.Wrapf(err, "couldn't determine pallets required by pallet %s", pallet.Path())
	}
	// TODO: once we can overlay pallets, the description of pallet & repo inclusions should probably
	// be made from the result of overlaying. We could also describe pre-overlay requirements from the
	// bundled pallet, in desc.Pallet.Requires.
	for _, req := range palletReqs {
		inclusion := forklift.BundlePalletInclusion{Req: req.PalletReq}
		// TODO: also check for overridden pallets
		desc.Includes.Pallets[req.RequiredPath] = inclusion
	}
	repoReqs, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return desc, errors.Wrapf(err, "couldn't determine repos required by pallet %s", pallet.Path())
	}
	for _, req := range repoReqs {
		desc.Includes.Repos[req.RequiredPath] = newBundleRepoInclusion(req, repoCache)
	}
	return desc, nil
}

func checkGitRepoVersion(palletPath string) (version string, clean bool) {
	gitRepo, err := git.Open(filepath.FromSlash(palletPath))
	if err != nil {
		return "", false
	}
	commit, err := gitRepo.GetHead()
	if err != nil {
		return "", false
	}
	versionLock, err := lockCommit(gitRepo, commit)
	if err != nil {
		return "", false
	}
	versionString, err := versionLock.Version()
	if err != nil {
		return "", false
	}
	status, err := gitRepo.Status()
	if err != nil {
		return versionString, false
	}
	return versionString, status.IsClean()
}

func newBundleRepoInclusion(
	req *forklift.FSRepoReq, repoCache forklift.PathedRepoCache,
) forklift.BundleRepoInclusion {
	inclusion := forklift.BundleRepoInclusion{Req: req.RepoReq}
	for {
		if repoCache == nil {
			return inclusion
		}
		layeredCache, ok := repoCache.(*forklift.LayeredRepoCache)
		if !ok {
			return inclusion
		}
		overlay := layeredCache.Overlay
		if overlay == nil {
			repoCache = layeredCache.Underlay
			continue
		}

		if repo, err := overlay.LoadFSRepo(
			req.RequiredPath, req.VersionLock.Version,
		); err == nil { // i.e. the repo was overridden
			inclusion.Override.Path = repo.FS.Path()
			inclusion.Override.Version, inclusion.Override.Clean = checkGitRepoVersion(repo.FS.Path())
			return inclusion
		}
		repoCache = layeredCache.Underlay
	}
}
