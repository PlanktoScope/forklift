package docker

import (
	"context"
	"fmt"

	"github.com/docker/compose/v2/pkg/api"
	dt "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

// docker container ls

func (c *Client) ListContainers(ctx context.Context, appName string) ([]dt.Container, error) {
	f := filters.NewArgs(appFilter(appName), oneOffFilter(false))
	if appName == "" {
		f = filters.NewArgs(oneOffFilter(false))
	}
	return c.Client.ContainerList(ctx, dt.ContainerListOptions{
		Filters: f,
		All:     true,
	})
}

func appFilter(appName string) filters.KeyValuePair {
	// This function is copied from the github.com/compose-spec/compose-go/pkg/compose package's
	// projectFilter function, which is licensed under Apache-2.0.
	return filters.Arg("label", fmt.Sprintf("%s=%s", api.ProjectLabel, appName))
}

func oneOffFilter(b bool) filters.KeyValuePair {
	// This function is copied from the github.com/compose-spec/compose-go/pkg/compose package's
	// oneOffFilter function, which is licensed under Apache-2.0.
	v := "False"
	if b {
		v = "True"
	}
	return filters.Arg("label", fmt.Sprintf("%s=%s", api.OneoffLabel, v))
}
