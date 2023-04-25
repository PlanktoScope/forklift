// Package docker simplifies docker operations
package docker

import (
	"context"
	"sort"

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

func (c *Client) RemoveStack(ctx context.Context, name string) error {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm
	// package's RunRemove function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in a command.Cli parameter, and by only removing a single stack.
	services, err := c.getStackServices(ctx, name)
	if err != nil {
		return err
	}
	networks, err := c.getStackNetworks(ctx, name)
	if err != nil {
		return err
	}
	var secrets []swarm.Secret
	if versions.GreaterThanOrEqualTo(c.Client.ClientVersion(), "1.25") {
		secrets, err = c.getStackSecrets(ctx, name)
		if err != nil {
			return err
		}
	}
	var configs []swarm.Config
	if versions.GreaterThanOrEqualTo(c.Client.ClientVersion(), "1.30") {
		configs, err = c.getStackConfigs(ctx, name)
		if err != nil {
			return err
		}
	}

	if len(services)+len(networks)+len(secrets)+len(configs) == 0 {
		return errors.Errorf("nothing found in stack %s", name)
	}

	if removed, err := c.removeSecrets(ctx, secrets); err != nil {
		return errors.Wrapf(
			err, "only partially removed secrets from stack %s (removed secrets %+v)", name, removed,
		)
	}
	if removed, err := c.removeConfigs(ctx, configs); err != nil {
		return errors.Wrapf(
			err, "only partially removed configs from stack %s (removed configs %+v)", name, removed,
		)
	}
	if removed, err := c.removeNetworks(ctx, networks); err != nil {
		return errors.Wrapf(
			err, "only partially removed networks from stack %s (removed networks %+v)", name, removed,
		)
	}
	if removed, err := c.removeServices(ctx, services); err != nil {
		return errors.Wrapf(
			err, "only partially removed services from stack %s (removed services %+v)", name, removed,
		)
	}

	return nil
}
