package forklift

import (
	"archive/tar"
	"bytes"
	"cmp"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	dct "github.com/compose-spec/compose-go/v2/types"
	"github.com/h2non/filetype"
	ftt "github.com/h2non/filetype/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	"github.com/forklift-run/forklift/pkg/structures"
)

// FSBundle

func NewFSBundle(path string) *FSBundle {
	return &FSBundle{
		FS: ffs.DirFS(path),
	}
}

// LoadFSBundle loads a FSBundle from a specified directory path in the provided base filesystem.
func LoadFSBundle(fsys ffs.PathedFS, subdirPath string) (b *FSBundle, err error) {
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
		if req.Req.VersionLock.Version, err = req.Req.VersionLock.Decl.Version(); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine requirement version of included pallet %s", path,
			)
		}
		b.Bundle.Manifest.Includes.Pallets[path] = req
	}
	return b, nil
}

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

func (b *FSBundle) Path() string {
	return b.FS.Path()
}

// FSBundle: Pallets

func (b *FSBundle) SetBundledPallet(pallet *FSPallet) error {
	shallow := pallet.FS
	for {
		merged, ok := shallow.(*ffs.MergeFS)
		if !ok {
			break
		}
		shallow = merged.Overlay
	}
	if shallow == nil {
		return errors.Errorf("pallet %s was not merged before bundling!", pallet.Path())
	}

	if err := CopyFS(shallow, filepath.FromSlash(b.getBundledPalletPath())); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files for unmerged pallet %s from %s", pallet.Path(), pallet.FS.Path(),
		)
	}

	if err := CopyFS(pallet.FS, filepath.FromSlash(b.getBundledMergedPalletPath())); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files for merged pallet %s from %s", pallet.Path(), pallet.FS.Path(),
		)
	}
	return nil
}

func CopyFS(fsys ffs.PathedFS, dest string) error {
	return fs.WalkDir(fsys, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			fileInfo, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(filepath.FromSlash(path.Join(dest, filePath)), fileInfo.Mode())
		}
		return copyFSFile(fsys, filePath, path.Join(dest, filePath), 0)
	})
}

func copyFSFile(fsys ffs.PathedFS, sourcePath, destPath string, destPerms fs.FileMode) error {
	if readLinkFS, ok := fsys.(ffs.ReadLinkFS); ok {
		sourceInfo, err := readLinkFS.StatLink(sourcePath)
		if err != nil {
			return errors.Wrapf(
				err, "couldn't stat source file %s for copying", path.Join(readLinkFS.Path(), sourcePath),
			)
		}
		if (sourceInfo.Mode() & fs.ModeSymlink) != 0 {
			return copyFSSymlink(readLinkFS, sourcePath, destPath)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Warning: %s was not loaded as a ReadLinkFS!\n", fsys.Path())
	}

	sourceFile, err := fsys.Open(sourcePath)
	fullSourcePath := path.Join(fsys.Path(), sourcePath)
	if err != nil {
		return errors.Wrapf(err, "couldn't open source file %s for copying", fullSourcePath)
	}
	defer func() {
		// FIXME: handle this error more rigorously
		if err := sourceFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: couldn't close source file %s\n", fullSourcePath)
		}
	}()
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return errors.Wrapf(err, "couldn't stat source file %s for copying", fullSourcePath)
	}
	if sourceInfo.IsDir() {
		fsys, err := fsys.Sub(sourcePath)
		if err != nil {
			return err
		}
		return CopyFS(fsys, destPath)
	}

	if destPerms == 0 {
		destPerms = sourceInfo.Mode().Perm()
	}
	destFile, err := os.OpenFile(
		destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, destPerms,
	)
	if err != nil {
		return errors.Wrapf(err, "couldn't open dest file %s for copying", destPath)
	}
	defer func() {
		// FIXME: handle this error more rigorously
		if err := destFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: couldn't close dest file %s\n", destPath)
		}
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return errors.Wrapf(err, "couldn't copy %s to %s", fullSourcePath, destPath)
	}

	return nil
}

func copyFSSymlink(fsys ffs.PathedFS, sourcePath, destPath string) error {
	readLinkFS, ok := fsys.(ffs.ReadLinkFS)
	if !ok {
		return errors.Errorf("%s is not a ReadLinkFS!", fsys.Path())
	}

	linkTarget, err := readLinkFS.ReadLink(sourcePath)
	if err != nil {
		return errors.Wrapf(err, "couldn't determine symlink target of %s", sourcePath)
	}
	return os.Symlink(linkTarget, destPath)
}

