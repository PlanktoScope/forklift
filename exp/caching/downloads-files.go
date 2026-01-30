package caching

import (
	"io/fs"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"
)

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
