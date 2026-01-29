package cli

import (
	"cmp"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	res "github.com/forklift-run/forklift/pkg/resources"
)

type ResolvedDeplsLoader interface {
	fplt.PkgReqLoader
	LoadDepls(searchPattern string) ([]fplt.Depl, error)
}

// Check checks the validity of the pallet or bundle. It prints check failures.
func Check(
	indent int, deplsLoader ResolvedDeplsLoader, pkgLoader fplt.FSPkgLoader,
) ([]*fplt.ResolvedDepl, []fplt.SatisfiedDeplDeps, error) {
	depls, err := deplsLoader.LoadDepls("**/*")
	if err != nil {
		return nil, nil, err
	}
	depls = fplt.FilterDeplsForEnabled(depls)
	resolved, err := fplt.ResolveDepls(deplsLoader, pkgLoader, depls)
	if err != nil {
		return nil, nil, err
	}

	fileExportsErr := checkFileExports(indent, os.Stderr, resolved)
	satisfied, resourcesErr := checkResources(indent, os.Stderr, resolved)
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
// to ensure that the source paths of those file exports are all valid. It prints check failures.
func checkFileExports(indent int, out io.Writer, depls []*fplt.ResolvedDepl) error {
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
			case fpkg.FileExportSourceTypeLocal, "":
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

	IndentedFprintln(indent, out, "Found invalid file exports among deployments:")
	indent++
	slices.Sort(invalidDeplNames)
	for _, depl := range depls {
		invalid := invalidFileExports[depl.Name]
		if len(invalid) == 0 {
			continue
		}
		printInvalidDeplFileExports(indent, out, depl, invalid)
	}
	return errors.Errorf(
		"file export checks failed (%d invalid exports)", len(invalidFileExports),
	)
}

func printInvalidDeplFileExports(
	indent int, out io.Writer, depl *fplt.ResolvedDepl, invalid []invalidFileExport,
) {
	IndentedFprintf(indent, out, "Deployment %s:\n", depl.Name)
	indent++
	for _, invalidFileExport := range invalid {
		BulletedFprintf(indent, out, "File export source: %s\n", invalidFileExport.sourcePath)
		IndentedFprintf(indent+1, out, "File export target: %s\n", invalidFileExport.targetPath)
		IndentedFprintf(indent+1, out, "Error: %s\n", invalidFileExport.err.Error())
	}
}

func checkFileOrSymlink(fsys ffs.PathedFS, file string) error {
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
// It prints check failures.
func checkResources(
	indent int, out io.Writer, depls []*fplt.ResolvedDepl,
) ([]fplt.SatisfiedDeplDeps, error) {
	conflicts, err := checkDeplConflicts(indent, out, depls)
	if err != nil {
		return nil, err
	}
	satisfied, missingDeps, err := checkDeplDeps(indent, out, depls)
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
	indent int, out io.Writer, depls []*fplt.ResolvedDepl,
) ([]fplt.DeplConflict, error) {
	conflicts, err := fplt.CheckDeplConflicts(depls)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't check for conflicts among deployments")
	}
	if len(conflicts) > 0 {
		IndentedFprintln(indent, out, "Found resource conflicts among deployments:")
	}
	for _, conflict := range conflicts {
		if err = printDeplConflict(1, out, conflict); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't print resource conflicts among deployments %s and %s",
				conflict.First.Name, conflict.Second.Name,
			)
		}
	}
	return conflicts, nil
}

func printDeplConflict(indent int, out io.Writer, conflict fplt.DeplConflict) error {
	IndentedFprintf(indent, out, "Between %s and %s:\n", conflict.First.Name, conflict.Second.Name)
	indent++

	if conflict.HasNameConflict() {
		IndentedFprintln(indent, out, "Conflicting deployment names")
	}
	if conflict.HasListenerConflict() {
		IndentedFprintln(indent, out, "Conflicting host port listeners:")
		if err := printResConflicts(indent+1, out, conflict.Listeners); err != nil {
			return errors.Wrap(err, "couldn't print conflicting host port listeners")
		}
	}
	if conflict.HasNetworkConflict() {
		IndentedFprintln(indent, out, "Conflicting Docker networks:")
		if err := printResConflicts(indent+1, out, conflict.Networks); err != nil {
			return errors.Wrap(err, "couldn't print conflicting docker networks")
		}
	}
	if conflict.HasServiceConflict() {
		IndentedFprintln(indent, out, "Conflicting network services:")
		if err := printResConflicts(indent+1, out, conflict.Services); err != nil {
			return errors.Wrap(err, "couldn't print conflicting network services")
		}
	}
	if conflict.HasFilesetConflict() {
		IndentedFprintln(indent, out, "Conflicting filesets:")
		if err := printResConflicts(indent+1, out, conflict.Filesets); err != nil {
			return errors.Wrap(err, "couldn't print conflicting filesets")
		}
	}
	if conflict.HasFileExportConflict() {
		IndentedFprintln(indent, out, "Conflicting file exports:")
		if err := printResConflicts(indent+1, out, conflict.FileExports); err != nil {
			return errors.Wrap(err, "couldn't print conflicting file exports")
		}
	}
	return nil
}

