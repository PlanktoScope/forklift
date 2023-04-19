// Package workspace handles forklift workspace operations
package workspace

import (
	"io/fs"
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

// Env

const envDirName = "env"

func LocalEnvPath(workspacePath string) string {
	return filepath.Join(workspacePath, envDirName)
}

func RemoveLocalEnv(workspacePath string) error {
	return os.RemoveAll(LocalEnvPath(workspacePath))
}

func LocalEnvFS(workspacePath string) fs.FS {
	return os.DirFS(LocalEnvPath(workspacePath))
}

// Pallets

const palletsDirName = "pallets"

func LocalPalletsPath(workspacePath string) string {
	return filepath.Join(workspacePath, palletsDirName)
}
