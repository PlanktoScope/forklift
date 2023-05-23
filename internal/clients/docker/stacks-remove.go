// Package docker simplifies docker operations
package docker

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/docker/cli/cli/compose/convert"
	dt "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/pkg/errors"
)

func (c *Client) removeServices(
	ctx context.Context, services []swarm.Service,
) (removed []string, err error) {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm package's
	// removeServices function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in a command.Cli parameter, and by returning an error.
	sort.Slice(services, func(i, j int) bool {
		return services[i].Spec.Name < services[j].Spec.Name
	})
	removed = make([]string, 0, len(services))
	for _, service := range services {
		removed = append(removed, service.Spec.Name)
		if err := c.Client.ServiceRemove(ctx, service.ID); err != nil {
			return removed, errors.Wrapf(err, "couldn't remove service %s", service.ID)
		}
	}
	return removed, nil
}

func (c *Client) removeSecrets(
	ctx context.Context, secrets []swarm.Secret,
) (removed []string, err error) {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm package's
	// removeSecrets function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in a command.Cli parameter, and by returning an error.
	sort.Slice(secrets, func(i, j int) bool {
		return secrets[i].Spec.Name < secrets[j].Spec.Name
	})
	removed = make([]string, 0, len(secrets))
	for _, secret := range secrets {
		removed = append(removed, secret.Spec.Name)
		if err := c.Client.SecretRemove(ctx, secret.ID); err != nil {
			return removed, errors.Wrapf(err, "couldn't remove secret %s", secret.ID)
		}
	}
	return removed, nil
}

func (c *Client) removeConfigs(
	ctx context.Context, configs []swarm.Config,
) (removed []string, err error) {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm package's
	// removeConfigs function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in a command.Cli parameter, and by returning an error.
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Spec.Name < configs[j].Spec.Name
	})
	removed = make([]string, 0, len(configs))
	for _, config := range configs {
		removed = append(removed, config.Spec.Name)
		if err := c.Client.ConfigRemove(ctx, config.ID); err != nil {
			return removed, errors.Wrapf(err, "couldn't remove config %s", config.ID)
		}
	}
	return removed, nil
}

func (c *Client) removeNetworks(
	ctx context.Context, networks []dt.NetworkResource,
) (removed []string, err error) {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm package's
	// removeNetworks function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in a command.Cli parameter, and by returning an error.
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})
	removed = make([]string, 0, len(networks))
	for _, network := range networks {
		removed = append(removed, network.Name)
		if err := c.Client.NetworkRemove(ctx, network.ID); err != nil {
			return removed, errors.Wrapf(err, "couldn't remove network %s", network.ID)
		}
	}
	return removed, nil
}

func (c *Client) waitForNetworkRemoval(ctx context.Context, networks []dt.NetworkResource) error {
	waitingNetworks := make(map[string]struct{})
	for _, network := range networks {
		waitingNetworks[network.Name] = struct{}{}
	}
	for {
		existingNetworks, err := c.Client.NetworkList(ctx, dt.NetworkListOptions{})
		if err != nil {
			return err
		}
		existingNetworkMap := make(map[string]struct{})
		for _, network := range existingNetworks {
			existingNetworkMap[network.Name] = struct{}{}
		}

		for name := range waitingNetworks {
			if _, ok := existingNetworkMap[name]; !ok {
				delete(waitingNetworks, name)
			}
		}
		if len(waitingNetworks) == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := ctx.Err(); err != nil {
				// Context was also canceled and it should have priority
				return err
			}
			names := make([]string, 0, len(waitingNetworks))
			for name := range waitingNetworks {
				names = append(names, name)
			}
			// TODO: emit this message to somewhere instead of printing it directly to stdout
			fmt.Printf(
				"Waiting for some networks (%s) to disappear after removal...\n", strings.Join(names, ", "),
			)
			// TODO: maybe the polling interval should be configurable?
			const pollInterval = 5
			time.Sleep(pollInterval * time.Second)
		}
	}
}

