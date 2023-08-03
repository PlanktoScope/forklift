package env

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-pallets

func cachePltAction(c *cli.Context) error {
	env, cache, err := processFullBaseArgs(c, false)
	if err != nil {
		return err
	}

	fmt.Println("Downloading pallets specified by the local environment...")
	changed, err := fcli.DownloadPallets(0, env, cache)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Done! No further actions are needed at this time.")
		return nil
	}
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift env apply`.")
	return nil
}

// ls-pallets

func lsPltAction(c *cli.Context) error {
	env, err := getEnv(c.String("workspace"))
	if err != nil {
		return err
	}

	return fcli.PrintEnvPallets(0, env)
}

// show-plt

func showPltAction(c *cli.Context) error {
	env, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	palletPath := c.Args().First()
	return fcli.PrintPalletInfo(0, env, cache, palletPath)
}
