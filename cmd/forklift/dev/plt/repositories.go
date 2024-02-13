package plt

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-repo

func cacheRepoAction(toolVersion, repoMinVersion, palletMinVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c, false, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, cache, toolVersion, repoMinVersion, palletMinVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		fmt.Printf("Downloading repos specified by the development pallet...\n")
		changed, err := fcli.DownloadRequiredRepos(0, pallet, cache.Underlay)
		if err != nil {
			return err
		}
		if !changed {
			fmt.Println("Done! No further actions are needed at this time.")
			return nil
		}

		// TODO: warn if any downloaded repo doesn't appear to be an actual repo, or if any repo's
		// forklift version is incompatible or ahead of the pallet version
		fmt.Println("Done! Next, you might want to run `sudo -E forklift dev plt apply`.")
		return nil
	}
}

// ls-repo

func lsRepoAction(c *cli.Context) error {
	pallet, err := getPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintPalletRepos(0, pallet)
}

// show-repo

func showRepoAction(c *cli.Context) error {
	pallet, cache, err := processFullBaseArgs(c, true, true)
	if err != nil {
		return err
	}

	repoPath := c.Args().First()
	return fcli.PrintRepoInfo(0, pallet, cache, repoPath)
}

// add-repo

func addRepoAction(toolVersion, repoMinVersion, palletMinVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c, false, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, cache, toolVersion, repoMinVersion, palletMinVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		repoQueries := c.Args().Slice()
		if err = fcli.ValidateGitRepoQueries(repoQueries); err != nil {
			return errors.Wrap(err, "one or more arguments is invalid")
		}
		resolved, err := fcli.ResolveQueriesUsingLocalMirrors(0, cache.Underlay.Path(), repoQueries)
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Printf("Saving configurations to %s...\n", pallet.FS.Path())
		for _, repoQuery := range repoQueries {
			req, ok := resolved[repoQuery]
			if !ok {
				return errors.Errorf("couldn't find configuration for %s", repoQuery)
			}
			reqsReposFS, err := pallet.GetRepoReqsFS()
			if err != nil {
				return err
			}
			repoReqPath := path.Join(reqsReposFS.Path(), req.Path(), forklift.VersionLockDefFile)
			marshaled, err := yaml.Marshal(req.VersionLock.Def)
			if err != nil {
				return errors.Wrapf(err, "couldn't marshal repo requirement from %s", repoReqPath)
			}
			if err := forklift.EnsureExists(filepath.FromSlash(path.Dir(repoReqPath))); err != nil {
				return errors.Wrapf(
					err, "couldn't make directory %s", filepath.FromSlash(path.Dir(repoReqPath)),
				)
			}
			const perm = 0o644 // owner rw, group r, public r
			if err := os.WriteFile(filepath.FromSlash(repoReqPath), marshaled, perm); err != nil {
				return errors.Wrapf(
					err, "couldn't save repo requirement to %s", filepath.FromSlash(repoReqPath),
				)
			}
		}
		fmt.Println("Done!")
		return nil
	}
}
