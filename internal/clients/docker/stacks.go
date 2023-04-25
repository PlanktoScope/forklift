// Package docker simplifies docker operations
package docker

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/compose/convert"
	dt "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	dc "github.com/docker/docker/client"
	"github.com/pkg/errors"
)

// Stack

type Stack struct {
	Name     string
	Services []swarm.Service
}

func getStackFilter(namespace string) filters.Args {
	// This function is copied from the github.com/docker/cli/cli/command/stack/swarm
	// package's getStackFilter function, which is licensed under Apache-2.0.
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("%s=%s", convert.LabelNamespace, namespace))
	return filter
}

// Client

type Client struct {
	Client *dc.Client
}

func NewClient() (*Client, error) {
	client, err := dc.NewClientWithOpts(
		dc.WithHostFromEnv(),
		dc.WithAPIVersionNegotiation(),
	)
	return &Client{
		Client: client,
	}, err
}

func (c *Client) getStackServices(ctx context.Context, stackName string) ([]swarm.Service, error) {
	// This function is copied from the github.com/docker/cli/cli/command/stack/swarm
	// package's getStackServices function, which is licensed under Apache-2.0.
	return c.Client.ServiceList(ctx, dt.ServiceListOptions{
		Filters: getStackFilter(stackName),
	})
}

func (c *Client) getStackNetworks(
	ctx context.Context, stackName string,
) ([]dt.NetworkResource, error) {
	// This function is copied from the github.com/docker/cli/cli/command/stack/swarm
	// package's getStackNetworks function, which is licensed under Apache-2.0.
	return c.Client.NetworkList(ctx, dt.NetworkListOptions{
		Filters: getStackFilter(stackName),
	})
}

func (c *Client) getStackSecrets(ctx context.Context, stackName string) ([]swarm.Secret, error) {
	// This function is copied from the github.com/docker/cli/cli/command/stack/swarm
	// package's getStackSecrets function, which is licensed under Apache-2.0.
	return c.Client.SecretList(ctx, dt.SecretListOptions{
		Filters: getStackFilter(stackName),
	})
}

func (c *Client) getStackConfigs(ctx context.Context, stackName string) ([]swarm.Config, error) {
	// This function is copied from the github.com/docker/cli/cli/command/stack/swarm
	// package's getStackConfigs function, which is licensed under Apache-2.0.
	return c.Client.ConfigList(ctx, dt.ConfigListOptions{
		Filters: getStackFilter(stackName),
	})
}

func (c *Client) ListStacks(ctx context.Context) ([]Stack, error) {
	swarmServices, err := c.Client.ServiceList(ctx, dt.ServiceListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't list Docker Swarm stacks")
	}
	stackNames := make([]string, 0, len(swarmServices))
	stacks := make(map[string]Stack)
	for _, swarmService := range swarmServices {
		name, ok := swarmService.Spec.Labels[convert.LabelNamespace]
		if !ok {
			return nil, errors.Errorf(
				"couldn't determine stack name from label %s for stack %s",
				convert.LabelNamespace, swarmService.ID,
			)
		}
		stack, ok := stacks[name]
		if !ok {
			stack.Name = name
			stackNames = append(stackNames, name)
		}
		stack.Services = append(stack.Services, swarmService)
		stacks[name] = stack
	}

	orderedStacks := make([]Stack, 0, len(stackNames))
	for _, name := range stackNames {
		orderedStacks = append(orderedStacks, stacks[name])
	}
	return orderedStacks, nil
}

// checkDaemonIsSwarmManager does an Info API call to verify that the daemon is a swarm manager.
// This is necessary because we must create networks before we create services, but the API call for
// creating a network does not return a proper status code when it can't create a network in the
// "global" scope.
func (c *Client) checkDaemonIsSwarmManager(ctx context.Context) error {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm package's
	// checkDaemonIsSwarmManager function, which is licensed under Apache-2.0. This function was
	// changed by removing the need to pass in a command.Cli parameter.
	info, err := c.Client.Info(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't get information about Docker server")
	}
	if !info.Swarm.ControlAvailable {
		return errors.New("this node is not a Docker Swarm manager, first run `docker swarm init`")
	}
	return nil
}
