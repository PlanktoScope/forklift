package stage

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// ls-bundle

func lsBunAction(c *cli.Context) error {
	store, err := getStageStore(c.String("workspace"), false)
	if err != nil {
		return err
	}
	if !store.Exists() {
		return errMissingStore
	}

	indices, err := store.List()
	if err != nil {
		return err
	}
	for _, index := range indices {
		fmt.Println(index)
	}
	return nil
}
