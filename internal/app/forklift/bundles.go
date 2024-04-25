package forklift

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	cp "github.com/otiai10/copy"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/core"
)

// FSBundle

func NewFSBundle(path string) *FSBundle {
	return &FSBundle{
		FS: core.AttachPath(os.DirFS(path), path),
	}
}

// LoadFSBundle loads a FSBundle from a specified directory path in the provided base filesystem.
func LoadFSBundle(fsys core.PathedFS, subdirPath string) (b *FSBundle, err error) {
	b = &FSBundle{}
	if b.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if b.Bundle.Manifest, err = loadBundleManifest(b.FS, BundleManifestFile); err != nil {
		return nil, errors.Errorf("couldn't load bundle manifest")
	}
	for path, req := range b.Bundle.Manifest.Includes.Pallets {
		if req.Req.VersionLock.Version, err = req.Req.VersionLock.Def.Version(); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine requirement version of included pallet %s", path,
			)
		}
		b.Bundle.Manifest.Includes.Pallets[path] = req
	}
	for path, req := range b.Bundle.Manifest.Includes.Repos {
		if req.Req.VersionLock.Version, err = req.Req.VersionLock.Def.Version(); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine requirement version of included repo %s", path,
			)
		}
		b.Bundle.Manifest.Includes.Repos[path] = req
	}
	return b, nil
}

func (b *FSBundle) WriteManifestFile() error {
	marshaled, err := yaml.Marshal(b.Manifest)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal bundle manifest")
	}
	outputPath := filepath.FromSlash(path.Join(b.FS.Path(), BundleManifestFile))
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(outputPath, marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save bundle manifest to %s", outputPath)
	}
	return nil
}

func (b *FSBundle) Path() string {
	return b.FS.Path()
}

// FSBundle: Pallets

func (b *FSBundle) SetBundledPallet(pallet *FSPallet) error {
	// TODO: once we upgrade to go1.23, use os.CopyFS instead (see
	// https://github.com/golang/go/issues/62484)
	if err := cp.Copy(
		filepath.FromSlash(pallet.FS.Path()), filepath.FromSlash(b.getBundledPalletPath()),
	); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files for pallet %s from %s", pallet.Path(), pallet.FS.Path(),
		)
	}
	return nil
}

func (b *FSBundle) getBundledPalletPath() string {
	return path.Join(b.FS.Path(), bundledPalletDirName)
}

// FSBundle: Deployments

func (b *FSBundle) AddResolvedDepl(depl *ResolvedDepl) (err error) {
	b.Manifest.Deploys[depl.Name] = depl.Depl.Def
	if b.Manifest.Exports[depl.Name], err = depl.GetFileExportTargets(); err != nil {
		return errors.Wrapf(err, "couldn't determine file exports of deployment %s", depl.Depl.Name)
	}
	// TODO: once we upgrade to go1.23, use os.CopyFS instead (see
	// https://github.com/golang/go/issues/62484)
	if err = cp.Copy(filepath.FromSlash(depl.Pkg.FS.Path()), filepath.FromSlash(
		path.Join(b.getPackagesPath(), depl.Def.Package),
	)); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files from package %s for deployment %s from %s",
			depl.Pkg.Path(), depl.Depl.Name, depl.Pkg.FS.Path(),
		)
	}
	return nil
}

func (b *FSBundle) LoadDepl(name string) (Depl, error) {
	depl, ok := b.Manifest.Deploys[name]
	if !ok {
		return Depl{}, errors.Errorf("bundle does not contain package deployment %s", name)
	}
	return Depl{
		Name: name,
		Def:  depl,
	}, nil
}

func (b *FSBundle) LoadDepls(searchPattern string) ([]Depl, error) {
	deplNames := make([]string, 0, len(b.Manifest.Deploys))
	for deplName := range b.Manifest.Deploys {
		match, err := doublestar.Match(searchPattern, deplName)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't search for package deployment configs matching %s", searchPattern,
			)
		}
		if !match {
			continue
		}
		deplNames = append(deplNames, deplName)
	}
	slices.Sort(deplNames)
	depls := make([]Depl, 0, len(deplNames))
	for _, deplName := range deplNames {
		depl, err := b.LoadDepl(deplName)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load package deployment %s from bundle", deplName)
		}
		depls = append(depls, depl)
	}
	return depls, nil
}

func (b *FSBundle) LoadResolvedDepl(name string) (depl *ResolvedDepl, err error) {
	resolved := &ResolvedDepl{
		Depl: Depl{
			Name: name,
			Def:  b.Manifest.Deploys[name],
		},
	}
	pkgPath := b.Manifest.Deploys[name].Package
	if resolved.PkgReq, err = b.LoadPkgReq(pkgPath); err != nil {
		return depl, err
	}
	if resolved.Pkg, err = b.LoadFSPkg(pkgPath, ""); err != nil {
		return depl, errors.Wrapf(err, "couldn't load package deployment %s from bundle", pkgPath)
	}
	return resolved, nil
}

