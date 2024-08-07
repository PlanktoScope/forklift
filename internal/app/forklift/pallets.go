package forklift

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/core"
)

// FSPallet

// LoadFSPallet loads a FSPallet from the specified directory path in the provided base filesystem.
func LoadFSPallet(fsys core.PathedFS, subdirPath string) (p *FSPallet, err error) {
	p = &FSPallet{}
	if p.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if p.Pallet.Def, err = loadPalletDef(p.FS, PalletDefFile); err != nil {
		return nil, errors.Errorf("couldn't load pallet config")
	}
	return p, nil
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

// LoadFSPallets loads all FSPallets from the provided base filesystem matching the specified search
// pattern. The search pattern should be a [doublestar] pattern, such as `**`, matching pallet
// directories to search for.
// In the embedded [Pallet] of each loaded FSPallet, the version is *not* initialized.
func LoadFSPallets(fsys core.PathedFS, searchPattern string) ([]*FSPallet, error) {
	searchPattern = path.Join(searchPattern, PalletDefFile)
	palletDefFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for pallet config files matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	orderedPallets := make([]*FSPallet, 0, len(palletDefFiles))
	pallets := make(map[string]*FSPallet)
	for _, palletDefFilePath := range palletDefFiles {
		if path.Base(palletDefFilePath) != PalletDefFile {
			continue
		}
		pallet, err := LoadFSPallet(fsys, path.Dir(palletDefFilePath))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load pallet from %s/%s", fsys.Path(), palletDefFilePath,
			)
		}

		orderedPallets = append(orderedPallets, pallet)
		pallets[pallet.Path()] = pallet
	}

	return orderedPallets, nil
}

// Exists checks whether the pallet actually exists on the OS's filesystem.
func (p *FSPallet) Exists() bool {
	return DirExists(p.FS.Path())
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (p *FSPallet) Remove() error {
	return os.RemoveAll(p.FS.Path())
}

// LoadReadme loads the readme file defined by the pallet.
func (p *FSPallet) LoadReadme() ([]byte, error) {
	readmePath := p.Def.Pallet.ReadmeFile
	bytes, err := fs.ReadFile(p.FS, readmePath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't read pallet readme %s/%s", p.FS.Path(), readmePath)
	}
	return bytes, nil
}

// Path returns either the pallet's path (if specified) or its path on the filesystem.
func (p *FSPallet) Path() string {
	if p.Def.Pallet.Path == "" {
		return p.FS.Path()
	}
	return p.Def.Pallet.Path
}

// FSPallet: Requirements

// getReqsFS returns the [fs.FS] in the pallet which contains requirement definitions.
func (p *FSPallet) getReqsFS() (core.PathedFS, error) {
	return p.FS.Sub(ReqsDirName)
}

// FSPallet: Pallet Requirements

// GetPalletReqsFS returns the [fs.FS] in the pallet which contains pallet requirement
// definitions.
func (p *FSPallet) GetPalletReqsFS() (core.PathedFS, error) {
	fsys, err := p.getReqsFS()
	if err != nil {
		return nil, err
	}
	return fsys.Sub(ReqsPalletsDirName)
}

// LoadFSPalletReq loads the FSPalletReq from the pallet for the pallet with the specified
// path.
func (p *FSPallet) LoadFSPalletReq(palletPath string) (r *FSPalletReq, err error) {
	palletsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for pallet requirements from pallet")
	}
	if r, err = loadFSPalletReq(palletsFS, palletPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't load pallet %s", palletPath)
	}
	return r, nil
}

// LoadFSPalletReqs loads all FSPalletReqs from the pallet matching the specified search
// pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching the pallet paths to
// search for.
func (p *FSPallet) LoadFSPalletReqs(searchPattern string) ([]*FSPalletReq, error) {
	palletsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for pallets in pallet")
	}
	return loadFSPalletReqs(palletsFS, searchPattern)
}

// FSPallet: Repo Requirements