func (b *FSBundle) getBundledPalletPath() string {
	return path.Join(b.FS.Path(), bundledPalletDirName)
}

func (b *FSBundle) getBundledMergedPalletPath() string {
	return path.Join(b.FS.Path(), bundledMergedPalletDirName)
}

// FSBundle: Deployments

func (b *FSBundle) AddResolvedDepl(depl *ResolvedDepl) (err error) {
	b.Manifest.Deploys[depl.Name] = depl.Depl.Decl
	downloads := BundleDeplDownloads{}
	if downloads.HTTPFile, err = depl.GetHTTPFileDownloadURLs(); err != nil {
		return errors.Wrapf(
			err, "couldn't determine HTTP file downloads for deployment %s", depl.Depl.Name,
		)
	}
	if downloads.OCIImage, err = depl.GetOCIImageDownloadNames(); err != nil {
		return errors.Wrapf(
			err, "couldn't determine OCI image downloads for deployment %s", depl.Depl.Name,
		)
	}
	b.Manifest.Downloads[depl.Name] = downloads

	if err = CopyFS(depl.Pkg.FS, filepath.FromSlash(
		path.Join(b.getPackagesPath(), depl.Decl.Package),
	)); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files from package %s for deployment %s from %s",
			depl.Pkg.Path(), depl.Depl.Name, depl.Pkg.FS.Path(),
		)
	}

	exports := BundleDeplExports{}
	if exports.File, err = depl.GetFileExportTargets(); err != nil {
		return errors.Wrapf(err, "couldn't determine file exports of deployment %s", depl.Depl.Name)
	}
	definesComposeApp, err := depl.DefinesComposeApp()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't check deployment %s for a Compose app", depl.Depl.Name,
		)
	}
	if definesComposeApp {
		exports.ComposeApp, err = makeComposeAppSummary(depl, b.FS)
		if err != nil {
			return errors.Wrap(err, "couldn't make summary of Compose app definition")
		}
	}
	b.Manifest.Exports[depl.Name] = exports

	allOCIImages := make(structures.Set[string])
	allOCIImages.Add(downloads.OCIImage...)
	allOCIImages.Add(exports.ComposeApp.Images...)
	downloads.OCIImage = slices.Sorted(allOCIImages.All())
	b.Manifest.Downloads[depl.Name] = downloads

	if downloads.Empty() {
		delete(b.Manifest.Downloads, depl.Name)
	}
	if exports.Empty() {
		delete(b.Manifest.Exports, depl.Name)
	}
	return nil
}

func makeComposeAppSummary(
	depl *ResolvedDepl, bundleFS ffs.PathedFS,
) (BundleDeplComposeApp, error) {
	bundlePkg, err := fpkg.LoadFSPkg(bundleFS, path.Join(packagesDirName, depl.Decl.Package))
	if err != nil {
		return BundleDeplComposeApp{}, errors.Wrapf(
			err, "couldn't load bundled package %s", depl.Pkg.Path(),
		)
	}
	depl = &ResolvedDepl{
		Depl:   depl.Depl,
		PkgReq: depl.PkgReq,
		Pkg:    bundlePkg,
	}

	appDef, err := depl.LoadComposeAppDefinition(true)
	if err != nil {
		return BundleDeplComposeApp{}, errors.Wrap(err, "couldn't load Compose app definition")
	}

	services := make(structures.Set[string])
	images := make(structures.Set[string])
	for _, service := range appDef.Services {
		services.Add(service.Name)
		images.Add(service.Image)
	}

	createdBindMounts, requiredBindMounts := makeComposeAppBindMountSummaries(appDef, bundleFS.Path())
	createdVolumes, requiredVolumes := makeComposeAppVolumeSummaries(appDef)
	createdNetworks, requiredNetworks := makeComposeAppNetworkSummaries(appDef)

	app := BundleDeplComposeApp{
		Name:               appDef.Name,
		Services:           slices.Sorted(services.All()),
		Images:             slices.Sorted(images.All()),
		CreatedBindMounts:  slices.Sorted(createdBindMounts.All()),
		RequiredBindMounts: slices.Sorted(requiredBindMounts.All()),
		CreatedVolumes:     slices.Sorted(createdVolumes.All()),
		RequiredVolumes:    slices.Sorted(requiredVolumes.All()),
		CreatedNetworks:    slices.Sorted(createdNetworks.All()),
		RequiredNetworks:   slices.Sorted(requiredNetworks.All()),
	}
	return app, nil
}

