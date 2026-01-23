package cli

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/internal/clients/git"
	ffs "github.com/forklift-run/forklift/pkg/fs"
)

func FprintCachedPallet(
	indent int, out io.Writer, cache ffs.Pather, pallet *forklift.FSPallet, printHeader bool,
) error {
	if printHeader {
		IndentedFprintf(indent, out, "Cached pallet: %s\n", pallet.Path())
		indent++
	}

	IndentedFprintf(indent, out, "Forklift version: %s\n", pallet.Decl.ForkliftVersion)
	_, _ = fmt.Fprintln(out)

	IndentedFprintf(indent, out, "Version: %s\n", pallet.Version)
	if ffs.CoversPath(cache, pallet.FS.Path()) {
		IndentedFprintf(indent, out, "Path in cache: %s\n", ffs.GetSubdirPath(cache, pallet.FS.Path()))
	} else {
		// Note: this is used when the pallet is replaced by an overlay from outside the cache
		IndentedFprintf(
			indent, out, "Absolute path (replacing any cached copy): %s\n", pallet.FS.Path(),
		)
	}
	IndentedFprintf(indent, out, "Description: %s\n", pallet.Decl.Pallet.Description)

	if err := fprintReadme(indent, out, pallet); err != nil {
		return errors.Wrapf(
			err, "couldn't preview readme file for pallet %s@%s from cache",
			pallet.Path(), pallet.Version,
		)
	}

	_, _ = fmt.Fprintln(out)
	if err := fprintRepoPkgs(indent, out, pallet.Repo); err != nil {
		return errors.Wrapf(err, "couldn't list packages provided by pallet %s", pallet.Path())
	}

	_, _ = fmt.Fprintln(out)
	if err := fprintPalletDepls(indent, out, pallet); err != nil {
		return errors.Wrapf(
			err, "couldn't list package deployments in by pallet %s", pallet.Path(),
		)
	}

	_, _ = fmt.Fprintln(out)
	if err := fprintPalletFeatures(indent, out, pallet); err != nil {
		return errors.Wrapf(
			err, "couldn't list importable features provided by pallet %s", pallet.Path(),
		)
	}
	return nil
}

func fprintPalletDepls(indent int, out io.Writer, pallet *forklift.FSPallet) error {
	IndentedFprint(indent, out, "Package deployments:")
	depls, err := pallet.LoadDepls("**/*")
	if err != nil {
		return err
	}
	if len(depls) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent += 1
	for _, depl := range depls {
		BulletedFprintf(indent, out, "%s: %s", depl.Name, depl.Decl.Package)
		slices.Sort(depl.Decl.Features)
		if len(depl.Decl.Features) > 0 {
			_, _ = fmt.Fprintf(out, " +[%s]", strings.Join(depl.Decl.Features, ", "))
		}
		if depl.Decl.Disabled {
			_, _ = fmt.Fprint(out, " (disabled)")
		}
		_, _ = fmt.Fprintln(out)
	}
	return nil
}

func fprintPalletFeatures(indent int, out io.Writer, pallet *forklift.FSPallet) error {
	IndentedFprint(indent, out, "Importable features:")
	imps, err := pallet.LoadFeatures("**/*")
	if err != nil {
		return err
	}
	if len(imps) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent += 1
	for _, imp := range imps {
		BulletedFprintf(indent, out, "%s\n", imp.Name)
	}
	return nil
}

func FprintPalletInfo(indent int, out io.Writer, pallet *forklift.FSPallet) error {
	IndentedFprintf(indent, out, "Pallet: %s\n", pallet.Path())
	indent++

	IndentedFprintf(indent, out, "Forklift version: %s\n", pallet.Decl.ForkliftVersion)
	_, _ = fmt.Fprintln(out)

	if pallet.Decl.Pallet.Path != "" {
		IndentedFprintf(indent, out, "Path in filesystem: %s\n", pallet.FS.Path())
	}
	IndentedFprintf(indent, out, "Description: %s\n", pallet.Decl.Pallet.Description)
	if pallet.Decl.Pallet.ReadmeFile == "" {
		_, _ = fmt.Fprintln(out)
	} else if err := fprintReadme(indent, out, pallet); err != nil {
		return errors.Wrapf(err, "couldn't preview readme file for pallet %s", pallet.FS.Path())
	}

	_, _ = fmt.Fprintln(out)
	if err := fprintGitRepoInfo(indent, out, pallet.FS.Path()); err != nil {
		return errors.Wrapf(
			err, "couldn't show information about local git repo for pallet %s", pallet.Path(),
		)
	}

	// Note: we don't automatically print the list of package deployments, because it'd require us to
	// merge the pallet before printing it.
	return nil
}

