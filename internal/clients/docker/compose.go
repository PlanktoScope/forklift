// Package docker provides a wrapper around Docker Compose's functionality
package docker

import (
	"context"
	"time"

	dct "github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	dc "github.com/docker/docker/client"
	"github.com/pkg/errors"
)

// Client

type Client struct {
	Client  *dc.Client
	Compose api.Service
}

func NewClient() (*Client, error) {
	client, err := dc.NewClientWithOpts(
		dc.WithHostFromEnv(),
		dc.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make docker client")
	}

	cli, err := command.NewDockerCli(
		command.WithAPIClient(client),
		command.WithDefaultContextStoreConfig(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make docker cli")
	}
	clientOptions := flags.ClientOptions{}
	if err = cli.Initialize(&clientOptions); err != nil {
		return nil, errors.Wrap(err, "couldn't initialize docker cli")
	}

	return &Client{
		Client:  client,
		Compose: compose.NewComposeService(cli),
	}, nil
}

// docker compose ls

func (c *Client) ListApps(ctx context.Context) ([]api.Stack, error) {
	return c.Compose.List(ctx, api.ListOptions{
		All: true,
	})
}

// docker compose up

func (c *Client) DeployApp(ctx context.Context, app *dct.Project, waitTimeout time.Duration) error {
	options := api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans:        true,
			Recreate:             api.RecreateDiverged,
			RecreateDependencies: api.RecreateDiverged,
			Timeout:              &waitTimeout,
		},
		Start: api.StartOptions{
			Project:     app,
			Wait:        true,
			WaitTimeout: waitTimeout,
		},
	}
	return c.Compose.Up(ctx, app, options)
	// FIXME: Up doesn't seem to prune networks which are no longer needed by the app!
}

// docker compose down

func (c *Client) RemoveApps(ctx context.Context, names []string) error {
	for _, name := range names {
		if err := c.Compose.Down(ctx, name, api.DownOptions{
			RemoveOrphans: true,
		}); err != nil {
			return err
		}
	}
	return nil
}
