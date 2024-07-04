package forklift

import (
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
)

// FSDownloadCache

// Exists checks whether the cache actually exists on the OS's filesystem.
func (c *FSDownloadCache) Exists() bool {
	return DirExists(filepath.FromSlash(c.FS.Path()))
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (c *FSDownloadCache) Remove() error {
	return os.RemoveAll(filepath.FromSlash(c.FS.Path()))
}

// Path returns the path of the cache's filesystem.
func (c *FSDownloadCache) Path() string {
	return c.FS.Path()
}

// FSDownloadCache: Files

// GetFilePath returns the path where the file from the specified URL should be stored in the
// cache's filesystem, if it is in the cache.
func (c *FSDownloadCache) GetFilePath(downloadURL string) (string, error) {
	normalized, err := normalizeHTTPDownloadURL(downloadURL)
	if err != nil {
		return "", err
	}
	return path.Join(c.FS.Path(), normalized), nil
}

func normalizeHTTPDownloadURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't parse URL: %s", rawURL)
	}
	u.Scheme = ""
	u.User = nil
	return path.Join("http-files", strings.TrimPrefix(u.String(), "//")), nil
}

// HasFile checks whether the file from the specified URL is stored in the cache.
func (c *FSDownloadCache) HasFile(downloadURL string) (bool, error) {
	if c == nil {
		return false, errors.New("cache is nil")
	}

	normalized, err := normalizeHTTPDownloadURL(downloadURL)
	if err != nil {
		return false, err
	}
	if _, err = fs.Stat(c.FS, normalized); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return err == nil, nil
}

// LoadFile loads the file downloaded from the specified URL.
func (c *FSDownloadCache) LoadFile(downloadURL string) ([]byte, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	u, err := normalizeHTTPDownloadURL(downloadURL)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't process URL of cached file: %s", downloadURL)
	}
	return fs.ReadFile(c.FS, u)
}

// OpenFile opens the file downloaded from the specified URL.
func (c *FSDownloadCache) OpenFile(downloadURL string) (fs.File, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	u, err := normalizeHTTPDownloadURL(downloadURL)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't process URL of cached file: %s", downloadURL)
	}
	return c.FS.Open(u)
}

// FSDownloadCache: OCI Images

// GetOCIImagePath returns the path where the OCI container image with the specified image name
// should be stored in the cache's filesystem, if it is in the cache.
func (c *FSDownloadCache) GetOCIImagePath(imageName string) (string, error) {
	normalized, err := normalizeOCIImageName(imageName)
	if err != nil {
		return "", err
	}
	return path.Join(c.FS.Path(), normalized), nil
}

func normalizeOCIImageName(rawImageName string) (string, error) {
	ref, err := name.ParseReference(rawImageName, name.StrictValidation)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't parse image name: %s", rawImageName)
	}
	parsed := ref.Name()

	parsed = strings.ReplaceAll(parsed, ":", "/") // turn the tag into a directory
	return path.Join("oci-image-fs-tarballs", parsed+".tar"), nil
}

// HasOCIImage checks whether the OCI container image with the specified image name is stored in the
// cache.
func (c *FSDownloadCache) HasOCIImage(imageName string) (bool, error) {
	if c == nil {
		return false, errors.New("cache is nil")
	}

	normalized, err := normalizeOCIImageName(imageName)
	if err != nil {
		return false, err
	}
	if _, err = fs.Stat(c.FS, normalized); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return err == nil, nil
}

// OpenOCIImage opens the OCI container image downloaded from the specified image name.
func (c *FSDownloadCache) OpenOCIImage(imageName string) (fs.File, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	u, err := normalizeOCIImageName(imageName)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't process URL of cached file: %s", imageName)
	}
	return c.FS.Open(u)
}
