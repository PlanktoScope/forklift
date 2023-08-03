package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

func processFullBaseArgs(c *cli.Context, ensureCache bool) (
	env *forklift.FSEnv, cache *forklift.LayeredCache, override *forklift.PalletOverrideCache,
	err error,
) {
	if env, err = getEnv(c.String("cwd")); err != nil {
		return nil, nil, nil, err
	}
	if cache, override, err = getCache(
		c.String("workspace"), c.StringSlice("pallets"), ensureCache,
	); err != nil {
		return nil, nil, nil, err
	}
	return env, cache, override, nil
}

func getEnv(cwdPath string) (env *forklift.FSEnv, err error) {
	if env, err = forklift.LoadFSEnvContaining(cwdPath); err != nil {
		return nil, errors.Wrapf(
			err, "The current working directory %s is not part of a Forklift environment", cwdPath,
		)
	}
	return env, nil
}

func getCache(
	wpath string, pallets []string, ensureCache bool,
) (*forklift.LayeredCache, *forklift.PalletOverrideCache, error) {
	cache := &forklift.LayeredCache{}
	replacementPallets, err := loadReplacementPallets(pallets)
	if err != nil {
		return nil, nil, err
	}
	override, err := forklift.NewPalletOverrideCache(replacementPallets, nil)
	if err != nil {
		return nil, nil, err
	}
	cache.Overlay = override

	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, nil, err
	}
	fsCache, err := workspace.GetCache()
	if err != nil && len(pallets) == 0 {
		return nil, nil, err
	}
	cache.Underlay = fsCache

	if (ensureCache || len(pallets) == 0) && !fsCache.Exists() {
		return nil, nil, errors.New(
			"you first need to cache the pallets specified by your environment with " +
				"`forklift dev env cache-pallet`",
		)
	}
	return cache, override, nil
}

func loadReplacementPallets(fsPaths []string) (replacements []*pallets.FSPallet, err error) {
	for _, path := range fsPaths {
		replacementPath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
		}
		if !forklift.Exists(replacementPath) {
			return nil, errors.Errorf("couldn't find pallet replacement path %s", replacementPath)
		}
		externalPallets, err := pallets.LoadFSPallets(
			pallets.AttachPath(os.DirFS(replacementPath), replacementPath), "**", nil,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't list replacement pallets in path %s", replacementPath)
		}
		if len(externalPallets) == 0 {
			return nil, errors.Errorf("no replacement pallets found in path %s", replacementPath)
		}
		replacements = append(replacements, externalPallets...)
	}
	return replacements, nil
}

func setOverrideCacheVersions(
	env *forklift.FSEnv, overrideCache *forklift.PalletOverrideCache,
) error {
	reqs, err := env.LoadFSPalletReqs("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify pallet requirements specified by environment %s", env.FS.Path(),
		)
	}
	palletVersions := make(map[string]map[string]struct{})
	for _, req := range reqs {
		palletPath := req.Path()
		version := req.VersionLock.Version
		if _, ok := palletVersions[palletPath]; !ok {
			palletVersions[palletPath] = make(map[string]struct{})
		}
		palletVersions[palletPath][version] = struct{}{}
	}

	for palletPath, versions := range palletVersions {
		overrideCache.SetVersions(palletPath, versions)
	}
	return nil
}

// show

func showAction(c *cli.Context) error {
	env, err := getEnv(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintEnvInfo(0, env)
}

// check

func checkAction(c *cli.Context) error {
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	if err := fcli.CheckEnv(0, env, cache); err != nil {
		return err
	}
	return nil
}

// plan

func planAction(c *cli.Context) error {
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	if err := fcli.PlanEnv(0, env, cache); err != nil {
		return err
	}
	return nil
}

// apply

func applyAction(c *cli.Context) error {
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	if err := fcli.ApplyEnv(0, env, cache); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}
