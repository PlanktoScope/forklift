package bundling

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
)

// BundleManifestFile is the name of the file describing each Forklift pallet bundle.
const BundleManifestFile = "forklift-bundle.yml"

// A BundleManifest describes a Forklift pallet bundle.
type BundleManifest struct {
	// ForkliftVersion indicates that the pallet bundle was created assuming the semantics of a given
	// version of Forklift. The version must be a valid Forklift version, and it sets the minimum
	// version of Forklift required to use the pallet bundle. The Forklift tool refuses to use pallet
	// bundles declaring newer Forklift versions for any operations beyond printing information. The
	// Forklift version of the pallet bundle must be greater than or equal to the Forklift version of
	// every required Forklift pallet.
	ForkliftVersion string `yaml:"forklift-version"`
	// Pallet describes the basic metadata for the bundled pallet.
	Pallet BundlePallet `yaml:"pallet"`
	// Includes describes pallets used to define the bundle's package deployments.
	Includes BundleInclusions `yaml:"includes,omitempty"`
	// Imports lists the files imported from required pallets and the fully-qualified paths of those
	// source files (relative to their respective source pallets). Keys are the target paths of the
	// files, while values are lists showing the chain of provenance of the respective files (with
	// the deepest ancestor at the end of each list).
	Imports map[string][]string `yaml:"imports,omitempty"`
	// Deploys describes deployments provided by the bundle. Keys are names of deployments.
	Deploys map[string]fplt.DeplDecl `yaml:"deploys,omitempty"`
	// Downloads lists the downloadable paths of resources downloaded for creation and/or use of the
	// bundle. Keys are the names of the bundle's deployments which include downloads.
	Downloads map[string]BundleDeplDownloads `yaml:"downloads,omitempty"`
	// Exports lists the exposed paths of resources created by the bundle's deployments. Keys are
	// names of the bundle's deployments which provide resources.
	Exports map[string]BundleDeplExports `yaml:"exports,omitempty"`
}

// BundlePallet describes a bundle's bundled pallet.
type BundlePallet struct {
	// Path is the pallet bundle's path, which acts as the canonical name for the pallet bundle. It
	// should just be the path of the Git repository for the bundled pallet.
	Path string `yaml:"path"`
	// Version is the version or pseudoversion of the bundled pallet, if one can be determined.
	Version string `yaml:"version"`
	// Clean indicates whether the bundled pallet has been determined to have no changes beyond its
	// latest Git commit, if the pallet is version-controlled with Git. This does not account for
	// overrides of required pallets - those should be checked in BundleInclusions instead.
	Clean bool `yaml:"clean"`
	// Description is a short description of the bundled pallet to be shown to users.
	Description string `yaml:"description,omitempty"`
}

// BundleInclusions describes the requirements used to build the bundled pallet.
type BundleInclusions struct {
	// Pallets describes external pallets used to build the bundled pallet.
	Pallets map[string]BundlePalletInclusion `yaml:"pallets,omitempty"`
}

// BundlePalletInclusion describes a pallet used to build the bundled pallet.
type BundlePalletInclusion struct {
	Req fplt.PalletReq `yaml:"requirement,inline"`
	// Override describes the pallet used to override the required pallet, if an override was
	// specified for the pallet when building the bundled pallet.
	Override BundleInclusionOverride `yaml:"override,omitempty"`
	// Includes describes pallets used to define the pallet, omitting information about file imports.
	Includes map[string]BundlePalletInclusion `yaml:"includes,omitempty"`
	// Imports lists the files imported from the pallet, organized by import group. Keys are the names
	// of the import groups, and values are the results of evaluating the respective import groups -
	// i.e. maps whose keys are target file paths (where the files are imported to) and whose values
	// are source file paths (where the files are imported from).
	Imports map[string]map[string]string `yaml:"imports,omitempty"`
}

// BundleInclusionOverride describes a pallet used to override a required pallet.
type BundleInclusionOverride struct {
	// Path is the path of the override. This should be a filesystem path.
	Path string `yaml:"path"`
	// Version is the version or pseudoversion of the override, if one can be determined.
	Version string `yaml:"version"`
	// Clean indicates whether the override has been determined to have no changes beyond its latest
	// Git commit, if the it's version-controlled with Git.
	Clean bool `yaml:"clean"`
}

