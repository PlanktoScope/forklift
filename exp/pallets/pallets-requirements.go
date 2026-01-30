// Package pallets implements the Forklift pallets specification for deployment and composition of
// Forklift packages.
package pallets

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/exp/fs"
	"github.com/forklift-run/forklift/exp/versioning"
)

// FSPallet: Pallet Requirements

// GetPalletReqsFS returns the [fs.FS] in the pallet which contains pallet requirement
// definitions.
func (p *FSPallet) GetPalletReqsFS() (ffs.PathedFS, error) {
	return p.FS.Sub(path.Join(ReqsDirName, ReqsPalletsDirName))
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
	palletReqsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for pallets in pallet")
	}
	return loadFSPalletReqs(palletReqsFS, searchPattern)
}

// loadPalletReq loads the PalletReq from the pallet with the specified pallet path.
func (p *FSPallet) loadPalletReq(palletPath string) (r PalletReq, err error) {
	fsPalletReq, err := p.LoadFSPalletReq(palletPath)
	if err != nil {
		return PalletReq{}, errors.Wrapf(err, "couldn't find pallet %s", palletPath)
	}
	return fsPalletReq.PalletReq, nil
}

// WriteFSPalletReq saves the provided req to the filesystem.
func (p *FSPallet) WriteFSPalletReq(req GitRepoReq) error {
	reqsPalletsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return err
	}
	palletReqPath := path.Join(reqsPalletsFS.Path(), req.Path(), versioning.LockDeclFile)
	if err = writeVersionLock(req.VersionLock, palletReqPath); err != nil {
		return errors.Wrapf(
			err, "couldn't write version lock for pallet requirement %s@%s",
			req.RequiredPath, req.VersionLock.Version,
		)
	}
	return nil
}

func writeVersionLock(lock versioning.Lock, writePath string) error {
	marshaled, err := yaml.Marshal(lock.Decl)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal version lock")
	}
	parentDir := filepath.FromSlash(path.Dir(writePath))
	if err := ffs.EnsureExists(parentDir); err != nil {
		return errors.Wrapf(err, "couldn't make directory %s", parentDir)
	}
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(filepath.FromSlash(writePath), marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save version lock to %s", filepath.FromSlash(writePath))
	}
	return nil
}

// RemoveFSPalletReq deletes the pallet requirement for the specified pallet from the filesystem, if
// it exists.
func (p *FSPallet) RemoveFSPalletReq(palletPath string) error {
	reqsPalletsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return err
	}
	palletReqPath := path.Join(reqsPalletsFS.Path(), palletPath)
	if err = os.RemoveAll(filepath.FromSlash(path.Join(
		palletReqPath, versioning.LockDeclFile,
	))); err != nil {
		return errors.Wrapf(
			err, "couldn't remove requirement for pallet %s, at %s", palletPath, palletReqPath,
		)
	}
	return nil
}

// FSPallet: Package Requirements

// LoadPkgReq loads the PkgReq from the pallet for the package with the specified package path.
func (p *FSPallet) LoadPkgReq(pkgPath string) (r PkgReq, err error) {
	if path.IsAbs(pkgPath) { // special case: package should be provided by the pallet itself
		return PkgReq{
			PkgSubdir: strings.TrimLeft(pkgPath, "/"),
			Pallet: PalletReq{
				GitRepoReq{RequiredPath: p.Decl.Pallet.Path},
			},
		}, nil
	}

	palletsFS, err := p.GetPalletReqsFS()
	if err != nil {
		return r, errors.Wrap(err, "couldn't open directory for pallet requirements from pallet")
	}
	fsPalletReq, err := LoadFSPalletReqContaining(palletsFS, pkgPath)
	if err != nil {
		return r, errors.Wrapf(err, "couldn't find pallet providing package %s in pallet", pkgPath)
	}
	r.Pallet = fsPalletReq.PalletReq
	r.PkgSubdir = fsPalletReq.GetPkgSubdir(pkgPath)
	return r, nil
}
