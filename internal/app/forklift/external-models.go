package forklift

import (
	"io/fs"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type ExternalRepo struct {
	Repo pallets.FSRepo
	FS   fs.FS
}
