package forklift

import (
	"bytes"
	"io/fs"

	"github.com/pkg/errors"
)

func loadFile(file fs.File) (bytes.Buffer, error) {
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(file)
	return buf, errors.Wrap(err, "couldn't load file")
}
