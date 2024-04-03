package forklift

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"

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
	if b.Bundle.Def, err = loadBundleDef(b.FS, BundleDefFile); err != nil {
		return nil, errors.Errorf("couldn't load bundle definition")
	}
	for path, req := range b.Bundle.Def.Includes.Pallets {
		if req.Req.VersionLock.Version, err = req.Req.VersionLock.Def.Version(); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine requirement version of included pallet %s", path,
			)
		}
		b.Bundle.Def.Includes.Pallets[path] = req
	}
	for path, req := range b.Bundle.Def.Includes.Repos {
		if req.Req.VersionLock.Version, err = req.Req.VersionLock.Def.Version(); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine requirement version of included repo %s", path,
			)
		}
		b.Bundle.Def.Includes.Repos[path] = req
	}
	return b, nil
}

func (b *FSBundle) Path() string {
	return b.FS.Path()
}

// FSBundle: Deployments

func (b *FSBundle) AddDepl(depl *ResolvedDepl) error {
	b.Def.Deploys[depl.Name] = depl.Depl.Def
	// TODO: once we upgrade to go1.23, use os.CopyFS instead (see
	// https://github.com/golang/go/issues/62484)
	if err := cp.Copy(depl.Pkg.FS.Path(), filepath.FromSlash(
		path.Join(b.getPackagesPath(), depl.Def.Package),
	)); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files from package %s for deployment %s from %s",
			depl.Pkg.Path(), depl.Depl.Name, depl.Pkg.FS.Path(),
		)
	}
	return nil
}

func (b *FSBundle) LoadDepl(name string) (depl *ResolvedDepl, err error) {
	resolved := &ResolvedDepl{
		Depl: Depl{
			Name: name,
			Def:  b.Def.Deploys[name],
		},
	}
	pkgPath := b.Def.Deploys[name].Package
	/*if resolved.PkgReq, err = b.LoadPkgReq(pkgPath); err != nil {
	  return depl, err
	}*/
	if resolved.Pkg, err = b.LoadFSPkg(pkgPath, ""); err != nil {
		return depl, errors.Wrapf(err, "couldn't load package %s from bundle", pkgPath)
	}
	return resolved, nil
}

/*func (b *FSBundle) LoadPkgReq(pkgPath string) (r PkgReq, err error) {
  if path.IsAbs(pkgPath) {
    return PkgReq{
      PkgSubdir: strings.TrimLeft(pkgPath, "/"),
      RepoReq: RepoReq{
        GitRepoReq{RequiredPath: b.Def.Pallet.Path},
      }
    }, nil
  }

  return PkgReq{
    PkgSubdir: "",
    Repo: RepoReq{
      GitRepoReq: GitRepoReq{
        RequiredPath: "",
        VersionLock: VersionLock{},
      },
    },
  }, nil
}*/

// FSBundle: Packages

func (b *FSBundle) getPackagesPath() string {
	return path.Join(b.FS.Path(), packagesDirName)
}

// WriteRepoDefFile creates a repo definition file at the packages path, so that all loaded packages
// are associated with a repo.
func (b *FSBundle) WriteRepoDefFile() error {
	repoDef := core.RepoDef{
		ForkliftVersion: b.Def.ForkliftVersion,
	}
	marshaled, err := yaml.Marshal(repoDef)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal bundle definition")
	}
	outputPath := filepath.FromSlash(path.Join(b.getPackagesPath(), core.RepoDefFile))
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(outputPath, marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save bundle definition to %s", outputPath)
	}
	return nil
}

// FSBundle: Definition

func (b *FSBundle) WriteDefFile() error {
	marshaled, err := yaml.Marshal(b.Def)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal bundle definition")
	}
	outputPath := filepath.FromSlash(path.Join(b.FS.Path(), BundleDefFile))
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(outputPath, marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save bundle definition to %s", outputPath)
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
	return repo.LoadFSPkg(pkgPath)
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

// BundleDef

// loadBundleDef loads a BundleDef from the specified file path in the provided base filesystem.
func loadBundleDef(fsys core.PathedFS, filePath string) (BundleDef, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return BundleDef{}, errors.Wrapf(
			err, "couldn't read bundle config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := BundleDef{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return BundleDef{}, errors.Wrap(err, "couldn't parse bundle config")
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
