// Package workspace handles forklift workspace operations
package workspace

import (
	"os"
	"path/filepath"
)

func Exists(path string) bool {
	dir, err := os.Stat(path)
	if err == nil && dir.IsDir() {
		return true
	}
	return false
}

func EnsureExists(path string) error {
	const perm = 755 // owner rwx, group rx, public rx
	return os.MkdirAll(path, perm)
}

func LocalEnvPath(workspacePath string) string {
	return filepath.Join(workspacePath, "env")
}

func RemoveLocalEnv(workspacePath string) error {
	return os.RemoveAll(LocalEnvPath(workspacePath))
}