func printResConflicts[Res any](
	indent int, out io.Writer, conflicts []res.Conflict[Res, []string],
) error {
	for _, resourceConflict := range conflicts {
		if err := printResConflict(indent, out, resourceConflict); err != nil {
			return errors.Wrap(err, "couldn't print resource conflict")
		}
	}
	return nil
}

func printResConflict[Res any](
	indent int, out io.Writer, conflict res.Conflict[Res, []string],
) error {
	BulletedFprintf(indent, out, "Conflicting resource from %s:\n", conflict.First.Origin[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResOrigin(indent+1, out, conflict.First.Origin[1:])
	if err := IndentedFprintYaml(resourceIndent+1, out, conflict.First.Res); err != nil {
		return errors.Wrap(err, "couldn't print first resource")
	}
	IndentedFprintf(indent, out, "Conflicting resource from %s:\n", conflict.Second.Origin[0])
	resourceIndent = printResOrigin(indent+1, out, conflict.Second.Origin[1:])
	if err := IndentedFprintYaml(resourceIndent+1, out, conflict.Second.Res); err != nil {
		return errors.Wrap(err, "couldn't print second resource")
	}

	IndentedFprint(indent, out, "Resources are conflicting because of:")
	if len(conflict.Errs) == 0 {
		_, _ = fmt.Fprint(out, " (unknown)")
	}
	_, _ = fmt.Fprintln(out)
	for _, err := range conflict.Errs {
		BulletedFprintf(indent+1, out, "%s\n", err)
	}
	return nil
}

func printResOrigin(indent int, out io.Writer, origin []string) (finalIndent int) {
	for i, line := range origin {
		finalIndent = indent + i
		IndentedFprintf(finalIndent, out, "%s:", line)
		_, _ = fmt.Fprintln(out)
	}
	return finalIndent
}

func checkDeplDeps(
	indent int, out io.Writer, depls []*fplt.ResolvedDepl,
) (satisfied []fplt.SatisfiedDeplDeps, missing []fplt.MissingDeplDeps, err error) {
	if satisfied, missing, err = fplt.CheckDeplDeps(depls); err != nil {
		return nil, nil, errors.Wrap(err, "couldn't check dependencies among deployments")
	}
	if len(missing) > 0 {
		IndentedFprintln(indent, out, "Found unmet resource dependencies among deployments:")
	}
	for _, missingDep := range missing {
		if err := printMissingDeplDep(1, out, missingDep); err != nil {
			return nil, nil, err
		}
	}
	return satisfied, missing, nil
}

func printMissingDeplDep(indent int, out io.Writer, deps fplt.MissingDeplDeps) error {
	IndentedFprintf(indent, out, "For %s:\n", deps.Depl.Name)
	indent++

	if deps.HasMissingNetworkDep() {
		IndentedFprintln(indent, out, "Missing Docker networks:")
		if err := printMissingDeps(indent+1, out, deps.Networks); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet Docker network dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	if deps.HasMissingServiceDep() {
		IndentedFprintln(indent, out, "Missing network services:")
		if err := printMissingDeps(indent+1, out, deps.Services); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet network service dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	if deps.HasMissingFilesetDep() {
		IndentedFprintln(indent, out, "Missing filesets:")
		if err := printMissingDeps(indent+1, out, deps.Filesets); err != nil {
			return errors.Wrapf(
				err, "couldn't print unmet fileset dependencies of deployment %s", deps.Depl.Name,
			)
		}
	}
	return nil
}

func printMissingDeps[Res any](
	indent int, out io.Writer, missingDeps []res.MissingDep[Res, []string],
) error {
	for _, missingDep := range missingDeps {
		if err := printMissingDep(indent, out, missingDep); err != nil {
			return errors.Wrap(err, "couldn't print unmet resource dependency")
		}
	}
	return nil
}

func printMissingDep[Res any](
	indent int,
	out io.Writer,
	missingDep res.MissingDep[Res, []string],
) error {
	BulletedFprintf(indent, out, "Resource required by %s:\n", missingDep.Required.Origin[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResOrigin(indent+1, out, missingDep.Required.Origin[1:])
	if err := IndentedFprintYaml(resourceIndent+1, out, missingDep.Required.Res); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}
	IndentedFprintln(indent, out, "Best candidates to meet requirement:")
	// indent++

	// TODO: find a more intelligent way to determine what candidates to print; right now the signal
	// to noise ratio is too low:
	/*
		for _, candidate := range missingDep.BestCandidates {
			if err := printDepCandidate(indent, out, candidate); err != nil {
				return errors.Wrap(err, "couldn't print dependency candidate")
			}
		}
	*/
	return nil
}

/*
func printDepCandidate[Res any](
	indent int, out io.Writer, candidate fpkg.ResDepCandidate[Res],
) error {
	BulletedFprintf(indent, out, "Candidate resource from %s:\n", candidate.Provided.Origin[0])
	indent++ // because the bullet adds an indentation level
	resourceIndent := printResOrigin(indent+1, out, candidate.Provided.Origin[1:])
	if err := IndentedFprintYaml(resourceIndent+1, out, candidate.Provided.Res); err != nil {
		return errors.Wrap(err, "couldn't print resource")
	}

	IndentedFprintln(indent, out, "Candidate doesn't meet requirement because of:")
	indent++
	for _, err := range candidate.Errs {
		BulletedFprintf(indent, out, "%s\n", err)
	}
	return nil
}
*/