func makeComposeAppBindMountSummaries(
	appDef *dct.Project, bundleRoot string,
) (created structures.Set[string], required structures.Set[string]) {
	created = make(structures.Set[string])
	required = make(structures.Set[string])
	for _, service := range appDef.Services {
		for _, volume := range service.Volumes {
			if volume.Type != "bind" {
				continue
			}
			// If the path on the host is declared as a relative path, then it's supposed to be a path
			// managed by Forklift, and its location will depend on where the bundle is. So we record it
			// relative to the path of the bundle.
			volume.Source = strings.TrimPrefix(volume.Source, bundleRoot+"/")
			if volume.Bind != nil && !volume.Bind.CreateHostPath {
				required.Add(volume.Source)
				continue
			}
			created.Add(volume.Source)
		}
	}

	return created.Difference(required), required
}

func makeComposeAppVolumeSummaries(
	appDef *dct.Project,
) (created structures.Set[string], required structures.Set[string]) {
	created = make(structures.Set[string])
	required = make(structures.Set[string])
	for volumeName, volume := range appDef.Volumes {
		if volume.External {
			required.Add(cmp.Or(volume.Name, volumeName))
			continue
		}
		created.Add(cmp.Or(volume.Name, volumeName))
	}
	return created, required
}

func makeComposeAppNetworkSummaries(
	appDef *dct.Project,
) (created structures.Set[string], required structures.Set[string]) {
	created = make(structures.Set[string])
	required = make(structures.Set[string])
	for networkName, network := range appDef.Networks {
		if network.External {
			if networkName == "default" && network.Name == "none" {
				// If the network is Docker's pre-made "none" network (which uses the null network driver),
				// we ignore it for brevity since the intention is to suppress creating a network for the
				// container.
				continue
			}
			required.Add(cmp.Or(network.Name, networkName))
			continue
		}
		created.Add(cmp.Or(network.Name, networkName))
	}
	return created, required
}

func (b *FSBundle) LoadDepl(name string) (Depl, error) {
	depl, ok := b.Manifest.Deploys[name]
	if !ok {
		return Depl{}, errors.Errorf("bundle does not contain package deployment %s", name)
	}
	return Depl{
		Name: name,
		Decl: depl,
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
			Decl: b.Manifest.Deploys[name],
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

// FSBundle: Exports

func (b *FSBundle) getExportsPath() string {
	return path.Join(b.FS.Path(), exportsDirName)
}

func (b *FSBundle) WriteFileExports(dlCache *FSDownloadCache) error {
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
			exportPath := path.Join(b.getExportsPath(), export.Target)
			if err := EnsureExists(filepath.FromSlash(path.Dir(exportPath))); err != nil {
				return errors.Wrapf(
					err, "couldn't make export directory %s in bundle", path.Dir(exportPath),
				)
			}
			switch export.SourceType {
			case fpkg.FileExportSourceTypeLocal:
				if err := exportLocalFile(resolved, export, exportPath); err != nil {
					return err
				}
			case fpkg.FileExportSourceTypeHTTP:
				if err := exportHTTPFile(export, exportPath, dlCache); err != nil {
					return err
				}
			case fpkg.FileExportSourceTypeHTTPArchive, fpkg.FileExportSourceTypeOCIImage:
				if err := exportArchiveFile(export, exportPath, dlCache); err != nil {
					return err
				}
			default:
				return errors.Errorf("unknown file export source type: %s", export.SourceType)
			}
		}
	}
	return nil
}

func exportLocalFile(resolved *ResolvedDepl, export fpkg.FileExportRes, exportPath string) error {
	if err := copyFSFile(
		resolved.Pkg.FS, strings.TrimPrefix(export.Source, "/"), filepath.FromSlash(exportPath),
		export.Permissions,
	); err != nil {
		return errors.Wrapf(err, "couldn't export file from %s to %s", export.Source, exportPath)
	}
	return nil
}

