package forklift

import (
	"io/fs"
)

type ExternalRepo struct {
	Repo CachedRepo
	FS   fs.FS
}
