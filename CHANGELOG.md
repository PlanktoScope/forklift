# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Changed

- (Breaking change; cli) The verbs `rm` and `remove` have been deleted from all commands (e.g. `forklift pallet rm`), because the two-character verb `rm` doesn't line up nicely with the three-character verb `add`; `del` or `delete` should be used instead (e.g. `forklift pallet del`).

### Fixed

- (cli) Git progress log messages are now printed to stderr instead of stdout.

## 0.8.0-alpha.5 - 2024-12-08

### Changed

- (Breaking change; cli) Log messages are now (supposed to be) printed to stderr instead of stdout; stdout is only meant to be used for outputting values meant to be piped or captured in a subshell.

## 0.8.0-alpha.4 - 2024-12-07

### Added

- (cli) Added a `inspector resolve-git-repo` command to resolve version queries on git repositories.

### Fixed

- (cli) The `[dev] plt show-plt-version` and `[dev] plt show-repo-version` commands no longer require the required pallets/repos to be cached before the respective commands work.

## 0.8.0-alpha.3 - 2024-12-06

### Added

- (cli) Added `[dev] plt show-plt-version` and `[dev] plt show-repo-version` commands to print the version/pseudoversion string for the required version of the specified pallet/repo.
- (cli) Added a `stage show-next-index` command to print the index of the next staged pallet bundle (if it exists).
- (cli) Added a `--platform` flag (and `FORKLIFT_PLATFORM` env var) to override the auto-detected platform (e.g. `linux/amd64` or `linux/arm64`) used for downloading container images for file exports and for pre-downloading container images needed for the next `forklift stage apply`.

## 0.8.0-alpha.2 - 2024-09-22

### Added

- (cli) Added support for `add-feature` and `remove-feature` types to file import group modifiers. `add-feature` will add all files determined by evaluation of a named feature flag exposed by the import group's referenced pallet, while `remove-feature` will remove those files. Pallet feature flags are constructed with the same file schema as file import groups, but are located in the pallet's `/features` directory and have a `.feature.yml` file extension instead.
- (cli) Added a `[dev] plt ls-feat` command to list feature flags exposed by the local/development pallet.
- (cli) Added a `[dev] plt show-feat` command to show the specified feature exposed by the local/development pallet, including any deprecation notices of deprecated features referenced directly or indirectly by this feature.
- (cli) Added a `[dev] plt ls-plt-feat` command to list feature flags exposed by the specified pallet required by the local/development pallet.
- (cli) Added a `[dev] plt show-plt-feat` command to show the specified feature exposed by the specified pallet required by the local/development pallet, including any deprecation notices of deprecated features referenced directly or indirectly by this feature.
- (cli) Added a `[dev] plt ls-plt-file` command to list files in the specified pallet required by the local/development pallet, including files imported by that required pallet from its own required pallets.
- (cli) Added a `[dev] plt locate-plt-file` command to print the actual filesystem path of the specified file in the specified pallet required by the local/development pallet.
- (cli) Added a `[dev] plt show-plt-file` command to print the contents of the specified file in the specified pallet required by the local/development pallet.
- (cli) Added a `cache rm-dl` command to delete the cache of downloaded files for export.
- (cli) The bundle manifest's `includes` section's description of required pallets now reports when required pallets were overridden.
- (cli) The bundle manifest's `includes` section's description of required pallets now recursively shows information about transitively-required pallets (but does not show information about file import groups in those transitively-required pallets).
- (cli) The bundle manifest's `includes` section's description of required pallets now shows the results (as target file path -> source file path mappings) of evaluating each file import group attached to their respective required pallets.
- (cli) The bundle manifest now has an `imports` section which describes the provenance of each imported file, as a list of how the file has been transitively imported across pallets (with pallets farther down the list being depeer in the transitive import chain).

### Changed

- (Breaking change; cli) Removed some aliases for `[dev] plt add-plt` and `[dev] plt add-repo` which should not have been added, because they were constructed as a combination of an abbrebiation and an unabbreviated word.
- (Breaking change; cli) Now, by default `[dev] plt ls-file` and `[dev] plt ls-plt-file` don't list files in hidden directories (i.e. directories whose names start with `.`) at the root of the pallet. To list all files including those in hidden directories, you should now specify `**` as the file path glob (e.g. by running `[dev] plt ls-file '**'` or `[dev] plt ls-plt-file required_pallet_path '**'`).
- (cli) Git repository mirrors of pallets and package repos in the cache are now stored in a single `mirrors` subdirectory of the Forklift workspace, rather than being split/duplicated across the `pallets` and `repositories` subdirectories.
- (cli) Suppressed some noisy Git cloning output in `[dev] plt cache-plt`, `[dev] plt cache-all`, and other related commands.
- (cli) `[dev] plt show-imp` now shows any deprecated notices of deprecated features referenced directly or indirectly by the specified import group.

### Fixed

- (cli) Now repos with packages constructed through pallet layering (for repos which are also layered pallets) can actually be used by other pallets as sources of packages. This is done by merging the pallet as part of the work of downloading it into the cache as a repo.
- (cli) Transitive imports of files across pallets (e.g. importing a file from a pallet which actually imports that file from another pallet) is no longer completely broken (it should work, but there may still be undiscovered bugs because the code paths have not been thoroughly tested).
- (cli) `[dev] plt cache-plt`, `[dev] plt cache-all`, and other related commands now recursively cache all transitively-required pallets of the local/development pallet, instead of only caching directly-required pallets.
- (cli) `plt switch` and `plt upgrade` now fetch changes (i.e. branch/tag refs and commit objects) from all remotes before checking whether the current commit of the local pallet exists on some remote, in order to prevent that check from spuriously failing when the remotes have new commits not yet in the local pallet.

