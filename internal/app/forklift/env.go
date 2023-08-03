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
	if e.Env.Def, err = loadEnvDef(e.FS, EnvDefFile); err != nil {
		return nil, errors.Errorf("couldn't load env config")
	}
	return e, nil
}

// LoadFSEnvContaining loads the FSEnv containing the specified sub-directory path in the provided
// base filesystem.
// The provided path should use the host OS's path separators.
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

// FSEnv: Requirements

// getReqsFS returns the [fs.FS] in the environment which contains requirement definitions.
func (e *FSEnv) getReqsFS() (pallets.PathedFS, error) {
	return e.FS.Sub(ReqsDirName)
}

// FSEnv: Pallet Requirements

// GetReqsPalletsFS returns the [fs.FS] in the environment which contains pallet requirement
// definitions.
func (e *FSEnv) GetReqsPalletsFS() (pallets.PathedFS, error) {
	fsys, err := e.getReqsFS()
	if err != nil {
		return nil, err
	}
	return fsys.Sub(ReqsPalletsDirName)
}

// LoadFSPalletReq loads the FSPalletReq from the environment for the pallet with the specified
// path.
func (e *FSEnv) LoadFSPalletReq(palletPath string) (r *FSPalletReq, err error) {
	palletsFS, err := e.GetReqsPalletsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for pallet requirements from environment")
	}
	if r, err = loadFSPalletReq(palletsFS, palletPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't load pallet %s", palletPath)
	}
	return r, nil
}

// LoadFSPalletReqs loads all FSPalletReqs from the environment matching the specified search
// pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching the pallet paths to
// search for.
func (e *FSEnv) LoadFSPalletReqs(searchPattern string) ([]*FSPalletReq, error) {
	palletsFS, err := e.GetReqsPalletsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for pallets in local environment")
	}
	return loadFSPalletReqs(palletsFS, searchPattern)
}

// FSEnv: Package Requirements

// LoadPkgReq loads the PkgReq from the environment for the package with the specified package path.
func (e *FSEnv) LoadPkgReq(pkgPath string) (r PkgReq, err error) {
	palletsFS, err := e.GetReqsPalletsFS()
	if err != nil {
		return PkgReq{}, errors.Wrap(
			err, "couldn't open directory for pallet requirements from environment",
		)
	}
	fsPalletReq, err := loadFSPalletReqContaining(palletsFS, pkgPath)
	if err != nil {
		return PkgReq{}, errors.Wrapf(
			err, "couldn't find pallet providing package %s in local environment", pkgPath,
		)
	}
	r.Pallet = fsPalletReq.PalletReq
	r.PkgSubdir = fsPalletReq.GetPkgSubdir(pkgPath)
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

// LoadDepls loads all package deployment configurations matching the specified search pattern.
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

// EnvDef

// loadEnvDef loads an EnvDef from the specified file path in the provided base filesystem.
func loadEnvDef(fsys pallets.PathedFS, filePath string) (EnvDef, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return EnvDef{}, errors.Wrapf(
			err, "couldn't read environment config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := EnvDef{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return EnvDef{}, errors.Wrap(err, "couldn't parse environment config")
	}
	return config, nil
}
