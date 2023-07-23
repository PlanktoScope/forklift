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

func FindParentEnv(cwd string) (path string, err error) {
	envCandidatePath, err := filepath.Abs(cwd)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't convert '%s' into an absolute path", cwd)
	}
	for envCandidatePath != "." && envCandidatePath != "/" {
		f := os.DirFS(envCandidatePath)
		_, err := fs.ReadFile(f, "forklift-env.yml")
		if err == nil {
			return envCandidatePath, nil
		}
		envCandidatePath = filepath.Dir(envCandidatePath)
	}
	return "", errors.Errorf(
		"no environment config file found in any parent directory of %s", cwd,
	)
}

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
	// return &FSEnv{
	// 	FS: pallets.AttachPath(os.DirFS(path), path),
	// }, nil
}

func (e *FSEnv) Exists() bool {
	return Exists(e.FS.Path())
}

func (e *FSEnv) Remove() error {
	return os.RemoveAll(e.FS.Path())
}

// FSEnv: Repo Requirements

func (e *FSEnv) GetRepoRequirementsFS() (pallets.PathedFS, error) {
	return e.FS.Sub(RepoRequirementsDirName)
}

func (e *FSEnv) LoadFSRepoRequirement(repoPath string) (r *FSRepoRequirement, err error) {
	reposFS, err := e.GetRepoRequirementsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for versioned Pallet repositories from environment",
		)
	}
	if r, err = loadFSRepoRequirement(reposFS, repoPath); err != nil {
		return nil, errors.Wrap(err, "couldn't load repo r")
	}
	return r, nil
}

// LoadFSRepoRequirements loads all FSRepoRequirements from the environment matching the specified
// search pattern. The search pattern should be a [doublestar] pattern, such as `**`, matching the
// repo paths to search for.
func (e *FSEnv) LoadFSRepoRequirements(searchPattern string) ([]*FSRepoRequirement, error) {
	reposFS, err := e.GetRepoRequirementsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for Pallet repositories in local environment",
		)
	}
	return loadFSRepoRequirements(reposFS, searchPattern)
}

// FSEnv: Versioned Packages

func (e *FSEnv) LoadVersionedPkg(cache *FSCache, pkgPath string) (p *VersionedPkg, err error) {
	p = &VersionedPkg{}
	if p.RepoRequirement, err = e.loadFSRepoRequirementOfPkg(pkgPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't find repo providing package %s in local environment", pkgPath,
		)
	}
	version := p.RepoRequirement.VersionLock.Version
	if p.FSPkg, err = cache.LoadFSPkg(pkgPath, version); err != nil {
		return nil, errors.Wrapf(err, "couldn't find package %s@%s in cache", pkgPath, version)
	}

	return p, nil
}

func (e *FSEnv) loadFSRepoRequirementOfPkg(pkgPath string) (*FSRepoRequirement, error) {
	repoCandidatePath := filepath.Dir(pkgPath)
	for repoCandidatePath != "." {
		repo, err := e.LoadFSRepoRequirement(repoCandidatePath)
		if err == nil {
			return repo, nil
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
	}
	return nil, errors.Errorf(
		"no repository config file found in %s or any parent directory in local environment",
		filepath.Dir(pkgPath),
	)
}

// FSEnv: Deployments

func (e *FSEnv) GetDeplsFS() (pallets.PathedFS, error) {
	return e.FS.Sub(DeplsDirName)
}

func (e *FSEnv) LoadDepl(
	cache *FSCache, replacementRepos map[string]*pallets.FSRepo, deplName string,
) (depl *Depl, err error) {
	depl = &Depl{
		Name: deplName,
	}

	deplsFS, err := e.GetDeplsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for package deployment configurations from environment",
		)
	}
	if depl.Config, err = loadDeplConfig(
		deplsFS, fmt.Sprintf("%s.deploy.yml", deplName),
	); err != nil {
		return nil, errors.Wrapf(err, "couldn't load package deployment config for %s", deplName)
	}

	pkgPath := depl.Config.Package
	repo, ok := FindExternalRepoOfPkg(replacementRepos, pkgPath)
	if ok {
		pkg, perr := repo.LoadFSPkg(repo.GetPkgSubdir(pkgPath))
		if perr != nil {
			return nil, errors.Wrapf(
				err, "couldn't find external package %s from replacement repo %s", pkgPath, repo.FS.Path(),
			)
		}
		depl.Pkg = AsVersionedPkg(pkg)
		return depl, nil
	}

	if depl.Pkg, err = e.LoadVersionedPkg(cache, pkgPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load versioned package %s to be deployed by local environment", pkgPath,
		)
	}

	return depl, nil
}

// TODO: rename this method
func (e *FSEnv) ListDepls(
	cache *FSCache, replacementRepos map[string]*pallets.FSRepo,
) ([]*Depl, error) {
	fsys, err := e.GetDeplsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for package deployment configurations from environment",
		)
	}
	files, err := doublestar.Glob(fsys, fmt.Sprintf("*%s", DeplsFileExt))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for Pallet package deployment configs")
	}

	deplNames := make([]string, 0, len(files))
	deplMap := make(map[string]*Depl)
	for _, filePath := range files {
		deplName := strings.TrimSuffix(filePath, ".deploy.yml")
		if _, ok := deplMap[deplName]; ok {
			return nil, errors.Errorf(
				"package deployment %s repeatedly specified by the local environment", deplName,
			)
		}
		deplNames = append(deplNames, deplName)
		deplMap[deplName], err = e.LoadDepl(cache, replacementRepos, deplName)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load package deployment specification %s", deplName)
		}
	}

	orderedDepls := make([]*Depl, 0, len(deplNames))
	for _, deplName := range deplNames {
		orderedDepls = append(orderedDepls, deplMap[deplName])
	}
	return orderedDepls, nil
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