## 0.8.0-alpha.1 - 2024-08-30

### Added

- (cli) Added a `[dev] plt add-plt` command to add a pallet requirement to the local/development pallet.
- (cli) Added a `[dev] plt rm-plt` command to remove a pallet requirement from the local/development pallet.
- (cli) Added a `[dev] plt ls-plt` command to list all pallets required by the local/development pallet.
- (cli) Added a `[dev] plt show-plt` command to show the specified pallet required by the local/development pallet.
- (cli) Added a `[dev] plt cache-plt` command to cache all pallets required by the local/development pallet.
- (cli) Added a `[dev] plt ls-imp` command to list all file import groups declared by the local/development pallet.
- (cli) Added a `[dev] plt show-imp` command to show the specified file import group declared by the local/development pallet.
- (cli) Added a `[dev] plt locate-repo` command to print the actual filesystem path of the specified available package repository. The actual filesystem path may be for a subdirectory in the repositories cache, or a subdirectory in an override repository (in the case of `dev plt` with the `--repos` flag).
- (cli) Added a `[dev] plt locate-pkg` command to print the actual filesystem path of the specified available package. The actual filesystem path may be for a subdirectory in the repositories cache, or a subdirectory in an override repository (in the case of `dev plt` with the `--repos` flag), or a subdirectory in the local/development pallet (in the case of a local package defined by the pallet), or a subdirectory in a required pallet (in the case of a local package imported from another pallet).
- (cli) Added a `[dev] plt ls-file` command to list files in the local/development pallet, including files imported by the pallet from required pallets.
- (cli) Added a `[dev] plt locate-file` command to print the actual filesystem path of the specified file in the pallet. The actual filesystem path may be for a file in the pallets cache, or a file in an override pallet (in the case of `dev plt` with the `--plts` flag), or a file in the local/development pallet (in the case of a local file defined by the pallet), or a file in a required pallet (in the case of a file imported from another pallet).
- (cli) Added a `[dev] plt show-file` command to print the contents of the specified file in the local/development pallet.
- (cli) Added a `[dev] plt edit-file` command to edit the specified file in the local/development pallet, using the editor set by the `$EDITOR` environment variable. If the file was previously only in an underlay, a temporary copy is provided to the editor; if changes are saved when the editor quits, the changed file will be saved as an override file into the local/development pallet.
- (cli) Added a `[dev] plt rm-file` command to delete the specified file/directory in the local/development pallet. If a file/directory still exists after the deletion because of files imported from other pallets, they are listed in a warning message.
- (cli) Added a `[dev] plt ls-dl` command to list all HTTP files and OCI images downloaded by the local/development pallet.
- (cli) Added an optional `--plts` flag to `dev plt` for overriding version-locked required pallets with pallets from other directories, like the existing `--repos` flag.

### Changed

- (Breaking change; cli) Now enabled feature flags in each package deployment are considered in alphabetically-sorted order (rather than the exact order used for listing feature flags in the package deployment declaration file) when sequencing file export operations implied by the enabled feature flags.
- (cli) Now the `[dev] plt cache-all` command, and all commands which can cache staging requirements, will cache pallets required by the local/development pallet.
- (cli) Now all `[dev] plt` commands are evaluated on the merged pallet (i.e. with file imports) if the pallet imports files from other pallets.
- (cli) Now the `[dev] plt show-pkg` and `cache show-pkg` commands also print information about file exports.
- (cli) Now the commands for viewing a pallet/repo (e.g. `[dev] plt show`) truncate the printout of the pallet/repo's readme file to the first ten lines of the file, to prevent long readme files from clogging up the command output.
- (cli) Now the `[dev] plt rm-repo` command only deletes the version lock file for the specified repository, instead of deleting the entire subdirectory for the repository.
- (cli) Now a pallet can include deployments for local packages even if it's missing a `forklift-repository.yml` file in the pallet root - in such cases, the repository declaration is automatically inferred from the `forklift-pallet.yml` file for the purposes of using the pallet. However, a `forklift-repository.yml` file is still needed to make the pallet usable as a Forklift package repository by other pallets.
- (cli) Now git clone/fetch-related messages are properly indented in command output to stdout.
- (cli) Now Docker image pull & Compose app change messages are properly indented in command output to stdout.

### Fixed

- (spec) Fixed an incorrect example for the `target` field of the file export object in the packaging spec.
- (spec) Fixed a formatting error in the description of the `url` field of the file export object
- (cli) Fixed a regression where the `[dev] plt ls-pkg` command failed with a `cache is nil` error on pallets which are not also package repositories.
- (cli) Fixed the behavior of `[dev] plt ls-pkg` to include local packages (i.e. those declared in the pallet) in the displayed list of packages.

## 0.8.0-alpha.0 - 2024-07-05

### Added