func exportHTTPFile(export fpkg.FileExportRes, exportPath string, dlCache *FSDownloadCache) error {
	sourcePath, err := dlCache.GetFilePath(export.URL)
	if err != nil {
		return errors.Wrapf(err, "couldn't determine cache path for HTTP download %s", export.URL)
	}
	if err := copyFSFile(
		dlCache.FS, strings.TrimPrefix(strings.TrimPrefix(sourcePath, dlCache.FS.Path()), "/"),
		filepath.FromSlash(exportPath),
		export.Permissions,
	); err != nil {
		return errors.Wrapf(err, "couldn't export file from %s to %s", sourcePath, exportPath)
	}
	return nil
}

func exportArchiveFile(
	export fpkg.FileExportRes, exportPath string, dlCache *FSDownloadCache,
) error {
	kind, err := determineFileType(export, dlCache)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't determine file type of cached download archive %s", export.URL,
		)
	}

	var archiveFile fs.File
	switch export.SourceType {
	default:
		return errors.Errorf("couldn't open downloaded archive of type %s", export.SourceType)
	case fpkg.FileExportSourceTypeHTTPArchive:
		if archiveFile, err = dlCache.OpenFile(export.URL); err != nil {
			return errors.Wrapf(err, "couldn't open cached http download archive %s", export.URL)
		}
	case fpkg.FileExportSourceTypeOCIImage:
		if archiveFile, err = dlCache.OpenOCIImage(export.URL); err != nil {
			return errors.Wrapf(err, "couldn't open cached oci image download tarball %s", export.URL)
		}
	}
	defer func() {
		if err := archiveFile.Close(); err != nil {
			// TODO: handle this error more rigorously
			fmt.Fprintf(os.Stderr, "Error: couldn't close cached download archive %s\n", export.URL)
		}
	}()

	var archiveReader *tar.Reader
	switch kind.MIME.Value {
	case "application/x-tar":
		archiveReader = tar.NewReader(archiveFile)
	case "application/gzip":
		uncompressed, err := gzip.NewReader(archiveFile)
		if err != nil {
			return errors.Wrapf(err, "couldn't create a gzip decompressor for %s", export.URL)
		}
		// TODO: check to ensure that the uncompressed file is actually a tar archive
		defer func() {
			_ = uncompressed.Close()
		}()
		archiveReader = tar.NewReader(uncompressed)
	default:
		return errors.Errorf(
			"unrecognized archive file type: %s (.%s)", kind.MIME.Value, kind.Extension,
		)
	}
	if err = extractFromArchive(
		archiveReader, export.Source, exportPath, export.Permissions,
	); err != nil {
		return errors.Wrapf(
			err, "couldn't extract %s from cached download archive %s to %s",
			export.Source, export.URL, exportPath,
		)
	}
	return nil
}

func determineFileType(
	export fpkg.FileExportRes, dlCache *FSDownloadCache,
) (ft ftt.Type, err error) {
	var archiveFile fs.File
	switch export.SourceType {
	default:
		return filetype.Unknown, errors.Errorf(
			"couldn't open downloaded archive of type %s", export.SourceType,
		)
	case fpkg.FileExportSourceTypeHTTPArchive:
		if archiveFile, err = dlCache.OpenFile(export.URL); err != nil {
			return filetype.Unknown, errors.Wrapf(
				err, "couldn't open cached http download archive %s", export.URL,
			)
		}
	case fpkg.FileExportSourceTypeOCIImage:
		if archiveFile, err = dlCache.OpenOCIImage(export.URL); err != nil {
			return filetype.Unknown, errors.Wrapf(
				err, "couldn't open cached oci image download tarball %s", export.URL,
			)
		}
	}
	defer func() {
		if err := archiveFile.Close(); err != nil {
			// TODO: handle this error more rigorously
			fmt.Fprintf(os.Stderr, "Error: couldn't close cached download %s\n", export.URL)
		}
	}()
	return filetype.MatchReader(archiveFile)
}

