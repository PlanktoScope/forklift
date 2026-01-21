package host

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/forklift-run/forklift/internal/clients/docker"
)

// ls-app

func lsAppAction(c *cli.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	apps, err := client.ListApps(context.Background())
	if err != nil {
		return errors.Wrap(err, "couldn't list running Docker Compose apps")
	}
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})
	for _, app := range apps {
		fmt.Printf("%+v\n", app.Name)
	}
	return nil
}

// ls-con

func lsConAction(c *cli.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	deplName := c.Args().Get(0)
	deplName = strings.ReplaceAll(deplName, "/", "_")
	containers, err := client.ListContainers(context.Background(), deplName)
	if err != nil {
		return errors.Wrapf(err, "couldn't list Docker containers for package deployment %s", deplName)
	}
	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Names[0] < containers[j].Names[0]
	})
	for _, container := range containers {
		fmt.Printf("%s\n", strings.TrimPrefix(container.Names[0], "/"))
	}
	return nil
}

// del

func delAction(c *cli.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	apps, err := client.ListApps(context.Background())
	if err != nil {
		return errors.Wrap(err, "couldn't list running Docker Compose apps")
	}
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})
	// FIXME: instead of sorting alphabetically, we must sort by dependencies - preferably using
	// Compose's dependency graph functionality

	names := make([]string, 0, len(apps))
	for _, app := range apps {
		names = append(names, app.Name)
	}
	if err := client.RemoveApps(context.Background(), names); err != nil {
		return errors.Wrap(
			err, "couldn't fully remove all apps (remaining resources must be manually removed)",
		)
	}
	fmt.Fprintln(os.Stderr, "Done!")
	return nil
}
