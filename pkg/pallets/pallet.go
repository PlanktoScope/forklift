package pallets

import (
	"io/fs"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// Paths

// SplitRepoPathSubdir splits pallet paths of form github.com/user-name/git-repo-name/pallet-subdir
// into github.com/user-name/git-repo-name and pallet-subdir.
func SplitRepoPathSubdir(palletPath string) (vcsRepoPath, palletSubdir string, err error) {
	const sep = "/"
	pathParts := strings.Split(palletPath, sep)
	if pathParts[0] != "github.com" {
		return "", "", errors.Errorf(
			"pallet %s does not begin with github.com, but support for non-GitHub repositories is not "+
				"yet implemented",
			palletPath,
		)
	}
	const splitIndex = 3
	if len(pathParts) < splitIndex {
		return "", "", errors.Errorf(
			"pallet %s does not appear to be within a GitHub Git repository", palletPath,
		)
	}
	return strings.Join(pathParts[0:splitIndex], sep), strings.Join(pathParts[splitIndex:], sep), nil
}

// FSPallet

// LoadFSPallet loads a FSPallet from the specified directory path in the provided base filesystem.
// In the loaded FSPallet's embedded [Pallet], the VCS repository path and pallet subdirectory are
// initialized from the pallet path declared in the pallet's configuration file, while the version
// is *not* initialized.
func LoadFSPallet(fsys PathedFS, subdirPath string) (r *FSPallet, err error) {
	r = &FSPallet{}
	if r.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if r.Pallet.Def, err = LoadPalletDef(r.FS, PalletDefFile); err != nil {
		return nil, errors.Wrapf(err, "couldn't load pallet config")
	}
	if r.VCSRepoPath, r.Subdir, err = SplitRepoPathSubdir(r.Def.Pallet.Path); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't parse pallet path %s", r.Def.Pallet.Path,
		)
	}
	return r, nil
}

// LoadFSPalletContaining loads the FSPallet containing the specified sub-directory path in the
// provided base filesystem.
// The sub-directory path does not have to actually exist.
// In the loaded FSPallet's embedded [Pallet], the VCS repository path and pallet subdirectory are
// initialized from the pallet path declared in the pallet's configuration file, while the version
// is *not* initialized.
func LoadFSPalletContaining(fsys PathedFS, subdirPath string) (*FSPallet, error) {
	palletCandidatePath := subdirPath
	for {
		if pallet, err := LoadFSPallet(fsys, palletCandidatePath); err == nil {
			return pallet, nil
		}
		palletCandidatePath = path.Dir(palletCandidatePath)
		if palletCandidatePath == "/" || palletCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no pallet config file was found in any parent directory of %s", subdirPath,
			)
		}
	}
}

// LoadFSPallets loads all FSPallets from the provided base filesystem matching the specified search
// pattern, modifying each FSPallet with the the optional processor function if a non-nil function is
// provided. The search pattern should be a [doublestar] pattern, such as `**`, matching pallet
// directories to search for.
// With a nil processor function, in the embedded [Pallet] of each loaded FSPallet, the VCS
// repository path and pallet subdirectory are initialized from the pallet path declared in the
// pallet's configuration file, while the version is not initialized.
// After the processor is applied to each pallet, all pallets are checked to enforce that multiple
// copies of the same pallet with the same version are not allowed to be in the provided filesystem.
func LoadFSPallets(
	fsys PathedFS, searchPattern string, processor func(pallet *FSPallet) error,
) ([]*FSPallet, error) {
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
		if processor != nil {
			if err = processor(pallet); err != nil {
				return nil, errors.Wrap(err, "couldn't run processors on loaded pallet")
			}
		}

		palletPath := pallet.Def.Pallet.Path
		if prevPallet, ok := pallets[palletPath]; ok {
			if prevPallet.FromSameVCSRepo(pallet.Pallet) && prevPallet.Version == pallet.Version &&
				prevPallet.FS.Path() == pallet.FS.Path() {
				return nil, errors.Errorf(
					"the same version of pallet %s was found in multiple different locations: %s, %s",
					palletPath, prevPallet.FS.Path(), pallet.FS.Path(),
				)
			}
		}
		orderedPallets = append(orderedPallets, pallet)
		pallets[palletPath] = pallet
	}

	return orderedPallets, nil
}

