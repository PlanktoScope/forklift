package forklift

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/exp/fs"
	fplt "github.com/forklift-run/forklift/exp/pallets"
)

func CheckDepl(
	pallet *fplt.FSPallet, pkgLoader fplt.FSPkgLoader, depl fplt.Depl,
) error {
	pkg, _, err := fplt.LoadRequiredFSPkg(pallet, pkgLoader, depl.Decl.Package)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't resolve package path %s to a package using the pallet's pallet requirements",
			depl.Decl.Package,
		)
	}

	allowedFeatures := pkg.Decl.Features
	unrecognizedFeatures := make([]string, 0, len(depl.Decl.Features))
	for _, name := range depl.Decl.Features {
		if _, ok := allowedFeatures[name]; !ok {
			unrecognizedFeatures = append(unrecognizedFeatures, name)
		}
	}
	if len(unrecognizedFeatures) > 0 {
		return errors.Errorf("unrecognized feature flags: %+v", unrecognizedFeatures)
	}
	return nil
}

func WriteDepl(pallet *fplt.FSPallet, depl fplt.Depl) error {
	deplsFS, err := pallet.GetDeplsFS()
	if err != nil {
		return err
	}
	deplPath := path.Join(deplsFS.Path(), fmt.Sprintf("%s.deploy.yml", depl.Name))
	buf := bytes.Buffer{}
	encoder := yaml.NewEncoder(&buf)
	const yamlIndent = 2
	encoder.SetIndent(yamlIndent)
	if err = encoder.Encode(depl.Decl); err != nil {
		return errors.Wrapf(err, "couldn't marshal package deployment for %s", deplPath)
	}
	if err := ffs.EnsureExists(filepath.FromSlash(path.Dir(deplPath))); err != nil {
		return errors.Wrapf(
			err, "couldn't make directory %s", filepath.FromSlash(path.Dir(deplPath)),
		)
	}
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(filepath.FromSlash(deplPath), buf.Bytes(), perm); err != nil {
		return errors.Wrapf(
			err, "couldn't save deployment declaration to %s", filepath.FromSlash(deplPath),
		)
	}
	return nil
}
