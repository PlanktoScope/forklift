package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/clients/docker"
)

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

	for _, stack := range stacks {
		fmt.Printf("Removing %s...\n", stack.Name)
		if err := client.RemoveStack(context.Background(), stack.Name); err != nil {
			return errors.Wrapf(err, "couldn't fully remove stack %s", stack.Name)
		}
	}
	return nil
}