- (cli) Added tracking of the last pallet path@version query used with the `plt clone` and `plt switch` subcommands, so that those subcommands can be called again with a partial query (i.e. `@version_query` or `pallet_path@` or `@`) to reuse the last provided value(s) for omitted parts of the query.
- (cli) Added a `--force` flag to the `plt switch` subcommand.
- (cli) Added a `plt upgrade` subcommand as a upgrade-specific version of `plt switch` (with additional checks and log messages).
- (cli) Added a `plt check-upgrade` subcommand to show whether an upgrade is available and, if so, what change to the local pallet would be made by `plt upgrade`.
- (cli) Added a `plt show-upgrade-query` subcommand to show the pallet path@version query which will be used for `plt upgrade` and for `plt clone/switch` subcommands with partial queries.
- (cli) Added a `plt set-upgrade-query` subcommand to modify the pallet path@version query which will be used for `plt upgrade` and for `plt clone/switch` subcommands with partial queries.
- (cli) Now `plt clone` and `plt switch` add a `forklift-cache-mirror` remote to the list of remotes of the local pallet, which points to the Forklift pallet cache's mirror of the `origin` remote of the local pallet.
- (cli) Now `plt show` will print git refs from the Forklift pallet cache's mirror of the `origin` remote of the local pallet, if the `origin` remote cannot be queried (e.g. due to lack of internet connection).

### Changed

- (Breaking change; cli) Now `plt switch` will quit early with an error message if you use it to try to replace a local pallet which 1) is not a Git repo, 2) has uncommitted changes, or 3) is on a commit which does not exist in the remote, unless you enable the `--force` flag. This is intended to prevent unintentional deletion of user customizations.

### Fixed

- (cli) Previously, the atomic commit mechanism for the stage store state file did not correctly error out if a swap file already existed. This check should now work.

## 0.7.3 - 2024-06-03

### Added

- (cli) Added a `[dev] plt ls-img` subcommand to list all images deployed by the pallet.
- (cli) Subcommands with an `rm` verb (short for `remove`) now have `del` verb alias (short for `delete`). This enables familiarity with apk (Alpine Package Keeper) commands, and also enables `add`/`del` as a 3-character verb pair instead of `add`/`rm` as a verb pair with inconsistent verb length.

## 0.7.2 - 2024-05-31

(no changes; this release just promotes v0.7.2-alpha.6 to v0.7.2)

## 0.7.2-alpha.6 - 2024-05-16

### Fixed

- (cli) Hard links should now be handled correctly when they need to be exported from downloaded archives or OCI images.
- (cli) Staging a pallet now includes the download of any missing files, OCI container images, and repos required for staging.

## 0.7.2-alpha.5 - 2024-05-15

### Fixed

- (cli) Symlinks are now handled correctly when they need to be exported from downloaded archives or OCI images.

## 0.7.2-alpha.4 - 2024-05-15

### Fixed

- (cli) Allowed the entire filetree in downloaded archives and OCI images to be used as a source for export, by specifying '/' or '.' as the source path.
- (spec) Clarified that, for the `source` field of file export resources with `source-type` `http-archive` and `oci-image`, a value of `/` or `.` will be interpreted as specifying that all files in the archive or OCI image will be exported.

## 0.7.2-alpha.3 - 2024-05-15

### Added

- (spec, cli) Added a `oci-image` file source type (for file export resources) which is downloaded and cached with `[dev] plt cache-dl` and `[dev] plt cache-all` subcommands; files can be extracted from the root filesystems of the downloaded OCI container image tarballs and exported as part of the pallet's bundle when the pallet is staged. The bundle's manifest now lists the names of downloaded OCI container images.
- (cli) Added `--stage` and `--apply` flags to the `plt clone` and `plt pull` subcommands to immediately stage/apply the pallet after cloning/pulling. Note that `plt clone --force --stage` is equivalent to `plt switch`, and `plt clone --force --apply` is equivalent to `plt switch --apply`.

## 0.7.2-alpha.2 - 2024-05-13

### Added

- (cli) Added a `[dev] plt rm-repo` subcommand which removes requirements for the specified repo paths, as an inverse of the `[dev] plt add-repo` subcommand.
- (cli) Added a `[dev] plt add-depl` subcommand which adds a package deployment at the specified deployment name, for the specified package path (and optionally for the specified feature flags and enabled/disabled setting).
- (cli) Added a `[dev] plt rm-depl` subcommand which deletes the package deployment declaration(s) at the specified deployment name(s), as the inverse of the `[dev] plt add-depl` subcommand.
- (cli) Added a `[dev] plt set-depl-pkg` subcommand which modifies a package deployment at the specified deployment name, to change the deployment's package.
- (cli) Added a `[dev] plt add-depl-feat` (or `[dev] plt enable-depl-feat`) subcommand which modifies a package deployment at the specified deployment name, to enable the specified feature flags (for feature flags which are not already enabled).
- (cli) Added a `[dev] plt rm-depl-feat` (or `[dev] plt disable-depl-feat`) subcommand which modifies a package deployment at the specified deployment name, to disable the specified feature flags (for feature flags which are not already disabled), as the inverse of the `[dev] plt add-depl-feat` subcommand.
- (cli) Added a `[dev] plt set-depl-disabled` (or `[dev] plt disable-depl`) subcommand which modifies a package deployment at the specified deployment name, to disable the deployment.
- (cli) Added a `[dev] plt unset-depl` (or `[dev] plt enable-depl`) subcommand which modifies a package deployment at the specified deployment name, to enable the deployment.

## 0.7.2-alpha.1 - 2024-05-07

