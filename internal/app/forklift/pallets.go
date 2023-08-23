package forklift

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/core"
)

// FSPallet

// LoadFSPallet loads a FSPallet from the specified directory path in the provided base filesystem.
func LoadFSPallet(fsys core.PathedFS, subdirPath string) (e *FSPallet, err error) {
	e = &FSPallet{}
	if e.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if e.Pallet.Def, err = loadPalletDef(e.FS, PalletDefFile); err != nil {
		return nil, errors.Errorf("couldn't load pallet config")
	}
	return e, nil
}

// LoadFSPalletContaining loads the FSPallet containing the specified sub-directory path in the
// provided base filesystem.
// The provided path should use the host OS's path separators.
// The sub-directory path does not have to actually exist.
func LoadFSPalletContaining(path string) (*FSPallet, error) {
	palletCandidatePath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
	}
	for {
		if fsPallet, err := LoadFSPallet(
			core.AttachPath(os.DirFS(palletCandidatePath), palletCandidatePath), ".",
		); err == nil {
			return fsPallet, nil
		}

		palletCandidatePath = filepath.Dir(palletCandidatePath)
		if palletCandidatePath == "/" || palletCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf("no pallet config file found in any parent directory of %s", path)
		}
	}
}

// Exists checks whether the pallet actually exists on the OS's filesystem.
func (e *FSPallet) Exists() bool {
	return Exists(e.FS.Path())
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (e *FSPallet) Remove() error {
	return os.RemoveAll(e.FS.Path())
}

// FSPallet: Requirements

// getReqsFS returns the [fs.FS] in the pallet which contains requirement definitions.
func (e *FSPallet) getReqsFS() (core.PathedFS, error) {
	return e.FS.Sub(ReqsDirName)
}

// FSPallet: Repo Requirements

// GetRepoReqsFS returns the [fs.FS] in the pallet which contains repo requirement
// definitions.
func (e *FSPallet) GetRepoReqsFS() (core.PathedFS, error) {
	fsys, err := e.getReqsFS()
	if err != nil {
		return nil, err
	}
	return fsys.Sub(ReqsReposDirName)
}

// LoadFSRepoReq loads the FSRepoReq from the pallet for the repo with the specified
// path.
func (e *FSPallet) LoadFSRepoReq(repoPath string) (r *FSRepoReq, err error) {
	reposFS, err := e.GetRepoReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for repo requirements from pallet")
	}
	if r, err = loadFSRepoReq(reposFS, repoPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't load repo %s", repoPath)
	}
	return r, nil
}

// LoadFSRepoReqs loads all FSRepoReqs from the pallet matching the specified search
// pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching the repo paths to
// search for.
func (e *FSPallet) LoadFSRepoReqs(searchPattern string) ([]*FSRepoReq, error) {
	reposFS, err := e.GetRepoReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for repos in pallet")
	}
	return loadFSRepoReqs(reposFS, searchPattern)
}

// FSPallet: Package Requirements

// LoadPkgReq loads the PkgReq from the pallet for the package with the specified package path.
func (e *FSPallet) LoadPkgReq(pkgPath string) (r PkgReq, err error) {
	reposFS, err := e.GetRepoReqsFS()
	if err != nil {
		return PkgReq{}, errors.Wrap(err, "couldn't open directory for repo requirements from pallet")
	}
	fsRepoReq, err := loadFSRepoReqContaining(reposFS, pkgPath)
	if err != nil {
		return PkgReq{}, errors.Wrapf(err, "couldn't find repo providing package %s in pallet", pkgPath)
	}
	r.Repo = fsRepoReq.RepoReq
	r.PkgSubdir = fsRepoReq.GetPkgSubdir(pkgPath)
	return r, nil
}

// FSPallet: Deployments

// getDeplsFS returns the [fs.FS] in the pallet which contains package deployment
// configurations.
func (e *FSPallet) getDeplsFS() (core.PathedFS, error) {
	return e.FS.Sub(DeplsDirName)
}

// LoadDepl loads the Depl with the specified name from the pallet.
func (e *FSPallet) LoadDepl(name string) (depl Depl, err error) {
	deplsFS, err := e.getDeplsFS()
	if err != nil {
		return Depl{}, errors.Wrap(
			err, "couldn't open directory for package deployment configurations from pallet",
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
func (e *FSPallet) LoadDepls(searchPattern string) ([]Depl, error) {
	fsys, err := e.getDeplsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for package deployment configurations from pallet",
		)
	}
	return loadDepls(fsys, searchPattern)
}

// PalletDef

// loadPalletDef loads an PalletDef from the specified file path in the provided base filesystem.
func loadPalletDef(fsys core.PathedFS, filePath string) (PalletDef, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return PalletDef{}, errors.Wrapf(
			err, "couldn't read pallet config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := PalletDef{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return PalletDef{}, errors.Wrap(err, "couldn't parse pallet config")
	}
	return config, nil
}
