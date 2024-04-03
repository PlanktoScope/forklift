package main

import (
	"log"
	"os"
	"runtime/debug"

	"github.com/carlmjohnson/versioninfo"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/cmd/forklift/cache"
	"github.com/PlanktoScope/forklift/cmd/forklift/dev"
	"github.com/PlanktoScope/forklift/cmd/forklift/host"
	"github.com/PlanktoScope/forklift/cmd/forklift/plt"
	"github.com/PlanktoScope/forklift/cmd/forklift/stage"
)

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var defaultWorkspaceBase, _ = os.UserHomeDir()

var app = &cli.App{
	Name:    "forklift",
	Version: toolVersion,
	Usage:   "Manages pallets and package deployments",
	Commands: []*cli.Command{
		plt.MakeCmd(plt.Versions{
			Tool:               toolVersion,
			MinSupportedRepo:   repoMinVersion,
			MinSupportedPallet: palletMinVersion,
			NewBundle:          newBundleVersion,
			NewStageStore:      newStageStoreVersion,
		}),
		stage.MakeCmd(stage.Versions{
			Tool:               toolVersion,
			MinSupportedBundle: bundleMinVersion,
			NewStageStore:      newStageStoreVersion,
		}),
		cache.Cmd,
		host.Cmd,
		dev.MakeCmd(dev.Versions{
			Tool:               toolVersion,
			MinSupportedRepo:   repoMinVersion,
			MinSupportedPallet: palletMinVersion,
			NewBundle:          newBundleVersion,
			NewStageStore:      newStageStoreVersion,
		}),
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workspace",
			Aliases: []string{"ws"},
			Value:   defaultWorkspaceBase,
			Usage:   "Path of the forklift workspace",
			EnvVars: []string{"FORKLIFT_WORKSPACE"},
		},
		&cli.BoolFlag{
			Name:    "ignore-tool-version",
			Value:   false,
			Usage:   "Ignore the version of the forklift tool in version compatibility checks",
			EnvVars: []string{"FORKLIFT_IGNORE_TOOL_VERSION"},
		},
	},
	Suggest: true,
}

// Versioning

const (
	// repoMinVersion is the minimum supported Forklift version among repos. A repo with a lower
	// Forklift version cannot be used.
	repoMinVersion = "v0.4.0"
	// palletMinVersion is the minimum supported Forklift version among pallets. A pallet with a
	// lower Forklift version cannot be used.
	palletMinVersion = "v0.4.0"
	// bundleMinVersion is the minimum supported Forklift version among staged pallet bundles. A
	// bundle with a lower Forklift version cannot be used.
	bundleMinVersion = "v0.7.0-dev"
	// newBundleVersion is the Forklift version reported in new staged pallet bundles made by Forklift.
	// Older versions of the Forklift tool cannot use such bundles.
	newBundleVersion = "v0.7.0-dev"
	// newStageStoreVersion is the Forklift version reported in a stage store initialized by Forklift.
	// Older versions of the Forklift tool cannot use the state store.
	newStageStoreVersion = "v0.7.0-dev"
	// fallbackVersion is the version reported which the Forklift tool reports itself as if its actual
	// version is unknown.
	fallbackVersion = "v0.7.0-dev"
)

var (
	toolVersion = determineVersion(buildSummary, fallbackVersion)
	// buildSummary should be overridden by ldflags, such as with GoReleaser's "Summary".
	buildSummary = ""
)

// determineVersion returns either a semver, a pseudoversion, or a Git hash based on information
// available from Go's `debug.ReadBuildInfo()`.
func determineVersion(override, fallback string) string {
	if override != "" {
		return override
	}

	const dirtySuffix = "-dirty"
	// Determine any version tags, if available
	if info, ok := debug.ReadBuildInfo(); ok &&
		info.Main.Version != "" && info.Main.Version != "(devel)" {
		v := info.Main.Version
		if versioninfo.DirtyBuild {
			v += dirtySuffix
		}
		return v
	}
	if v := versioninfo.Version; v != "unknown" && v != "(devel)" {
		if versioninfo.DirtyBuild {
			v += dirtySuffix
		}
		return v
	}

	// Fall back to whatever is available
	if r := versioninfo.Revision; r != "unknown" && r != "" {
		if versioninfo.DirtyBuild {
			r += dirtySuffix
		}
		return r
	}
	return fallback
}