### Fixed

- (cli) File permissions are now preserved in the exports of `http-archive` source files extracted from `.tar`/`.tar.gz` archives.

## 0.7.2-alpha.0 - 2024-05-07

### Added

- (spec, cli) Added `source-type` and `url` fields to the file export resource, and added `http` and `http-archive` file source types; for simplicity/backwards-compatibility, by default `source-type` is assumed to be `local` (for local files) and `url` is ignored. `url` is used for `http` and `http-archive` file source types (see next changelog item).
- (cli) `http` and `http-archive` file sources (for file export resources) are downloaded and cached with `[dev] plt cache-dl` and `[dev] plt cache-all` subcommands, and downloaded files (whether downloaded directly from an HTTP(S) URL or extracted from a `.tar.gz`/`.tar` archive downloaded from an HTTP(S) URL) are now exported as part of the pallet's bundle when the pallet is staged. The bundle's manifest now lists the URLs of downloaded files/archives.

## 0.7.1 - 2024-04-29

### Added

- (cli) Added checking for validity of source paths of file exports in `[dev] plt check` and `stage check` subcommands.

## 0.7.0 - 2024-04-25

### Added

- (cli) The `dev plt add-repo` subcommand now has additional aliases with clearer names: `require-repo` and `require-repositories`.
- (cli) Added a `plt add-repo` subcommand which is just like `dev plt add-repo` (including its new `require-repo` and `require-repositories` aliases).
- (cli) By default, now the `[dev] plt add-repo` subcommand will also cache all repos required by the pallet after adding/updating a repo requirement (or multiple repo requirements). This added behavior can be disabled with a new `--no-cache-req` flag.
- (cli) By default, now the `plt clone` and `plt pull` subcommands will also cache all required repos after cloning/pulling the pallet. This added behavior can be disabled with a new `--no-cache-req` flag.
- (cli) Added a `stage unset-next` subcommand which will update the stage store so that no staged pallet bundle will be applied next.
- (cli) Now the `stage set-next` subcommand will accept an index of 0, which will update the stage store so that no staged pallet bundle will be applied next.
- (cli) Added a `stage set-next-result` subcommand which can be used on non-Docker systems (where `forklift stage apply` doesn't work) to record whether the next staged pallet bundle to be applied has been successfully applied or has failed to be applied (or to reset its state from "failed" to "pending", representing that we don't know whether it has been applied successfully or unsuccessfully). This is intended to be used by systems which need to use the files exported by the next staged pallet bundle but might encounter unrecoverable errors.

### Changed

- (Breaking change; spec) The file exports specification has changed so that only a single target path (instead of a list of target paths) can be specified per file export object; accordingly, the field has been renamed from `targets` to `target`. Additionally, if the `source` path is left empty, it is interpreted to have the same value as the `target` path.
- (Breaking change; cli) The `--parallel` flag for various subcommands has now been consolidated and moved to the top level (e.g. `forklift --parallel plt cache-img` instead of `forklift plt cache-img --parallel`). Additionally, now the flag is enabled by default (because sequential downloading of images and bringup of Docker containers is so much slower than parallel downloading/bringup); to avoid parallel execution, use `--parallel=false` (e.g. `forklift --parallel=false plt cache-img`).
- (Breaking change; cli) `plt clone` no longer deletes the `.git` directory after cloning a pallet, because the new pallet staging functionality makes it feasible to keep a long-running local pallet which can change independently of what is actually applied on a computer.

## 0.7.0-alpha.3 - 2024-04-13

### Fixed

- (cli) Subcommands under `stage` no longer require the workspace to be set via `--workspace` or `FORKLIFT_WORKSPACE` (which defaults to `$HOME`, which may be unset in systemd system services) if the path to the stage store is explicitly set via `--stage-store` or `FORKLIFT_STAGE_STORE`.

## 0.7.0-alpha.2 - 2024-04-13

### Added

- (cli) Added a `--no-cache-img` flag to all `plt switch`, `[dev] plt stage`, and `stage set-next` to enable non-root execution in setup scripts where the Docker socket can only be accessed with root permissions.

### Fixed

- (cli) Added missing a `--parallel` flag to `plt stage`.
- (cli) Fixed incorrect usage descriptions for the `--parallel` flag for `[dev] plt plan`, `[dev] plt apply`, and `stage plan`.

### Added

- (spec) Added a file export resource type as a resource which packages can provide as part of their deployments and/or feature flags.
- (cli) Added checking of conflicts between file export resources with `plt check`/`stage check`.
- (cli) When information about a package is shown (e.g. with `cache show-pkg` or `[dev] plt show-pkg`), the source and target paths of file exports are also shown.
- (cli) When information about a package deployment is shown (e.g. with `stage show-bun-depl` or `[dev] plt show-depl`), the target paths of file exports are also shown.
- (cli) When information about a staged pallet bundle is shown (e.g. with `stage show-bun`), the target paths of file exports are also shown for each package deployment.
- (cli) Information about the target paths of file exports for each package deployment in a staged pallet bundle is now recorded in the staged pallet bundle's manifest file, in a new `exports` section.
- (cli) Added a `stage locate-bun` subcommand to show the absolute file path of the specified staged pallet bundle.
- (cli) Added a global `--stage-store` string flag which, when not empty, overrides the path of the store of staged pallet bundles to an arbitrary path. When the string is empty (the default behavior), the CLI uses the sstore in the workspace specified by the global `--workspace` flag (i.e. path-of-workspace/.local/share/forklift/stages).

