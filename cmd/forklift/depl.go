package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/clients/docker"
)

// CLI

var deplCmd = &cli.Command{
	Name:    "depl",
	Aliases: []string{"d", "deployments"},
	Usage:   "Manages active Pallet package deployments in the local environment",
	Subcommands: []*cli.Command{
		{
			Name:     "ls-stack",
			Category: "Query the active deployment",
			Aliases:  []string{"ls-s", "list-stacks"},
			Usage:    "Lists running Docker stacks",
			Action:   deplLsStackAction,
		},
		{
			Name:     "rm",
			Category: "Modify the active deployment",
			Aliases:  []string{"remove"},
			Usage:    "Removes all Docker stacks",
			Action:   deplRmAction,
		},
	},
}

// ls-stack

func deplLsStackAction(c *cli.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	stacks, err := client.ListStacks(context.Background())
	if err != nil {
		return errors.Wrap(err, "couldn't list running Docker stacks")
	}
	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i].Name < stacks[j].Name
	})
	for _, stack := range stacks {
		fmt.Printf("%s\n", stack.Name)
	}
	return nil
}

// rm

func deplRmAction(c *cli.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	stacks, err := client.ListStacks(context.Background())
	if err != nil {
		return errors.Wrap(err, "couldn't list running Docker stacks")
	}
	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i].Name < stacks[j].Name
	})

	stackNames := make([]string, 0, len(stacks))
	for _, stack := range stacks {
		stackNames = append(stackNames, stack.Name)
	}
	if err := client.RemoveStacks(context.Background(), stackNames); err != nil {
		return errors.Wrap(
			err, "couldn't fully remove all stacks (remaining resources must be manually removed)",
		)
	}
	fmt.Println("Done!")
	return nil
}
