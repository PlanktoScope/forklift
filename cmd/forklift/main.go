package main

import (
	"log"
	"os"
	"runtime/debug"

	"github.com/carlmjohnson/versioninfo"
	"github.com/urfave/cli/v2"

	"github.com/forklift-run/forklift/cmd/forklift/cache"
	"github.com/forklift-run/forklift/cmd/forklift/dev"
	"github.com/forklift-run/forklift/cmd/forklift/host"
	"github.com/forklift-run/forklift/cmd/forklift/inspector"
	"github.com/forklift-run/forklift/cmd/forklift/plt"
	"github.com/forklift-run/forklift/cmd/forklift/stage"
	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
	"github.com/forklift-run/forklift/internal/clients/crane"
)

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var (
	defaultWorkspaceBase, _ = os.UserHomeDir()
	defaultPlatform         = crane.DetectPlatform().String()
)

var fcliVersions fcli.StagingVersions = fcli.StagingVersions{
	Core: fcli.Versions{
		Tool:               toolVersion,
		MinSupportedPallet: palletMinVersion,
	},
	MinSupportedBundle: bundleMinVersion,
	NewBundle:          newBundleVersion,
}

var app = &cli.App{
	Name:    "forklift",
	Version: toolVersion,
	Usage:   "Manages pallets and package deployments",
	Commands: []*cli.Command{
		plt.MakeCmd(plt.Versions{
			Staging:       fcliVersions,
			NewStageStore: newStageStoreVersion,
		}),
		stage.MakeCmd(stage.Versions{
			Tool:               toolVersion,
			MinSupportedBundle: bundleMinVersion,
			NewStageStore:      newStageStoreVersion,
		}),
		cache.Cmd,
		host.Cmd,
		inspector.Cmd,
		dev.MakeCmd(dev.Versions{
			Staging:       fcliVersions,
			NewStageStore: newStageStoreVersion,
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
		&cli.StringFlag{
			Name:    "stage-store",
			Aliases: []string{"ss"},
			Value:   "",
			Usage:   "Path of the forklift stage store, overriding the default path in the workspace",
			EnvVars: []string{"FORKLIFT_STAGE_STORE"},
		},
		&cli.BoolFlag{
			Name:    "ignore-tool-version",
			Value:   false,
			Usage:   "Ignore the version of the forklift tool in version compatibility checks",
			EnvVars: []string{"FORKLIFT_IGNORE_TOOL_VERSION"},
		},
		&cli.BoolFlag{
			Name:  "parallel",
			Value: true,
			Usage: "Allow parallel execution of I/O-bound tasks, such as downloading container images " +
				"or starting containers",
			EnvVars: []string{"FORKLIFT_PARALLEL"},
		},
		&cli.StringFlag{
			Name:    "platform",
			Value:   defaultPlatform,
			Usage:   "Select the os/arch or os/arch/variant platform for downloading container images",
			EnvVars: []string{"FORKLIFT_PLATFORM"},
		},
	},
	Suggest: true,
}

// Versioning

const (
	// palletMinVersion is the minimum supported Forklift version among pallets. A pallet with a
	// lower Forklift version cannot be used.
	palletMinVersion = "v0.4.0"
	// bundleMinVersion is the minimum supported Forklift version among staged pallet bundles. A
	// bundle with a lower Forklift version cannot be used.
	bundleMinVersion = "v0.7.0"
	// newBundleVersion is the Forklift version reported in new staged pallet bundles made by Forklift.
	// Older versions of the Forklift tool cannot use such bundles.
	newBundleVersion = "v0.8.0-alpha.6"
	// newStageStoreVersion is the Forklift version reported in a stage store initialized by Forklift.
	// Older versions of the Forklift tool cannot use the stage store.
	newStageStoreVersion = "v0.7.0"
	// fallbackVersion is the version reported which the Forklift tool reports itself as if its actual
	// version is unknown.
	fallbackVersion = "v0.9.0-dev"
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