## 0.7.0-alpha.0 - 2024-04-10

### Added

- (cli) Added a `[dev] plt stage` subcommand to bundle and stage the pallet as the next one to be applied.
- (cli) Added a `stage ls-bun` subcommand to list staged pallet bundles.
- (cli) Added a `stage show-bun` subcommand to show info about a staged pallet bundle.
- (cli) Added a `stage show-bun-depl` subcommand to show info about package deployment of a staged pallet bundle.
- (cli) Added a `stage locate-bun-depl-pkg` subcommand to show the absolute file path of the package for a package deployment of a staged pallet bundle.
- (cli) Added a `stage add-bun-name` subcommand to assign a name to a staged pallet bundle.
- (cli) Added a `stage ls-bun-names` subcommand to list all assigned names for staged pallet bundles.
- (cli) Added a `stage rm-bun-name` subcommand to unassign a name for a staged pallet bundle.
- (cli) Added a `stage cache-img` subcommand to cache all Docker container images required by the next staged pallet bundle to be applied, and all container images required by the last successfully-applied bundle (if it is different) as a fallback in case the next staged bundle fails to be applied.
- (cli) Added `stage check` and `stage plan` subcommands which provide equivalent functionality as `[dev] plt check` and `[dev] plt plan`, but for the next staged pallet bundle to be applied.
- (cli) Added a `stage apply` subcommand which tries to apply the next staged pallet bundle and then, if that bundle could not be successfully applied, falls back to applying the last successfully-applied bundle in subsequent invocations.
- (cli) Added a `stage set-next` subcommand which changes which staged pallet bundle will be applied next (and resets `stage apply`'s tracking of whether the next staged pallet bundle to be applied has encountered a failure in the past)
- (cli) Added a `stage show` subcommand which shows a summary of the staged pallets and what will happen when on the next invocation of the `stage apply` subcommand.
- (cli) Added a `stage show-hist` subcommand which lists all staged pallet bundles which have been successfully applied in the past.
- (cli) Added a `stage rm-bun` subcommand to delete a staged pallet bundle.
- (cli) Added a `stage prune-bun` subcommand to delete all staged pallet bundle not referred to by names or by the history of successfully-applied bundles.

### Changed

- (cli) The `[dev] plt apply` and `plt switch` subcommands now automatically bundle and stage the pallet before applying it.
- (Breaking change: cli) By default, the `plt switch` subcommand no longer applies the pallet after staging it; instead, `stage apply` must be run afterwards to apply the staged pallet. The previous behavior (immediate application of the pallet) is now available through `plt switch --apply`.

## 0.6.0 - 2024-02-28

### Added

- (cli) Added a `cache add-plt` subcommand to download a pallet to the local cache, given the pallet's path and a version query. If the pallet is already in the local cache, the subcommand will complete successfully even if there is no internet connection.
- (cli) Added a `cache show-plt` subcommand to show information about a specified pallet in the local cache.
- (cli) Added a `cache ls-plt` subcommand to list all pallets in the local cache.
- (cli) Added a `cache rm-plt` subcommand to delete all pallets in the local cache.
- (cli) Added a `cache add-repo` subcommand to download a repo to the local cache, given the repo's path and a version query. If the pallet is already in the local cache, the subcommand will complete successfully even if there is no internet connection.
- (cli) Added a `[dev] plt cache-all` subcommand which just does everything in `[dev] plt cache-repo` and `[dev] plt cache-img` in a single command.
- (release) Restored builds of macOS and Windows binaries (warning: these have not been tested to work!).

### Changed

- (cli) The `plt clone` and `plt switch` subcommands now update the local pallet cache, and they initialize the local pallet from the local pallet cache. This way, version queries can still be resolved (for re-cloning or switching pallets) even without internet access, as long as the local pallet cache is up-to-date.
- (cli) The `plt clone` and `plt switch` subcommands now create local branches tracking all remote branches, and providing a branch name as the version query causes the corresponding local branch to be checked out (instead of checking out the remote branch). This makes it easier to add local commits and push/pull between the local repository and the remote repository when a branch is checked out on the local repository.
- (cli) The `[dev] plt add-repo` subcommand now updates the local pallet cache, and it runs version queries on the local pallet cache. This way, version queries can still be resolved (for re-cloning or switching pallets) even without internet access, as long as the local pallet cache is up-to-date.

### Removed

- (Breaking change: cli) Local mirrors of remote Git repos are no longer deleted and re-cloned when Git fetch operations fail on them as part of resolving version queries; such local mirrors will instead need to be manually deleted. This removal of the previous behavior is needed to prevent local mirrors from being deleted when internet access is unavailable.

## 0.5.3 - 2024-02-10

### Fixed

- (cli) When performing operations on a pallet without any external repo requirements, forklift no longer complains if you haven't cached any repos.

## 0.5.2 - 2024-02-10

### Added

- (spec) Added a "fileset" resource type for files (which can include directories).
- (cli) Added support to for a pallet to deploy packages defined in that same pallet, by referring to the package as an absolute path (rooted at the root of the pallet), if the pallet declares itself as a Forklift repo with the same path.

## 0.5.1 - 2024-02-07

- (cli) Added a `plt switch` subcommand which is the equivalent of running `plt clone --force` and then running `plt cache-repo` and then running `plt apply`. This allows a common task (switching the version of a pallet and applying it immediately) to be run with a single command, for a simpler user experience.

## 0.5.0 - 2024-01-14

### Added

- (cli) Added a `[dev] plt locate-depl-pkg` subcommand which prints the absolute filesystem path of the package deployed by the specified deployment (useful for making shell scripts in directories available for use, e.g. in systemd services).
- (cli) Added a `cache rm-repo` subcommand which only deletes cached repositories.
- (cli) Added a `cache rm-img` subcommand which only deletes unused Docker container images.
- (cli) Added a `--include-disabled` flag to the `plt cache-img` and `dev plt cache-img` subcommands to also cache images used by disabled package deployments.
- (cli) Added a `--parallel` flag to the `plt apply` and `dev plt apply` subcommands to enable parallel bringup of Docker Compose apps without any dependency relationships between them.
- (cli) Added a `--parallel` flag to the `plt cache-img` and `dev plt cache-img` subcommands to enable parallel caching of Docker container images. Speedup will depend on the host machine, but on a high-performance laptop it led to a ~40% speedup.
- (spec) Added a `nonblocking` field to service resource requirement objects in the package specification to allow a resource requirement to be ignored for the purposes of planning the order in which package deployments are to be added or modified.

### Changed

- (Breaking change: cli) Renamed the `cache rm` command to `cache rm-all`
- (Breaking change: cli) By default, now the `plt cache-img` and `dev plt cache-img` commands don't cache images used only by disabled package deployments.
- (Breaking change: cli, workspace) The default value of the `--workspace` flag has changed from `$HOME/.forklift` to `$HOME`.
- (Breaking change: workspace) The forklift repository cache has been moved from `(workspace path)/.cache` to `(workspace path)/.cache/forklift`, so that (with the default workspace location) the forklift repository cache now matches the correct default location according to the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html). For simplicity, the cache has a fixed location with respect to the workspace, instead of being wherever is specified by the `$XDG_CACHE_HOME` environment variable.
- (Breaking change: workspace) The main forklift pallet has been moved from `(workspace path)/.forklift/cache` to `(workspace path)/.local/share/forklift/pallet`, so that (with the default workspace location) the main forklift pallet cache now matches the correct default location according to the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html). For simplicity, the main pallet has a fixed location with respect to the workspace, instead of being wherever is specified by the `$XDG_DATA_HOME` environment variable.

