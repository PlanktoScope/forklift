package forklift

import (
	"os"
)

func (c *FSCache) Exists() bool {
	return Exists(c.FS.Path())
}

func (c *FSCache) Remove() error {
	return os.RemoveAll(c.FS.Path())
}