// GetRepoReqsFS returns the [fs.FS] in the pallet which contains repo requirement
// definitions.
func (p *FSPallet) GetRepoReqsFS() (core.PathedFS, error) {
	fsys, err := p.getReqsFS()
	if err != nil {
		return nil, err
	}
	return fsys.Sub(ReqsReposDirName)
}

// LoadFSRepoReq loads the FSRepoReq from the pallet for the repo with the specified
// path.
func (p *FSPallet) LoadFSRepoReq(repoPath string) (r *FSRepoReq, err error) {
	reposFS, err := p.GetRepoReqsFS()
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
func (p *FSPallet) LoadFSRepoReqs(searchPattern string) ([]*FSRepoReq, error) {
	reposFS, err := p.GetRepoReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for repos in pallet")
	}
	return loadFSRepoReqs(reposFS, searchPattern)
}

// FSPallet: Package Requirements

// LoadPkgReq loads the PkgReq from the pallet for the package with the specified package path.
func (p *FSPallet) LoadPkgReq(pkgPath string) (r PkgReq, err error) {
	if path.IsAbs(pkgPath) { // special case: package should be provided by the pallet itself
		return PkgReq{
			PkgSubdir: strings.TrimLeft(pkgPath, "/"),
			Repo: RepoReq{
				GitRepoReq{RequiredPath: p.Def.Pallet.Path},
			},
		}, nil
	}

	reposFS, err := p.GetRepoReqsFS()
	if err != nil {
		return PkgReq{}, errors.Wrap(err, "couldn't open directory for repo requirements from pallet")
	}
	fsRepoReq, err := LoadFSRepoReqContaining(reposFS, pkgPath)
	if err != nil {
		return PkgReq{}, errors.Wrapf(err, "couldn't find repo providing package %s in pallet", pkgPath)
	}
	r.Repo = fsRepoReq.RepoReq
	r.PkgSubdir = fsRepoReq.GetPkgSubdir(pkgPath)
	return r, nil
}

// FSPallet: Deployments

// GetDeplsFS returns the [fs.FS] in the pallet which contains package deployment
// configurations.
func (p *FSPallet) GetDeplsFS() (core.PathedFS, error) {
	return p.FS.Sub(DeplsDirName)
}

// LoadDepl loads the Depl with the specified name from the pallet.
func (p *FSPallet) LoadDepl(name string) (depl Depl, err error) {
	deplsFS, err := p.GetDeplsFS()
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
func (p *FSPallet) LoadDepls(searchPattern string) ([]Depl, error) {
	fsys, err := p.GetDeplsFS()
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for package deployment configurations from pallet",
		)
	}
	return loadDepls(fsys, searchPattern)
}

// Pallet

// Path returns the repo path of the Pallet instance.
func (p Pallet) Path() string {
	return p.Def.Pallet.Path
}

// VersionQuery represents the Pallet instance as a version query.
func (p Pallet) VersionQuery() string {
	return fmt.Sprintf("%s@%s", p.Path(), p.Version)
}

// Check looks for errors in the construction of the repo.
func (p Pallet) Check() (errs []error) {
	errs = append(errs, core.ErrsWrap(p.Def.Check(), "invalid repo config")...)
	return errs
}

// ComparePallets returns an integer comparing two [Pallet] instances according to their paths and
// versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func ComparePallets(r, s Pallet) int {
	if result := core.ComparePaths(r.Path(), s.Path()); result != core.CompareEQ {
		return result
	}
	if result := semver.Compare(r.Version, s.Version); result != core.CompareEQ {
		return result
	}
	return core.CompareEQ
}

// PalletDef

// loadPalletDef loads a PalletDef from the specified file path in the provided base filesystem.
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

// Check looks for errors in the construction of the pallet configuration.
func (d PalletDef) Check() (errs []error) {
	return core.ErrsWrap(d.Pallet.Check(), "invalid pallet spec")
}

// PalletSpec

// Check looks for errors in the construction of the pallet spec.
func (s PalletSpec) Check() (errs []error) {
	if s.Path == "" {
		errs = append(errs, errors.Errorf("pallet spec is missing `path` parameter"))
	}
	return errs
}