### Fixed

- (cli) `plt clone` can now resolve a branch name as the version query, because it now treats the branch name as the name of a remote branch from the "origin" remote (since that is the only source of branches immediately after cloning).

## 0.4.0 - 2023-10-23

### Added

- The `forklift-package.yml` files now have an optional `compose-files` field in feature flags to define Compose files which should be merged into the Compose app for every package deployment which enables that feature.
- The `forklift-pallet.yml` file can now optionally specify a README file and a pallet path. When specified, those fields are displayed by the `plt show` and `dev plt show` commands.
- The `.deploy.yml` files now have a `disabled` boolean flag which specifies whether that deployment definition should be ignored (so that it is excluded from `plt plan`, `plt check`, and `plt apply` commands).
- The `show-pkg` subcommand now shows a list of Compose files associated with each package feature flag.
- The `show-depl` subcommand now shows a list of the Compose files used to define the Compose app for the deployment.
- The `show-depl` subcommand now shows more details about the Compose app specified by the deployment.
- The `forklift-repository.yml` and `forklift-pallet.yml` now have a `forklift-version` field which indicates that the repository/pallet was written assuming the semantics of a given version of Forklift, and which sets the minimum version of Forklift required to use the repository/pallet. The Forklift version of a pallet cannot be less than the Forklift version of any repo required by the pallet. The Forklift tool also checks version compatibility - an older version of the Forklift tool is incompatible with repositories/pallets with newer Forklift versions, and the Forklift tool is also sets the minimum Forklift version of any repository/pallet it is compatible with (so for example v0.4.0 of the Forklift tool is incompatible with any repositories/pallets with Forklift version below v0.4.0, due to other breaking changes made for Forklift v0.4.0).

### Changed

- (Breaking change: spec) The `definition-files` field in `forklift-package.yml` files has been renamed to `compose-files`, for unambiguity and future-proofing (so that we can add other definition types, such as for regular files rather than Docker apps).
- (Breaking change: spec) The `forklift-version-lock.yml` file now requires a `type` field which specifies whether the version lock is to be interpreted as a tagged version or as a pseudoversion. The `commit` and `timestamp` fields are now required for all types, instead of being used to determine whether the version lock is for a tagged version or a pseudoversion.
- (Breaking change: spec) The `DefinesApp` method has been removed from `PkgDeplSpec`, since now a Compose App may be defined purely by feature flags.

### Fixed

- Now the `dev plt add-repo` command correctly specifies version-locking information when locking a repo at a tagged version.

## 0.3.1 - 2023-08-24

### Removed

- Removed builds for Darwin and Windows targets, because v0.3.0 couldn't be released due to the CI workflow running out of disk space.

## 0.3.0 - 2023-08-23

### Added

- Now Git repositories providing Forklift packages can be hosted anywhere with a URL, not just on GitHub.

### Changed

