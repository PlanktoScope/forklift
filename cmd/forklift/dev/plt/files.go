package plt

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-file

func lsFileAction(c *cli.Context) error {
	plt, _, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	filter := c.Args().First()
	if filter == "" {
		// Exclude hidden directories such as `.git`
		filter = "{*,[^.]*/**}"
	}
	paths, err := fcli.ListPalletFiles(plt, filter)
	if err != nil {
		return err
	}
	for _, p := range paths {
		fmt.Println(p)
	}
	return nil
}

// locate-file

func locateFileAction(c *cli.Context) error {
	plt, _, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	location, err := fcli.GetFileLocation(plt, c.Args().First())
	if err != nil {
		return err
	}
	fmt.Println(location)
	return nil
}

// show-file

func showFileAction(c *cli.Context) error {
	plt, _, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintFile(os.Stdout, plt, c.Args().First())
}

// edit-file

func editFileAction(c *cli.Context) error {
	plt, _, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	return fcli.EditFileWithCOW(plt, c.Args().First(), c.String("editor"))
}

// del-file

func delFileAction(c *cli.Context) error {
	plt, _, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	return fcli.RemoveFile(0, plt, c.Args().First())
}
