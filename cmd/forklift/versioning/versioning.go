// Package versioning deals with tool and spec version compatibility checking
package versioning

import (
	"runtime/debug"

	"github.com/carlmjohnson/versioninfo"
)

const (
	devVersion = "v0.4.0" // Git-independent fallback version used outside of builds
)

// DetermineToolVersion returns either a semver, a pseudoversion, or a Git hash based on information
// available from Go's `debug.ReadBuildInfo()`
func DetermineToolVersion(override string) string {
	if override != "" {
		return override
	}

	// Determine any version tags, if available
	if info, ok := debug.ReadBuildInfo(); ok &&
		info.Main.Version != "" && info.Main.Version != "(devel)" {
		v := info.Main.Version
		if versioninfo.DirtyBuild {
			v += "-dirty"
		}
		return v
	}
	if v := versioninfo.Version; v != "unknown" && v != "(devel)" {
		if versioninfo.DirtyBuild {
			v += "-dirty"
		}
		return v
	}

	// Fall back to whatever is available
	if r := versioninfo.Revision; r != "unknown" && r != "" {
		if versioninfo.DirtyBuild {
			r += "-dirty"
		}
		return r
	}
	return devVersion + "-dev"
}