- (Breaking change: spec) Now only a single Forklift repository is permitted per Git repository, and the root of the Forklift repository must be the root of the Git repository. This means that the path of the Forklift repository is just the path of the Git repository corresponding to that Forklift repository, and thus the repository definition file must be located at the root of the Git repository.
- (Breaking change: spec, cli) Renamed "Forklift pallet"/"pallet" to "Forklift repository"/"repository". All commands now use `repo` instead of `plt`. This partially reverts a change made in 0.2.0.
- (Breaking change: spec, cli) Renamed "environment"/"env" to "pallet"/"plt". All commands now use `plt` instead of `env`.
- (Breaking change: spec) Changed the name of repository definition files from `forklift-pallet.yml` to `forklift-repository.yml`. This partially reverts a change made in 0.2.0.
- (Breaking change: spec) Changed the name of the repository specification section in the repository definition file from `pallet` to `repository`. This reverts a change made in 0.2.0.
- (Breaking change: spec) Changed the name of the repository requirements directory in environments from `requirements/pallets` to `requirements/repositories`. This partially reverts a change made in 0.2.0.
- (Breaking change: workspace) Changed the name of the repository cache directory in the workspace from `cache` to `cache/repositories`. This partially reverts a change made in 0.2.0.
- (Breaking change: workspace) Changed the name of the pallet directory in the workspace from `env` to `pallet`.
- (Breaking change: cli) Renamed the `depl` subcommand to `host`.
- (Breaking change: cli) Renamed the `env` subcommand to `plt`, and the `dev env` subcommand to `dev plt`.

## 0.2.2 - 2023-08-11

### Fixed

- Updated the Makefile for the `make release` target to provide the `GITHUB_TOKEN` environment variable to the `goreleaser-cross` Docker container used for that Makefile target.

## 0.2.1 - 2023-08-11

### Fixed

- Updated the Makefile for the `make release` target to also use the `goreleaser-cross` Docker image which is used for the `make build` target.

## 0.2.0 - 2023-08-11

### Added

- Added `depl ls-con` command which either lists the containers for the specified package deployment or (if no package deployment name is specified) lists all containers.

### Changed

- (Breaking change: spec, cli) Forklift now manages Docker Compose applications instead of Docker Stacks, due to Docker Swarm Mode's lack of support for [devices](https://github.com/moby/swarmkit/issues/1244) (important for talking to hardware) and privileged containers (very useful for gradually migrating non-containerized applications into containers). Note that this causes the compiled binaries to approximately double in size, from ~20-25 MB (on linux_amd64) to ~50-60 MB, because of all the unnecessary dependencies pulled in by the `github.comdocker/compose/v2` package; similarly, the compressed archives for the binaries double in size, from ~8 MB to ~17 MB. Hopefully we can return to more reasonable uncompressed binary sizes in the future.
- (Breaking change: spec, cli) Renamed "Pallet repository" to "Forklift pallet"/"pallet". All commands now use `plt` instead of `repo`.
- (Breaking change: spec) Changed the name of pallet definition files from `pallet-repository.yml` to `forklift-pallet.yml`.
- (Breaking change: spec) Changed the name of the pallet specification section in the pallet definition file from `repository` to `pallet`.
- (Breaking change: spec) Changed the name of package definition files from `pallet-package.yml` to `forklift-package.yml`.
- (Breaking change: spec) Changed the way the Docker Compose application is specified in the `deployment` section of a Pallet package definition, from a `definition-file` field for a single string to a `definition-files` field for a list of strings.
- (Breaking change: spec) Changed the name of the pallet requirements directory in environments from `repositories` to `requirements/pallets`.
- (Breaking change: spec) Changed the name of pallet version lock files in environments from `forklift-repo.yml` to `forklift-version-lock.yml`.
- (Breaking change: workspace) Changed the name of the pallet cache directory in the workspace from `cache` to `cache/pallets`.

### Removed

- (Breaking change: cli) Removed one-letter abbreviations in all aliases.

## 0.1.10 - 2023-08-03

### Added

- `env plan`, `dev env plan`, `env apply`, and `dev env apply` now return an error (and report the problems) if there are resource conflicts or missing resource dependencies (the same problems which would be reported by `env check` and `dev env check`).
- `cache show-repo`, `env show-repo`, and `dev env show-repo` now print the Pallet repository's readme file, hard-wrapped to a max line length of 100 characters.

### Fixed

- `env plan`, `dev env plan`, `env apply`, and `dev env apply` now account for the resource dependency relationships among package deployments when planning the sequence of changes to make to the Docker host, so that a Docker stack which requires a network provided by another Docker stack won't be deployed before that other Docker stack (since such a deployment would fail).
- The `dev env add-repo` subcommand now makes any directories it needs to make in order to write repository requirement definition files to the appropriate locations.
- File path separators should no longer be obviously incorrect on Windows systems (though they may still be incorrect, since Forklift is not tested on Windows).

## 0.1.9 - 2023-07-29

### Fixed

- Fixed regression from v0.1.8 where the `dev env add-repo` would not properly print out the resolved versions of repo version queries.

## 0.1.8 - 2023-07-29

### Added