// BundleDeplDownloads lists the downloadable paths of resources which are downloaded for a
// deployment, whether during creation of the bundle or during staging of the bundle.
type BundleDeplDownloads struct {
	// HTTPFile lists HTTP(S) URLs of files downloaded for export by the deployment.
	HTTPFile []string `yaml:"http,omitempty"`
	// OCIImage lists URLs of OCI images downloaded either for export by the deployment or for use in
	// the deployment's Docker Compose app.
	OCIImage []string `yaml:"oci-image,omitempty"`
}

// BundleDeplExports lists the exposed paths of resources which are provided by a deployment.
type BundleDeplExports struct {
	// File lists the filesystem target paths of files exported by the deployment.
	File []string `yaml:"file,omitempty"`
	// ComposeApp lists the name of the Docker Compose app exported by the deployment.
	ComposeApp BundleDeplComposeApp `yaml:"compose-app,omitempty"`
}

// BundleDeplComposeApp lists information about a Docker Compose app provided by a deployment.
type BundleDeplComposeApp struct {
	// Name is the name of the Docker Compose app.
	Name string `yaml:"name,omitempty"`
	// Services lists the names of the services of the Docker Compose app.
	Services []string `yaml:"services,omitempty"`
	// Images lists the names of the container images used by services of the Docker Compose app.
	Images []string `yaml:"images,omitempty"`
	// CreatedBindMounts lists the names of the bind mounts created by the Docker Compose app.
	CreatedBindMounts []string `yaml:"created-bind-mounts,omitempty"`
	// RequiredBindMounts lists the names of the bind mounts required by the Docker Compose app.
	RequiredBindMounts []string `yaml:"required-bind-mounts,omitempty"`
	// CreatedVolumes lists the names of the volumes created by the Docker Compose app.
	CreatedVolumes []string `yaml:"created-volumes,omitempty"`
	// RequiredVolumes lists the names of the volumes required by the Docker Compose app.
	RequiredVolumes []string `yaml:"required-volumes,omitempty"`
	// CreatedNetworks lists the names of the networks created by the Docker Compose app.
	CreatedNetworks []string `yaml:"created-networks,omitempty"`
	// RequiredNetworks lists the names of the networks required by the Docker Compose app.
	RequiredNetworks []string `yaml:"required-networks,omitempty"`
}

// BundleManifest

// loadBundleManifest loads a BundleManifest from the specified file path in the provided base
// filesystem.
func loadBundleManifest(fsys ffs.PathedFS, filePath string) (BundleManifest, error) {
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
	return len(i.Pallets) > 0
}

func (i *BundleInclusions) HasOverrides() bool {
	for _, inclusion := range i.Pallets {
		if inclusion.Override != (BundleInclusionOverride{}) {
			return true
		}
	}
	return false
}

// BundleDownloads

func (d BundleDeplDownloads) Empty() bool {
	if len(d.HTTPFile) > 0 {
		return false
	}
	if len(d.OCIImage) > 0 {
		return false
	}
	return true
}

// BundleExports

func (d BundleDeplExports) Empty() bool {
	if len(d.File) > 0 {
		return false
	}
	if d.ComposeApp.Name != "" {
		return false
	}
	return true
}

// FSBundle: Manifests

func (b *FSBundle) WriteManifestFile() error {
	buf := bytes.Buffer{}
	encoder := yaml.NewEncoder(&buf)
	const yamlIndent = 2
	encoder.SetIndent(yamlIndent)
	if err := encoder.Encode(b.Manifest); err != nil {
		return errors.Wrapf(err, "couldn't marshal bundle manifest")
	}
	outputPath := filepath.FromSlash(path.Join(b.FS.Path(), BundleManifestFile))
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(outputPath, buf.Bytes(), perm); err != nil {
		return errors.Wrapf(err, "couldn't save bundle manifest to %s", outputPath)
	}
	return nil
}
