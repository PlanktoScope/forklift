package forklift

import (
	"os"
	"path"
	"path/filepath"

	cp "github.com/otiai10/copy"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/core"
)

func NewFSBundle(path string) *FSBundle {
	return &FSBundle{
		FS: core.AttachPath(os.DirFS(path), path),
	}
}

// FSBundle: Deployments

func (b *FSBundle) AddDepl(depl *ResolvedDepl) error {
	b.Def.Deploys[depl.Name] = depl.Depl.Def
	// TODO: once we upgrade to go1.23, use os.CopyFS instead (see
	// https://github.com/golang/go/issues/62484)
	if err := cp.Copy(depl.Pkg.FS.Path(), filepath.FromSlash(
		path.Join(b.getDeploymentsPath(), depl.Depl.Name),
	)); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files for deployment %s from %s", depl.Depl.Name, depl.Pkg.FS.Path(),
		)
	}
	return nil
}

func (b *FSBundle) getDeploymentsPath() string {
	return path.Join(b.FS.Path(), deploymentsDirName)
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
