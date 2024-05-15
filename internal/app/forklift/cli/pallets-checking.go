package cli

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
)

type ResolvedDeplsLoader interface {
	forklift.PkgReqLoader
	LoadDepls(searchPattern string) ([]forklift.Depl, error)
}

// Check checks the validity of the pallet or bundle.
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

	fileExportsErr := checkFileExports(indent, resolved)
	satisfied, resourcesErr := checkResources(indent, resolved)
	// FIXME: it'd be better to use errors.Join from go's errors package, but we're using
	// github.com/pkg/errors which doesn't have a Join function...
	if fileExportsErr != nil {
		return resolved, satisfied, fileExportsErr
	}
	return resolved, satisfied, resourcesErr
}

type invalidFileExport struct {
	sourcePath string
	targetPath string
	err        error
}

// checkFileExports checks the file exports of all package deployments in the pallet or bundle
// to ensure that the source paths of those file exports are all valid.
func checkFileExports(indent int, depls []*forklift.ResolvedDepl) error {
	invalidDeplNames := make([]string, 0, len(depls))
	invalidFileExports := make(map[string][]invalidFileExport)
	for _, depl := range depls {
		exports, err := depl.GetFileExports()
		if err != nil {
			return errors.Wrapf(err, "couldn't determine file exports for deployment %s", depl.Name)
		}
		for _, export := range exports {
			switch export.SourceType {
			default:
				// TODO: should we also check file exports from files in the cache of downloaded files?
				continue
			case core.FileExportSourceTypeLocal, "":
			}
			sourcePath := cmp.Or(export.Source, export.Target)
			if err = checkFileOrSymlink(depl.Pkg.FS, sourcePath); err != nil {
				invalidFileExports[depl.Name] = append(
					invalidFileExports[depl.Name],
					invalidFileExport{
						sourcePath: sourcePath,
						targetPath: export.Target,
						err:        err,
					},
				)
			}
		}
	}
	if len(invalidFileExports) == 0 {
		return nil
	}

	IndentedPrintln(indent, "Found invalid file exports among deployments:")
	indent++
	slices.Sort(invalidDeplNames)
	for _, depl := range depls {
		invalid := invalidFileExports[depl.Name]
		if len(invalid) == 0 {
			continue
		}
		printInvalidDeplFileExports(indent, depl, invalid)
	}
	return errors.Errorf(
		"file export checks failed (%d invalid exports)", len(invalidFileExports),
	)
}

func printInvalidDeplFileExports(
	indent int, depl *forklift.ResolvedDepl, invalid []invalidFileExport,
) {
	IndentedPrintf(indent, "Deployment %s:\n", depl.Name)
	indent++
	for _, invalidFileExport := range invalid {
		BulletedPrintf(indent, "File export source: %s\n", invalidFileExport.sourcePath)
		IndentedPrintf(indent+1, "File export target: %s\n", invalidFileExport.targetPath)
		IndentedPrintf(indent+1, "Error: %s\n", invalidFileExport.err.Error())
	}
}

func checkFileOrSymlink(fsys core.PathedFS, file string) error {
	if _, err := fs.Stat(fsys, file); err == nil {
		return nil
	}
	// fs.Stat will return an error if the sourcePath exists but is a symlink pointing to a
	// nonexistent location. Really we want fs.Lstat (which is not implemented yet); until fs.Lstat
	// is implemented, when we get an error when we'll just check if a DirEntry exists for the path
	// (and if so, we'll assume the file is valid).
	dirEntries, err := fs.ReadDir(fsys, path.Dir(file))
	if err != nil {
		return err
	}
	for _, dirEntry := range dirEntries {
		if dirEntry.Name() == path.Base(file) {
			return nil
		}
	}
	return errors.Errorf(
		"couldn't find %s in %s", path.Base(file), path.Join(fsys.Path(), path.Dir(file)),
	)
}

// checkResources checks the resource constraints among package deployments in the pallet or bundle.
func checkResources(
	indent int, depls []*forklift.ResolvedDepl,
) ([]forklift.SatisfiedDeplDeps, error) {
	conflicts, err := checkDeplConflicts(indent, depls)
	if err != nil {
		return nil, err
	}
	satisfied, missingDeps, err := checkDeplDeps(indent, depls)
	if err != nil {
		return nil, err
	}
	if len(conflicts)+len(missingDeps) > 0 {
		return nil, errors.Errorf(
			"resource constraint checks failed (%d conflicts, %d missing dependencies)",
			len(conflicts), len(missingDeps),
		)
	}
	return satisfied, nil
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