// pruneServices removes services which are no longer referenced in the source for a stack.
func (c *Client) pruneServices(
	ctx context.Context, namespace convert.Namespace, services map[string]struct{},
) (pruned []string, err error) {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm package's
	// pruneServices function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in a command.Cli parameter, and by returning an error.
	oldServices, err := c.getStackServices(ctx, namespace.Name())
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't list services of stack %s", namespace.Name())
	}

	servicesToPrune := make([]swarm.Service, 0, len(oldServices))
	for _, service := range oldServices {
		if _, ok := services[namespace.Descope(service.Spec.Name)]; !ok {
			servicesToPrune = append(servicesToPrune, service)
		}
	}
	pruned, err = c.removeServices(ctx, servicesToPrune)
	return pruned, errors.Wrap(err, "couldn't fully prune services")
}

func (c *Client) identifyStackResources(
	ctx context.Context, name string,
) (
	services []swarm.Service, secrets []swarm.Secret,
	configs []swarm.Config, networks []dt.NetworkResource, err error,
) {
	if services, err = c.getStackServices(ctx, name); err != nil {
		return nil, nil, nil, nil, err
	}
	if versions.GreaterThanOrEqualTo(c.Client.ClientVersion(), "1.25") {
		if secrets, err = c.getStackSecrets(ctx, name); err != nil {
			return nil, nil, nil, nil, err
		}
	}
	if versions.GreaterThanOrEqualTo(c.Client.ClientVersion(), "1.30") {
		if configs, err = c.getStackConfigs(ctx, name); err != nil {
			return nil, nil, nil, nil, err
		}
	}
	if networks, err = c.getStackNetworks(ctx, name); err != nil {
		return nil, nil, nil, nil, err
	}

	if len(services)+len(secrets)+len(configs)+len(networks) == 0 {
		return nil, nil, nil, nil, errors.Errorf("no resources found in stack %s", name)
	}
	return services, secrets, configs, networks, nil
}

func (c *Client) RemoveStacks(ctx context.Context, names []string) error {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm
	// package's RunRemove function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in command.Cli and options.Remove parameters.
	allServices := make(map[string][]swarm.Service)
	allSecrets := make(map[string][]swarm.Secret)
	allConfigs := make(map[string][]swarm.Config)
	allNetworks := make(map[string][]dt.NetworkResource)
	for _, name := range names {
		var err error
		if allServices[name], allSecrets[name],
			allConfigs[name], allNetworks[name], err = c.identifyStackResources(ctx, name); err != nil {
			return errors.Wrapf(err, "couldn't identify resources in stack %s", name)
		}
	}

	// First remove all services, as some services may depend on secrets/configs/networks provided by
	// other stacks
	for _, name := range names {
		if removed, err := c.removeServices(ctx, allServices[name]); err != nil {
			return errors.Wrapf(
				err, "only partially removed services from stack %s (removed services %+v)", name, removed,
			)
		}
	}
	for _, name := range names {
		if removed, err := c.removeSecrets(ctx, allSecrets[name]); err != nil {
			return errors.Wrapf(
				err, "only partially removed secrets from stack %s (removed secrets %+v)", name, removed,
			)
		}
	}
	for _, name := range names {
		if removed, err := c.removeConfigs(ctx, allConfigs[name]); err != nil {
			return errors.Wrapf(
				err, "only partially removed configs from stack %s (removed configs %+v)", name, removed,
			)
		}
	}
	removedNetworks := make([]dt.NetworkResource, 0)
	for _, name := range names {
		removed, err := c.removeNetworks(ctx, allNetworks[name])
		if err != nil {
			return errors.Wrapf(
				err, "only partially removed networks from stack %s (removed networks %+v)", name, removed,
			)
		}
		removedNetworks = append(removedNetworks, allNetworks[name]...)
	}
	return c.waitForNetworkRemoval(ctx, removedNetworks)
}