// LoadFSPkg loads a package at the specified filesystem path from the FSPallet instance
// The loaded package is fully initialized.
func (r *FSPallet) LoadFSPkg(pkgSubdir string) (pkg *FSPkg, err error) {
	if pkg, err = LoadFSPkg(r.FS, pkgSubdir); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load package %s from pallet %s", pkgSubdir, r.Path(),
		)
	}
	if err = pkg.AttachFSPallet(r); err != nil {
		return nil, errors.Wrap(err, "couldn't attach pallet to package")
	}
	return pkg, nil
}

// LoadFSPkgs loads all packages in the FSPallet instance.
// The loaded packages are fully initialized.
func (r *FSPallet) LoadFSPkgs(searchPattern string) ([]*FSPkg, error) {
	pkgs, err := LoadFSPkgs(r.FS, searchPattern)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		if err = pkg.AttachFSPallet(r); err != nil {
			return nil, errors.Wrap(err, "couldn't attach pallet to package")
		}
	}
	return pkgs, nil
}

// LoadReadme loads the readme file defined by the pallet.
func (r *FSPallet) LoadReadme() ([]byte, error) {
	readmePath := r.Def.Pallet.ReadmeFile
	bytes, err := fs.ReadFile(r.FS, readmePath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't read pallet readme %s/%s", r.FS.Path(), readmePath)
	}
	return bytes, nil
}

// Pallet

// Path returns the pallet path of the Pallet instance.
func (r Pallet) Path() string {
	return path.Join(r.VCSRepoPath, r.Subdir)
}

// FromSameVCSRepo determines whether the candidate pallet is provided by the same VCS repo as the
// Pallet instance.
func (r Pallet) FromSameVCSRepo(candidate Pallet) bool {
	return r.VCSRepoPath == candidate.VCSRepoPath && r.Version == candidate.Version
}

// Check looks for errors in the construction of the pallet.
func (r Pallet) Check() (errs []error) {
	if r.Path() != r.Def.Pallet.Path {
		errs = append(errs, errors.Errorf(
			"pallet path %s is inconsistent with path %s specified in pallet configuration",
			r.Path(), r.Def.Pallet.Path,
		))
	}
	errs = append(errs, ErrsWrap(r.Def.Check(), "invalid pallet config")...)
	return errs
}

// The result of comparison functions is one of these values.
const (
	CompareLT = -1
	CompareEQ = 0
	CompareGT = 1
)

// ComparePallets returns an integer comparing two [Pallet] instances according to their paths and
// versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func ComparePallets(r, s Pallet) int {
	if result := ComparePaths(r.Path(), s.Path()); result != CompareEQ {
		return result
	}
	if result := semver.Compare(r.Version, s.Version); result != CompareEQ {
		return result
	}
	return CompareEQ
}

// ComparePaths returns an integer comparing two paths. The result will be 0 if the r and s are
// the same; -1 if r alphabetically comes before s; or +1 if r alphabetically comes after s.
func ComparePaths(r, s string) int {
	if r < s {
		return CompareLT
	}
	if r > s {
		return CompareGT
	}
	return CompareEQ
}

// PalletDef

// LoadPalletDef loads a PalletDef from the specified file path in the provided base filesystem.
func LoadPalletDef(fsys PathedFS, filePath string) (PalletDef, error) {
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
func (c PalletDef) Check() (errs []error) {
	return ErrsWrap(c.Pallet.Check(), "invalid pallet spec")
}

// PalletSpec

// Check looks for errors in the construction of the pallet spec.
func (s PalletSpec) Check() (errs []error) {
	if s.Path == "" {
		errs = append(errs, errors.Errorf("pallet spec is missing `path` parameter"))
	}
	return errs
}