func extractFromArchive(
	tarReader *tar.Reader, sourcePath, exportPath string, destPerms fs.FileMode,
) error {
	if sourcePath == "/" || sourcePath == "." {
		sourcePath = ""
	}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if sourcePath != "" && sourcePath != header.Name &&
			!strings.HasPrefix(header.Name, sourcePath+"/") {
			continue
		}

		if err = extractFile(header, tarReader, sourcePath, exportPath, destPerms); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(
	// FIXME: also handle destPerms for directories and symlinks!
	header *tar.Header, tarReader *tar.Reader, sourcePath, exportPath string, destPerms fs.FileMode,
) error {
	targetPath := path.Join(exportPath, strings.TrimPrefix(header.Name, sourcePath))
	switch header.Typeflag {
	default:
		return errors.Errorf("unknown type of file %s in archive: %b", header.Name, header.Typeflag)
	case tar.TypeDir:
		if err := EnsureExists(filepath.FromSlash(targetPath)); err != nil {
			return errors.Wrapf(
				err, "couldn't export directory %s from archive to %s", header.Name, targetPath,
			)
		}
	case tar.TypeReg:
		if err := extractRegularFile(header, tarReader, sourcePath, targetPath, destPerms); err != nil {
			return errors.Wrapf(
				err, "couldn't export regular file %s from archive to %s", header.Name, targetPath,
			)
		}
	case tar.TypeSymlink:
		if err := os.Symlink(
			filepath.FromSlash(header.Linkname), filepath.FromSlash(targetPath),
		); err != nil {
			return errors.Wrapf(
				err, "couldn't export symlink %s from archive to %s", header.Name, targetPath,
			)
		}
	case tar.TypeLink:
		if err := os.Link(
			filepath.FromSlash(path.Join(exportPath, strings.TrimPrefix(header.Linkname, sourcePath))),
			filepath.FromSlash(targetPath),
		); err != nil {
			return errors.Wrapf(
				err, "couldn't export hardlink %s from archive to %s", header.Name, targetPath,
			)
		}
	}
	return nil
}

func extractRegularFile(
	header *tar.Header, tarReader *tar.Reader, sourcePath, targetPath string, destPerms fs.FileMode,
) error {
	if destPerms == 0 {
		destPerms = fs.FileMode( //nolint:gosec // (G115) tar's Mode won't(?) overflow fs.FileMode
			header.Mode,
		) & fs.ModePerm
	}
	// FIXME: we suppress gosec G304 below, but for security we should check targetPath to ensure it's
	// a valid path (i.e. within the Forklift workspace)!
	targetFile, err := os.OpenFile(
		filepath.Clean(filepath.FromSlash(targetPath)), os.O_RDWR|os.O_CREATE|os.O_TRUNC, destPerms,
	)
	if err != nil {
		return errors.Wrapf(err, "couldn't create export file at %s", targetPath)
	}
	defer func(file fs.File, filePath string) {
		if err := file.Close(); err != nil {
			// FIXME: handle this error more rigorously
			fmt.Fprintf(os.Stderr, "Error: couldn't close export file %s\n", filePath)
		}
	}(targetFile, targetPath)

	if _, err = io.Copy(targetFile, tarReader); err != nil {
		return errors.Wrapf(
			err, "couldn't copy file %s in tar archive to %s", sourcePath, targetPath,
		)
	}
	return nil
}

// FSBundle: FSPalletLoader

func (b *FSBundle) LoadFSPallet(palletPath string, version string) (*FSPallet, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	return LoadFSPallet(b.FS, path.Join(packagesDirName, palletPath))
}

func (b *FSBundle) LoadFSPallets(searchPattern string) ([]*FSPallet, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	return LoadFSPallets(b.FS, path.Join(packagesDirName, searchPattern))
}

func (b *FSBundle) LoadFSPkgTree() (*fpkg.FSPkgTree, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	fsys, err := b.FS.Sub(packagesDirName)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load package tree from bundle")
	}
	return &fpkg.FSPkgTree{
		FS: fsys,
	}, nil
}

// FSBundle: FSPkgLoader

func (b *FSBundle) LoadFSPkg(pkgPath string, version string) (*fpkg.FSPkg, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	pkgTree, err := b.LoadFSPkgTree()
	if err != nil {
		return nil, err
	}
	return pkgTree.LoadFSPkg(strings.TrimLeft(pkgPath, "/"))
}

func (b *FSBundle) LoadFSPkgs(searchPattern string) ([]*fpkg.FSPkg, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	pkgTree, err := b.LoadFSPkgTree()
	if err != nil {
		return nil, err
	}
	return pkgTree.LoadFSPkgs(searchPattern)
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
