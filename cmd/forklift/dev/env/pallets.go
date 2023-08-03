package env

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
	"github.com/PlanktoScope/forklift/internal/clients/git"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// cache-pallet

func cachePalletAction(c *cli.Context) error {
	env, cache, _, err := processFullBaseArgs(c, false)
	if err != nil {
		return err
	}

	fmt.Printf("Downloading pallets specified by the development environment...\n")
	changed, err := fcli.DownloadPallets(0, env, cache.Underlay)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Done! No further actions are needed at this time.")
		return nil
	}

	fmt.Println("Done! Next, you might want to run `sudo -E forklift dev env apply`.")
	return nil
}

// ls-pallet

func lsPalletAction(c *cli.Context) error {
	env, err := getEnv(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintEnvPallets(0, env)
}

// show-pallet

func showPalletAction(c *cli.Context) error {
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	palletPath := c.Args().First()
	return fcli.PrintPalletInfo(0, env, cache, palletPath)
}

// add-pallet

func addPalletAction(c *cli.Context) error {
	env, cache, _, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	remoteReleases := c.Args().Slice()
	if len(remoteReleases) == 0 {
		return errors.Errorf("at least one pallet must be specified")
	}

	if err = validateRemoteReleases(remoteReleases); err != nil {
		return errors.Wrap(err, "one or more arguments is invalid")
	}
	fmt.Println("Updating local mirrors of remote Git repos...")
	if err = updateLocalRepoMirrors(remoteReleases, cache.Underlay.Path()); err != nil {
		return errors.Wrap(err, "couldn't update local repo mirrors")
	}

	fmt.Println()
	fmt.Println("Resolving version queries...")
	palletDefs, err := determinePalletDefs(remoteReleases, cache.Underlay.Path())
	if err != nil {
		return errors.Wrap(err, "couldn't resolve version queries for pallets")
	}
	fmt.Println()
	fmt.Printf("Saving configurations to %s...\n", env.FS.Path())
	for _, remoteRelease := range remoteReleases {
		config, ok := palletDefs[remoteRelease]
		if !ok {
			return errors.Errorf("couldn't find configuration for %s", remoteRelease)
		}
		reqsPalletsFS, err := env.GetReqsPalletsFS()
		if err != nil {
			return err
		}
		palletReqPath := path.Join(
			reqsPalletsFS.Path(), config.VCSRepoPath, config.PalletSubdir, forklift.VersionLockDefFile,
		)
		marshaled, err := yaml.Marshal(config.VersionLock.Def)
		if err != nil {
			return errors.Wrapf(err, "couldn't marshal config for %s", palletReqPath)
		}
		if err := forklift.EnsureExists(filepath.FromSlash(path.Dir(palletReqPath))); err != nil {
			return errors.Wrapf(
				err, "couldn't make directory %s", filepath.FromSlash(path.Dir(palletReqPath)),
			)
		}
		const perm = 0o644 // owner rw, group r, public r
		if err := os.WriteFile(filepath.FromSlash(palletReqPath), marshaled, perm); err != nil {
			return errors.Wrapf(err, "couldn't save config to %s", filepath.FromSlash(palletReqPath))
		}
	}
	fmt.Println("Done!")
	return nil
}

func validateRemoteReleases(remoteReleases []string) error {
	for _, remoteRelease := range remoteReleases {
		_, _, _, err := splitRemoteRepoRelease(remoteRelease)
		if err != nil {
			return errors.Wrapf(err, "'%s' is not a valid argument", remoteRelease)
		}
	}
	return nil
}

func splitRemoteRepoRelease(
	remoteRepoRelease string,
) (vcsRepoPath, palletSubdir, versionQuery string, err error) {
	remote, versionQuery, err := git.ParseRemoteRelease(remoteRepoRelease)
	if err != nil {
		return "", "", "", err
	}
	vcsRepoPath, palletSubdir, err = pallets.SplitRepoPathSubdir(remote)
	if err != nil {
		return "", "", "", err
	}
	return vcsRepoPath, palletSubdir, versionQuery, nil
}

func updateLocalRepoMirrors(remoteReleases []string, cachePath string) error {
	updatedRepos := make(map[string]struct{})
	for _, remoteRelease := range remoteReleases {
		vcsRepoPath, _, _, err := splitRemoteRepoRelease(remoteRelease)
		if err != nil {
			return err
		}
		if _, updated := updatedRepos[vcsRepoPath]; updated {
			continue
		}

		if err = updateLocalRepoMirror(
			vcsRepoPath, path.Join(cachePath, vcsRepoPath),
		); err != nil {
			return errors.Wrapf(err, "couldn't update local mirror of %s", vcsRepoPath)
		}
		updatedRepos[vcsRepoPath] = struct{}{}
	}
	return nil
}

func updateLocalRepoMirror(remote, cachedPath string) error {
	remote = filepath.FromSlash(remote)
	cachedPath = filepath.FromSlash(cachedPath)
	if _, err := os.Stat(cachedPath); err == nil {
		fmt.Printf("Fetching updates for %s...\n", cachedPath)
		if _, err = git.Fetch(cachedPath); err == nil {
			return err
		}
		fmt.Printf(
			"Warning: couldn't fetch updates in local mirror, will try to re-clone instead: %e\n", err,
		)
		if err = os.RemoveAll(cachedPath); err != nil {
			return errors.Wrapf(
				err, "couldn't remove %s in order to re-clone %s", cachedPath, remote,
			)
		}
	}

	fmt.Printf("Cloning %s to %s...\n", remote, cachedPath)
	_, err := git.CloneMirrored(remote, cachedPath)
	return err
}

func determinePalletDefs(
	remoteReleases []string, cachePath string,
) (map[string]forklift.PalletReq, error) {
	vcsRepoDefs := make(map[string]forklift.PalletReq)
	palletDefs := make(map[string]forklift.PalletReq)
	for _, remoteRelease := range remoteReleases {
		vcsRepoPath, palletSubdir, versionQuery, err := splitRemoteRepoRelease(remoteRelease)
		if err != nil {
			return nil, err
		}
		vcsRepoRelease := fmt.Sprintf("%s@%s", vcsRepoPath, versionQuery)
		if _, configured := vcsRepoDefs[vcsRepoRelease]; !configured {
			if vcsRepoDefs[vcsRepoRelease], err = resolveVCSRepoVersionQuery(
				cachePath, vcsRepoPath, versionQuery,
			); err != nil {
				return nil, errors.Wrapf(
					err, "couldn't resolve version query %s for pallet %s/%s",
					versionQuery, vcsRepoPath, palletSubdir,
				)
			}
		}

		config := vcsRepoDefs[vcsRepoRelease]
		config.PalletSubdir = palletSubdir
		fmt.Printf("Resolved %s as %+v", remoteRelease, config.VersionLock.Version)
		if config.VersionLock.Def.BaseVersion != "" {
			fmt.Printf(", version %s", config.VersionLock.Def.BaseVersion)
		}
		fmt.Println()
		palletDefs[remoteRelease] = config
	}
	return palletDefs, nil
}

func resolveVCSRepoVersionQuery(
	cachePath, vcsRepoPath, versionQuery string,
) (forklift.PalletReq, error) {
	req := forklift.PalletReq{
		VCSRepoPath: vcsRepoPath,
	}
	if versionQuery == "" {
		return forklift.PalletReq{}, errors.New(
			"support for empty version queries is not yet implemented!",
		)
	}
	localPath := filepath.FromSlash(path.Join(cachePath, vcsRepoPath))
	gitRepo, err := git.Open(localPath)
	if err != nil {
		return forklift.PalletReq{}, errors.Wrapf(
			err, "couldn't open local mirror of %s", vcsRepoPath,
		)
	}
	commit, err := queryRefs(gitRepo, versionQuery)
	if err != nil {
		return forklift.PalletReq{}, err
	}
	if commit == "" {
		commit, err = gitRepo.GetCommitFullHash(versionQuery)
		if err != nil {
			commit = ""
		}
	}
	if commit == "" {
		return forklift.PalletReq{}, errors.Errorf(
			"couldn't find matching commit for '%s' in %s", versionQuery, localPath,
		)
	}
	if req.VersionLock.Def, err = lockCommit(gitRepo, commit); err != nil {
		return forklift.PalletReq{}, err
	}
	if req.VersionLock.Version, err = req.VersionLock.Def.Version(); err != nil {
		return forklift.PalletReq{}, err
	}
	return req, nil
}

func queryRefs(gitRepo *git.Repo, versionQuery string) (commit string, err error) {
	refs, err := gitRepo.Refs()
	if err != nil {
		return "", err
	}
	for _, ref := range refs {
		if ref.Name().Short() != versionQuery {
			continue
		}

		if ref.Type() != git.HashReference {
			return "", errors.New("only hash references are supported")
		}
		return ref.Hash().String(), nil
	}
	return "", nil
}

func lockCommit(gitRepo *git.Repo, commit string) (config forklift.VersionLockDef, err error) {
	config.Commit = commit
	if config.Timestamp, err = forklift.GetCommitTimestamp(gitRepo, config.Commit); err != nil {
		return forklift.VersionLockDef{}, err
	}
	// FIXME: look for a version tagged on the commit, or the last version if it's a pseudoversion.
	// If there's a proper tagged version, write the tag as the base version and write the commit hash
	// but omit the timestamp.
	return config, nil
}