func fprintGitRepoInfo(indent int, out io.Writer, palletPath string) error {
	ref, err := git.Head(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query pallet %s for its HEAD", palletPath)
	}
	IndentedFprintf(indent, out, "Currently on: %s\n", git.StringifyRef(ref))
	// TODO: report any divergence between head and remotes
	if err := fprintUncommittedChanges(indent+1, out, palletPath); err != nil {
		return err
	}
	if err := fprintLocalRefsInfo(indent, out, palletPath); err != nil {
		return err
	}
	if err := fprintRemotesInfo(indent, out, palletPath); err != nil {
		return err
	}
	return nil
}

func fprintUncommittedChanges(indent int, out io.Writer, palletPath string) error {
	status, err := git.Status(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query the pallet %s for its status", palletPath)
	}
	IndentedFprint(indent, out, "Uncommitted changes:")
	if len(status) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent++

	for file, status := range status {
		if status.Staging == git.StatusUnmodified && status.Worktree == git.StatusUnmodified {
			continue
		}
		if status.Staging == git.StatusRenamed {
			file = fmt.Sprintf("%s -> %s", file, status.Extra)
		}
		BulletedFprintf(indent, out, "%c%c %s\n", status.Staging, status.Worktree, file)
	}
	return nil
}

func fprintLocalRefsInfo(indent int, out io.Writer, palletPath string) error {
	refs, err := git.Refs(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query pallet %s for its refs", palletPath)
	}

	IndentedFprint(indent, out, "References:")
	if len(refs) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent++

	for _, ref := range refs {
		BulletedFprintf(indent, out, "%s\n", git.StringifyRef(ref))
	}

	return nil
}

func fprintRemotesInfo(indent int, out io.Writer, palletPath string) error {
	remotes, err := git.Remotes(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query pallet %s for its remotes", palletPath)
	}

	IndentedFprint(indent, out, "Remotes:")
	if len(remotes) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent++

	SortRemotes(remotes)
	printCacheMirrorRemote := false
	for _, remote := range remotes {
		if remote.Config().Name == ForkliftCacheMirrorRemoteName && !printCacheMirrorRemote {
			IndentedFprintf(
				indent, out,
				"%s: (skipped because origin was successfully queried)\n", remote.Config().Name,
			)
			continue
		}

		if err := fprintRemoteInfo(
			indent, out, remote,
		); err != nil && remote.Config().Name == OriginRemoteName {
			printCacheMirrorRemote = true
		}
	}
	return nil
}

func SortRemotes(remotes []*ggit.Remote) {
	slices.SortFunc(remotes, func(a, b *ggit.Remote) int {
		if a.Config().Name == OriginRemoteName {
			return -1
		}
		if b.Config().Name == OriginRemoteName {
			return 1
		}
		return cmp.Compare(a.Config().Name, b.Config().Name)
	})
}

func fprintRemoteInfo(indent int, out io.Writer, remote *ggit.Remote) error {
	config := remote.Config()
	IndentedFprintf(indent, out, "%s:\n", config.Name)
	indent++

	IndentedFprint(indent, out, "URLs:")
	if len(config.URLs) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	for i, url := range config.URLs {
		BulletedFprintf(indent+1, out, "%s: ", url)
		if i == 0 {
			_, _ = fmt.Fprint(out, "fetch, ")
		}
		_, _ = fmt.Fprintln(out, "push")
	}

	IndentedFprint(indent, out, "Up-to-date references:")
	refs, err := remote.List(git.EmptyListOptions())
	if err != nil {
		_, _ = fmt.Fprintf(out, " (couldn't retrieve references: %s)\n", err)
		return err
	}

	if len(refs) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	slices.SortFunc(refs, func(a, b *plumbing.Reference) int {
		return cmp.Compare(git.StringifyRef(a), git.StringifyRef(b))
	})
	for _, ref := range refs {
		if strings.HasPrefix(git.StringifyRef(ref), "pull/") {
			continue
		}
		BulletedFprintf(indent+1, out, "%s\n", git.StringifyRef(ref))
	}

	return nil
}
