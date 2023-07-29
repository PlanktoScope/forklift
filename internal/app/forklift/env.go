package forklift

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// FSEnv

// LoadFSEnv loads a FSEnv from the specified directory path in the provided base filesystem.
func LoadFSEnv(fsys pallets.PathedFS, subdirPath string) (e *FSEnv, err error) {
	e = &FSEnv{}
	if e.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if e.Env.Config, err = loadEnvConfig(e.FS, EnvSpecFile); err != nil {
		return nil, errors.Errorf("couldn't load env config")
	}
	return e, nil
}

// LoadFSEnvContaining loads the FSEnv containing the specified sub-directory path in the provided
// base filesystem.
// The sub-directory path does not have to actually exist.
func LoadFSEnvContaining(path string) (*FSEnv, error) {
	envCandidatePath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
	}
	for {
		if fsEnv, err := LoadFSEnv(
			pallets.AttachPath(os.DirFS(envCandidatePath), envCandidatePath), ".",
		); err == nil {
			return fsEnv, nil
		}

		envCandidatePath = filepath.Dir(envCandidatePath)
		if envCandidatePath == "/" || envCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no environment config file found in any parent directory of %s", path,
			)
		}
	}
}

// Exists checks whether the environment actually exists on the OS's filesystem.
func (e *FSEnv) Exists() bool {
	return Exists(e.FS.Path())
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (e *FSEnv) Remove() error {
	return os.RemoveAll(e.FS.Path())
}

// FSEnv: Repo Requirements

// getRepoReqsFS returns the [fs.FS] in the environment which contains repository requirement
// configurations.
func (e *FSEnv) getRepoReqsFS() (pallets.PathedFS, error) {
	return e.FS.Sub(RepoReqsDirName)
}

// LoadFSRepoReq loads the FSRepoReq from the environment for the repository with the specified
// Pallet repository path.
func (e *FSEnv) LoadFSRepoReq(repoPath string) (r *FSRepoReq, err error) {
	reposFS, err := e.getRepoReqsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for Pallet repository requirements from environment",
		)
	}
	if r, err = loadFSRepoReq(reposFS, repoPath); err != nil {
		return nil, errors.Wrap(err, "couldn't load repo r")
	}
	return r, nil
}

// LoadFSRepoReqs loads all FSRepoReqs from the environment matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching the repo paths to
// search for.
func (e *FSEnv) LoadFSRepoReqs(searchPattern string) ([]*FSRepoReq, error) {
	reposFS, err := e.getRepoReqsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for Pallet repositories in local environment",
		)
	}
	return loadFSRepoReqs(reposFS, searchPattern)
}

// FSEnv: Package Requirements

// LoadPkgReq loads the PkgReq from the environment for the package with the specified Pallet
// package path.
func (e *FSEnv) LoadPkgReq(pkgPath string) (r PkgReq, err error) {
	reposFS, err := e.getRepoReqsFS()
	if err != nil {
		return PkgReq{}, errors.Wrap(
			err, "couldn't open directory for Pallet repository requirements from environment",
		)
	}
	fsRepoReq, err := loadFSRepoReqContaining(reposFS, pkgPath)
	if err != nil {
		return PkgReq{}, errors.Wrapf(
			err, "couldn't find repo providing package %s in local environment", pkgPath,
		)
	}
	r.Repo = fsRepoReq.RepoReq
	r.PkgSubdir = fsRepoReq.GetPkgSubdir(pkgPath)
	return r, nil
}

// FSEnv: Deployments

// getDeplsFS returns the [fs.FS] in the environment which contains package deployment
// configurations.
func (e *FSEnv) getDeplsFS() (pallets.PathedFS, error) {
	return e.FS.Sub(DeplsDirName)
}

// LoadDepl loads the Depl with the specified name from the environment.
func (e *FSEnv) LoadDepl(name string) (depl Depl, err error) {
	deplsFS, err := e.getDeplsFS()
	if err != nil {
		return Depl{}, errors.Wrap(
			err, "couldn't open directory for package deployment configurations from environment",
		)
	}
	if depl, err = loadDepl(deplsFS, name); err != nil {
		return Depl{}, errors.Wrapf(err, "couldn't load package deployment for %s", name)
	}
	return depl, nil
}

// LoadDepls loads all Pallet package deployment configurations matching the specified search
// pattern.
// The search pattern should not include the file extension for deployment specification files - the
// file extension will be appended to the search pattern by LoadDepls.
func (e *FSEnv) LoadDepls(searchPattern string) ([]Depl, error) {
	fsys, err := e.getDeplsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for package deployment configurations from environment",
		)
	}
	return loadDepls(fsys, searchPattern)
}

// EnvConfig

// loadEnvConfig loads an EnvConfig from the specified file path in the provided base filesystem.
func loadEnvConfig(fsys pallets.PathedFS, filePath string) (EnvConfig, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return EnvConfig{}, errors.Wrapf(
			err, "couldn't read environment config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := EnvConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return EnvConfig{}, errors.Wrap(err, "couldn't parse environment config")
	}
	return config, nil
}