func (b *FSBundle) LoadPkgReq(pkgPath string) (r PkgReq, err error) {
	return PkgReq{
		PkgSubdir: strings.TrimLeft(pkgPath, "/"),
	}, nil
}

// FSBundle: Packages

func (b *FSBundle) getPackagesPath() string {
	return path.Join(b.FS.Path(), packagesDirName)
}

// WriteRepoDefFile creates a repo definition file at the packages path, so that all loaded packages
// are associated with a repo.
func (b *FSBundle) WriteRepoDefFile() error {
	repoDef := core.RepoDef{
		ForkliftVersion: b.Manifest.ForkliftVersion,
	}
	marshaled, err := yaml.Marshal(repoDef)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal bundle manifest")
	}
	outputPath := filepath.FromSlash(path.Join(b.getPackagesPath(), core.RepoDefFile))
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(outputPath, marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save bundle manifest to %s", outputPath)
	}
	return nil
}

// FSBundle: Exports

func (b *FSBundle) getExportsPath() string {
	return path.Join(b.FS.Path(), exportsDirName)
}

func (b *FSBundle) WriteFileExports() error {
	if err := EnsureExists(filepath.FromSlash(b.getExportsPath())); err != nil {
		return errors.Wrapf(err, "couldn't make directory for all file exports")
	}
	for deplName := range b.Manifest.Deploys {
		resolved, err := b.LoadResolvedDepl(deplName)
		if err != nil {
			return errors.Wrapf(err, "couldn't resolve deployment %s", deplName)
		}
		exports, err := resolved.GetFileExports()
		if err != nil {
			return errors.Wrapf(err, "couldn't determine file exports for deployment %s", deplName)
		}
		for _, export := range exports {
			sourcePath := path.Join(resolved.Pkg.FS.Path(), export.Source)
			if export.Source == "" {
				sourcePath = path.Join(resolved.Pkg.FS.Path(), export.Target)
			}
			exportPath := path.Join(b.getExportsPath(), export.Target)
			if err := EnsureExists(filepath.FromSlash(path.Dir(exportPath))); err != nil {
				return errors.Wrapf(
					err, "couldn't make export directory %s in bundle", path.Dir(exportPath),
				)
			}
			// TODO: once we upgrade to go1.23, use os.CopyFS instead (see
			// https://github.com/golang/go/issues/62484)
			if err := cp.Copy(
				filepath.FromSlash(sourcePath), filepath.FromSlash(exportPath),
			); err != nil {
				return errors.Wrapf(err, "couldn't export file from %s to %s", sourcePath, exportPath)
			}
		}
	}
	return nil
}

// FSBundle: FSRepoLoader

func (b *FSBundle) LoadFSRepo(repoPath string, version string) (*core.FSRepo, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	return core.LoadFSRepo(b.FS, path.Join(packagesDirName, repoPath))
}

func (b *FSBundle) LoadFSRepos(searchPattern string) ([]*core.FSRepo, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	return core.LoadFSRepos(b.FS, path.Join(packagesDirName, searchPattern))
}

// FSBundle: FSPkgLoader

func (b *FSBundle) LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	repo, err := b.LoadFSRepo(".", "")
	if err != nil {
		return nil, err
	}
	return repo.LoadFSPkg(strings.TrimLeft(pkgPath, "/"))
}

func (b *FSBundle) LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	repo, err := b.LoadFSRepo(".", "")
	if err != nil {
		return nil, err
	}
	return repo.LoadFSPkgs(searchPattern)
}

// BundleManifest

// loadBundleManifest loads a BundleManifest from the specified file path in the provided base
// filesystem.
func loadBundleManifest(fsys core.PathedFS, filePath string) (BundleManifest, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return BundleManifest{}, errors.Wrapf(
			err, "couldn't read bundle config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := BundleManifest{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return BundleManifest{}, errors.Wrap(err, "couldn't parse bundle config")
	}
	return config, nil
}

// BundleInclusions

func (i *BundleInclusions) HasInclusions() bool {
	return len(i.Pallets)+len(i.Repos) > 0
}

func (i *BundleInclusions) HasOverrides() bool {
	for _, inclusion := range i.Pallets {
		if inclusion.Override != (BundleInclusionOverride{}) {
			return true
		}
	}
	for _, inclusion := range i.Repos {
		if inclusion.Override != (BundleInclusionOverride{}) {
			return true
		}
	}
	return false
}
