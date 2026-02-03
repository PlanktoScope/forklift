package docker

import (
	"context"
	"fmt"

	"github.com/docker/compose/v5/pkg/api"
	"github.com/moby/moby/client"
)

// docker container ls

func (c *Client) ListContainers(
	ctx context.Context, appName string,
) (client.ContainerListResult, error) {
	f := make(client.Filters).Add("label", fmt.Sprintf("%s=%s", api.OneoffLabel, "False"))
	if appName != "" {
		f = f.Add("label", fmt.Sprintf("%s=%s", api.ProjectLabel, appName))
	}
	return c.Client.ContainerList(ctx, client.ContainerListOptions{
		Filters: f,
		All:     true,
	})
}