- `env check` and `dev env check` now checks resource constraints against all provided resources and resource requirements among all package deployments in the environment, and reports any identified resource constraint violations.
- `dev env` now allows specifying one or more directories containing Pallet repositories to replace any corresponding cached repositories, using the `--repo` flag (which can be specified repeatedly).
- `env plan` and `dev env plan` now show the changes which will be made by `env apply` and `dev env apply`, respectively.
- The (draft) implementation of the (draft) specification for the Pallets package management system is now available in the `/pkg/pallets` directory of this repository. Note that the specification and implementation will be changed to simplify terminology, so the API will definitely change.
- Pallet package deployment specifications can now be defined in subdirectories under the `deployments` directory of a Forklift environment, instead of having to be defined only in the `deployments` directory.

### Changed

- A major internal refactoring was done. No breaking changes are expected, but breakage is still possible.

### 0.1.7 - 2023-06-01

- `cache ls-img` can now search for locally-downloaded Docker container images matching a provided repository and tag.
- `cache ls-img` now shows the first tag of each Docker container image together with the image repository name, if the tag exists

### Fixed

- Previously, `env apply` and `dev env apply` would always explicitly attempt to pull the image needed for each Docker Stack service, leading them to fail with an error if the computer had no internet connection. Now, they will only explicitly attempt to pull the image if it is not found among the locally-downloaded Docker container images.

## 0.1.6 - 2023-05-31

### Added

- `env cache-img` and `dev env cache-img` commands are now provided to download all Docker container images ahead-of-time before applying the specified environment.
- `cache ls-img` and `cache show-img` commands are now provided to show information about the Docker container images available in the Docker environment.
- `cache rm` now also removes all unused Docker container images which had been locally downloaded.

### Changed

- (Breaking change: cli) Renamed the `env deploy` and `dev env deploy` commands to `env apply` and `dev env apply`, respectively. This is meant to make the mental model for forklift slightly more familiar to people who have used Terraform.
- (Breaking change: cli) Renamed the `env cache` and `dev env cache` commands to `env cache-repo` and `dev env cache-repo`, respectively. This disambiguates the commands for caching Pallets-related data and for caching Docker container images, while allowing them to be run separately (useful on Docker environments where root permissions are required to talk to the Docker daemon).

### Fixed

- When the `env apply` and `dev env apply` commands pull images as part of the process of deploying Docker stacks, they now pull images before creating the stack services with proper image tags, since the Docker API client pulling images without any tags.

## 0.1.5 - 2023-05-26

### Fixed

- Previously, the `dev env add` command did not correctly update the local cache's mirrors of remote Git repositories. Now it should (probably) update them correctly.

## 0.1.4 - 2023-05-24

### Added

- A `dev env add` subcommand to add one or more Pallet repositories to the development environment, and/or to update their configured versions in the development environment

### Changed

- (Breaking change: spec) Renamed the `version` field of `forklift-repo.yml` files to `base-version`.

## 0.1.3 - 2023-05-23

### Added

- An `env info` subcommand to display info about the local environment as a Git repository
- A `dev env` command with subcommands to display info about a development environment (at a user-set path), with the same subcommand structure as the `env` command

### Changed

- Changed the "info" verb in subcommands to "show".
- Standardized abbreviations and expansions of verbs for subcommands (e.g. "d"/"deploy", "ls"/"list", or "s"/"show").
- Standardized abbreviations and expansions of nouns for subcommands (e.g. "d"/"depl"/"deployment"/"deployments" or "r"/"repo"/"repository"/"repositories"). Now the longest alias of a noun-verb subcommand always makes grammatical sense (e.g. "list-repositories" instead of "list-repository", "show-repository" instead of "information-repository"), and the shortest alias of a subcommand always has a one-to-three-letter verb and a one-to-three-letter noun, and the main name of a subcommand is of intermediate length (e.g. "ls-repo", "show-pkg", "show-depl").

### Removed

- Release channels are no longer tracked for each Pallet repository within a Forklift environment, for simplicity.
- (Breaking change: spec) The `forklift-repo-lock.yml` file has been renamed to `forklift-repo.yml`, for simplicity.

### Fixed

- The `depl rm` subcommand now waits until all deleted networks actually disappear before it finishes. This is to help prevent the `env deploy` and `dev env deploy` subcommands from being run while the state of the Docker Swarm is still changing as a result of a previous `depl rm` subcommand.

## 0.1.2 - 2023-04-26

### Fixed

- Set correct file permission flags when making the forklift workspace if it doesn't already exist.

## 0.1.1 - 2023-04-26

### Added

- Handling of stacks which need to be removed as part of the `forklift env deploy` command

### Fixed

- Order of deleting resources (services, networks, secrets, configs) in the `forklift depl rm` command, so that it does not error out when one of the stacks to be deleted provides a resource (e.g. a network) used by other stacks as an external resource.

## 0.1.0 - 2023-04-25

### Added

- Basic commands for cloning and tracking a Pallet environment from a remote Git repository
- A basic command for downloading (into a local cache of Pallet repositories/packages) all Pallet repositories specified by the local Pallet environment
- A basic command with minimal functionality for deploying the local Pallet environment into the local Docker Swarm; this does not fully implement the Pallet specification (notably, all package features are always enabled)
- Basic commands for displaying information about the local Pallet environment and the local cache
- A basic command for deleting the local Pallet environment
- A basic command for deleting the local cache of Pallet repositories/packages
- A basic command for checking what Docker stacks are running in the local Docker Swarm
- A basic command with minimal functionality for deleting all stacks from the local Docker Swarm; this is not fully correct in deleting resources created by Pallet packages (for example, it can't properly delete a network created by one package which is used as an external network by other packages)
