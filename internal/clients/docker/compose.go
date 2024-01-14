// Package docker provides a wrapper around Docker Compose's functionality
package docker

import (
	"context"
	"io"
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

type clientOptions struct {
	quiet     bool
	apiClient []dc.Opt
	cli       []command.DockerCliOption
	cliFlags  flags.ClientOptions
}

type ClientOption func(clientOptions) clientOptions

func WithConcurrencySafeOutput() ClientOption {
	return func(options clientOptions) clientOptions {
		options.quiet = true
		options.cli = append(options.cli, command.WithErrorStream(io.Discard))
		options.cliFlags.LogLevel = "warning"
		return options
	}
}

type Client struct {
	options clientOptions
	Client  *dc.Client
	Compose api.Service
}

func NewClient(opts ...ClientOption) (*Client, error) {
	options := clientOptions{
		apiClient: []dc.Opt{
			dc.WithHostFromEnv(),
			dc.WithAPIVersionNegotiation(),
		},
		cli: []command.DockerCliOption{
			command.WithDefaultContextStoreConfig(),
		},
		cliFlags: flags.ClientOptions{},
	}
	for _, opt := range opts {
		options = opt(options)
	}
	client, err := dc.NewClientWithOpts(options.apiClient...)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make docker client")
	}

	options.cli = append([]command.DockerCliOption{command.WithAPIClient(client)}, options.cli...)
	cli, err := command.NewDockerCli(options.cli...)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make docker cli")
	}
	if err = cli.Initialize(&options.cliFlags); err != nil {
		return nil, errors.Wrap(err, "couldn't initialize docker cli")
	}

	return &Client{
		options: options,
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

func (c *Client) DeployApp(
	ctx context.Context, app *dct.Project, waitTimeout time.Duration,
) error {
	options := api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans:        true,
			Recreate:             api.RecreateDiverged,
			RecreateDependencies: api.RecreateDiverged,
			Timeout:              &waitTimeout,
			QuietPull:            c.options.quiet,
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
