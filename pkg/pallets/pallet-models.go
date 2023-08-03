// Package pallets implements the specification of pallets and packages for the Forklift package
// management system.
package pallets

// A FSPallet is a Forklift pallet stored at the root of a [fs.FS] filesystem.
type FSPallet struct {
	// Pallet is the Forklift pallet at the root of the filesystem.
	Pallet
	// FS is a filesystem which contains the pallet's contents.
	FS PathedFS
}

// A Pallet is a collection of Forklift packages which are tested, released, distributed, and
// upgraded together.
type Pallet struct {
	// VCSRepoPath is the path of the VCS repository path which provides the pallet.
	VCSRepoPath string
	// Subdir is the path of the pallet within the VCS repository which provides the pallet.
	Subdir string
	// Def is the definition of the pallet.
	Def PalletDef
	// Version is the version or pseudoversion of the pallet.
	Version string
}

// PalletDefFile is the name of the file defining each pallet.
const PalletDefFile = "forklift-pallet.yml"

// A PalletDef defines a pallet.
type PalletDef struct {
	// Pallet defines the basic metadata for the pallet.
	Pallet PalletSpec `yaml:"pallet"`
}

// PalletSpec defines the basic metadata for a pallet.
type PalletSpec struct {
	// Path is the pallet path, which acts as the canonical name for the pallet. Typically, it
	// consists of a VCS repository root path followed by either a subdirectory or by nothing at all.
	Path string `yaml:"path"`
	// Description is a short description of the pallet to be shown to users.
	Description string `yaml:"description"`
	// ReadmeFile is the name of a readme file to be shown to users.
	ReadmeFile string `yaml:"readme-file"`
}
