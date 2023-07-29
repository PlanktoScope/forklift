package forklift

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
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
	if e.Env.Config, err = LoadEnvConfig(e.FS, EnvSpecFile); err != nil {
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

// GetRepoReqsFS returns the [fs.FS] in the environment which contains repository requirement
// configurations.
func (e *FSEnv) GetRepoReqsFS() (pallets.PathedFS, error) {
	return e.FS.Sub(RepoReqsDirName)
}

// LoadFSRepoReq loads the FSRepoReq from the environment for the repository with the specified
// Pallet repository path.
func (e *FSEnv) LoadFSRepoReq(repoPath string) (r *FSRepoReq, err error) {
	reposFS, err := e.GetRepoReqsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for versioned Pallet repositories from environment",
		)
	}
	if r, err = loadFSRepoReq(reposFS, repoPath); err != nil {
		return nil, errors.Wrap(err, "couldn't load repo r")
	}
	return r, nil
}

// loadFSRepoReqContaining loads the FSRepoReq containing the specified sub-directory path in the
// environment.
// The sub-directory path does not have to actually exist; however, it would usually be provided as
// a Pallet package path.
func (e *FSEnv) loadFSRepoReqContaining(subdirPath string) (*FSRepoReq, error) {
	reposFS, err := e.GetRepoReqsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for versioned Pallet repositories from environment",
		)
	}
	return loadFSRepoReqContaining(reposFS, subdirPath)
}

// LoadFSRepoReqs loads all FSRepoReqs from the environment matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching the repo paths to
// search for.
func (e *FSEnv) LoadFSRepoReqs(searchPattern string) ([]*FSRepoReq, error) {
	reposFS, err := e.GetRepoReqsFS()
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
	fsRepoReq, err := e.loadFSRepoReqContaining(pkgPath)
	if err != nil {
		return PkgReq{}, errors.Wrapf(
			err, "couldn't find repo providing package %s in local environment", pkgPath,
		)
	}
	r.Repo = fsRepoReq.RepoReq
	r.PkgSubdir = fsRepoReq.GetPkgSubdir(pkgPath)
	return r, nil
}

// LoadRequiredFSPkg loads the specified package from the cache according to the versioning
// requirement for the package's repository as configured in the environment.
func LoadRequiredFSPkg(
	env *FSEnv, loader FSPkgLoader, pkgPath string,
) (*pallets.FSPkg, PkgReq, error) {
	req, err := env.LoadPkgReq(pkgPath)
	if err != nil {
		return nil, PkgReq{}, errors.Wrapf(
			err, "couldn't determine package requirement for package %s", pkgPath,
		)
	}
	fsPkg, err := LoadFSPkgFromPkgReq(loader, req)
	return fsPkg, req, err
}

// FSEnv: Deployments

// GetDeplsFS returns the [fs.FS] in the environment which contains package deployment
// configurations.
func (e *FSEnv) GetDeplsFS() (pallets.PathedFS, error) {
	return e.FS.Sub(DeplsDirName)
}

// LoadDepl loads the Depl with the specified name from the environment.
func (e *FSEnv) LoadDepl(name string) (depl Depl, err error) {
	deplsFS, err := e.GetDeplsFS()
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

func ResolveDepl(env *FSEnv, loader FSPkgLoader, depl Depl) (resolved *ResolvedDepl, err error) {
	resolved = &ResolvedDepl{
		Depl: depl,
	}
	pkgPath := resolved.Config.Package
	if resolved.Pkg, resolved.PkgReq, err = LoadRequiredFSPkg(env, loader, pkgPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load package %s to resolved from package deployment %s",
			pkgPath, depl.Name,
		)
	}
	return resolved, nil
}

// TODO: delegate some functionality to an env-independent LoadDepls function
// TODO: take a search pattern
func (e *FSEnv) LoadDepls() ([]Depl, error) {
	fsys, err := e.GetDeplsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for package deployment configurations from environment",
		)
	}
	files, err := doublestar.Glob(fsys, fmt.Sprintf("**/*%s", DeplSpecFileExt))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for Pallet package deployment configs")
	}
	deplNames := make([]string, 0, len(files))
	deplMap := make(map[string]Depl)
	for _, filePath := range files {
		deplName := strings.TrimSuffix(filePath, ".deploy.yml")
		if _, ok := deplMap[deplName]; ok {
			return nil, errors.Errorf(
				"package deployment %s repeatedly specified by the local environment", deplName,
			)
		}
		deplNames = append(deplNames, deplName)
		deplMap[deplName], err = e.LoadDepl(deplName)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load package deployment specification %s", deplName)
		}
	}

	orderedDepls := make([]Depl, 0, len(deplNames))
	for _, deplName := range deplNames {
		orderedDepls = append(orderedDepls, deplMap[deplName])
	}
	return orderedDepls, nil
}

// TODO: delegate some functionality to an env-independent LoadDepls function
// TODO: take a search pattern
// TODO: make a ResolveDepls function separate from a LoadDepls method
func (e *FSEnv) LoadResolvedDepls(loader FSPkgLoader) ([]*ResolvedDepl, error) {
	depls, err := e.LoadDepls()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't load pre-resolution package deployment configurations from environment",
		)
	}

	resolvedDepls := make([]*ResolvedDepl, 0, len(depls))
	for _, depl := range depls {
		resolved, err := ResolveDepl(e, loader, depl)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't resolve package deployment %s", depl.Name)
		}
		resolvedDepls = append(resolvedDepls, resolved)
	}

	return resolvedDepls, nil
}

// EnvConfig

// LoadEnvConfig loads an EnvConfig from the specified file path in the provided base filesystem.
func LoadEnvConfig(fsys pallets.PathedFS, filePath string) (EnvConfig, error) {
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
