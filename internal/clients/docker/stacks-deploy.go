// Package docker simplifies docker operations
package docker

import (
	"context"

	"github.com/docker/cli/cli/compose/convert"
	dct "github.com/docker/cli/cli/compose/types"
	"github.com/docker/cli/cli/streams"
	dt "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/swarm"
	dc "github.com/docker/docker/client"
	"github.com/pkg/errors"
)

func getServicesDeclaredNetworks(serviceConfigs []dct.ServiceConfig) map[string]struct{} {
	// This function is copied from the github.com/docker/cli/cli/command/stack/swarm
	// package's getServicesDeclaredNetworks function, which is licensed under Apache-2.0.
	serviceNetworks := make(map[string]struct{})
	for _, serviceConfig := range serviceConfigs {
		if len(serviceConfig.Networks) == 0 {
			serviceNetworks["default"] = struct{}{}
			continue
		}
		for network := range serviceConfig.Networks {
			serviceNetworks[network] = struct{}{}
		}
	}
	return serviceNetworks
}

func (c *Client) validateExternalNetworks(ctx context.Context, externalNetworks []string) error {
	// This function is copied from the github.com/docker/cli/cli/command/stack/swarm
	// package's validateExternalNetworks function, which is licensed under Apache-2.0.
	for _, networkName := range externalNetworks {
		if !container.NetworkMode(networkName).IsUserDefined() {
			// Networks that are not user defined always exist on all nodes as local-scoped networks, so
			// there's no need to inspect them.
			continue
		}
		network, err := c.Client.NetworkInspect(ctx, networkName, dt.NetworkInspectOptions{})
		switch {
		case dc.IsErrNotFound(err):
			return errors.Errorf(
				"network %q is declared as external, but could not be found; you need to create a "+
					"swarm-scoped network before the stack is deployed",
				networkName,
			)
		case err != nil:
			return err
		case network.Scope != "swarm":
			return errors.Errorf(
				"network %q is declared as external, but it's in the wrong scope: %q instead of \"swarm\"",
				networkName, network.Scope,
			)
		}
	}
	return nil
}

const defaultNetworkDriver = "overlay"

func (c *Client) createNetworks(
	ctx context.Context, namespace convert.Namespace, networks map[string]dt.NetworkCreate,
) (created []string, err error) {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm package's
	// createNetworks function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in a command.Cli parameter.
	existingNetworks, err := c.Client.NetworkList(ctx, dt.NetworkListOptions{
		Filters: getStackFilter(namespace.Name()),
	})
	if err != nil {
		return nil, err
	}

	existingNetworkMap := make(map[string]dt.NetworkResource)
	for _, network := range existingNetworks {
		existingNetworkMap[network.Name] = network
	}

	created = make([]string, 0, len(networks))
	for name, createOpts := range networks {
		if _, ok := existingNetworkMap[name]; ok {
			continue
		}
		if createOpts.Driver == "" {
			createOpts.Driver = defaultNetworkDriver
		}
		if _, err := c.Client.NetworkCreate(ctx, name, createOpts); err != nil {
			return created, errors.Wrapf(err, "failed to create network %s", name)
		}
		created = append(created, name)
	}
	return created, nil
}

func (c *Client) deployServices(
	ctx context.Context, services map[string]swarm.ServiceSpec, namespace convert.Namespace,
) (added, updated []string, err error) {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm package's
	// deployServices function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in a command.Cli parameter, and removing the sendAuth and
	// resolveImage parameters.
	existingServices, err := c.getStackServices(ctx, namespace.Name())
	if err != nil {
		return nil, nil, err
	}

	existingServiceMap := make(map[string]swarm.Service)
	for _, service := range existingServices {
		existingServiceMap[service.Spec.Name] = service
	}

	added = make([]string, 0, len(services))
	updated = make([]string, 0, len(services))
	for internalName, serviceSpec := range services {
		name := namespace.Scope(internalName)
		if service, ok := existingServiceMap[name]; ok {
			updateOpts := dt.ServiceUpdateOptions{
				QueryRegistry: true,
			}

			// Preserve existing ForceUpdate value so that tasks are not re-deployed if not updated.
			serviceSpec.TaskTemplate.ForceUpdate = service.Spec.TaskTemplate.ForceUpdate

			_, err := c.Client.ServiceUpdate(
				ctx, service.ID, service.Version, serviceSpec, updateOpts,
			)
			if err != nil {
				return added, updated, errors.Wrapf(err, "failed to update service %s", name)
			}
			// TODO: handle warnings

			updated = append(updated, name)
		} else {
			createOpts := dt.ServiceCreateOptions{
				QueryRegistry: true,
			}

			if _, err := c.Client.ServiceCreate(ctx, serviceSpec, createOpts); err != nil {
				return added, updated, errors.Wrapf(err, "failed to create service %s", name)
			}
			added = append(added, name)
		}
	}
	return added, updated, nil
}

func (c *Client) pullServiceImages(
	ctx context.Context, services []dct.ServiceConfig, outStream *streams.Out,
) error {
	orderedImages := make([]string, 0, len(services))
	images := make(map[string]struct{})
	for _, service := range services {
		if _, ok := images[service.Image]; !ok {
			images[service.Image] = struct{}{}
			orderedImages = append(orderedImages, service.Image)
		}
	}

	for _, image := range orderedImages {
		if _, err := c.PullImage(ctx, image, outStream); err != nil {
			return errors.Wrapf(err, "couldn't download %s", image)
		}
	}
	return nil
}

func (c *Client) DeployStack(
	ctx context.Context, name string, config *dct.Config, outStream *streams.Out,
) error {
	// This function is adapted from the github.com/docker/cli/cli/command/stack/swarm package's
	// deployCompose function, which is licensed under Apache-2.0. This function was changed by
	// removing the need to pass in a command.Cli parameter, and by returning an error, and by
	// pulling all required images first in order to associate container images with tags.
	if err := c.checkDaemonIsSwarmManager(ctx); err != nil {
		return err
	}
	namespace := convert.NewNamespace(name)
	if err := c.pullServiceImages(ctx, config.Services, outStream); err != nil {
		return errors.Wrap(err, "couldn't pull container images for stack")
	}

	servicesMap := make(map[string]struct{})
	for _, service := range config.Services {
		servicesMap[service.Name] = struct{}{}
	}
	if pruned, err := c.pruneServices(ctx, namespace, servicesMap); err != nil {
		return errors.Wrapf(err, "only pruned some services (%+v)", pruned)
	}

	serviceNetworks := getServicesDeclaredNetworks(config.Services)
	networks, externalNetworks := convert.Networks(namespace, config.Networks, serviceNetworks)
	if err := c.validateExternalNetworks(ctx, externalNetworks); err != nil {
		return err
	}
	if created, err := c.createNetworks(ctx, namespace, networks); err != nil {
		return errors.Wrapf(err, "only created some networks (%+v)", created)
	}

	// TODO: implement
	// secrets, err := convert.Secrets(namespace, config.Secrets)
	// if err != nil {
	// 	return err
	// }
	// if err := c.createSecrets(ctx, secrets); err != nil {
	// 	return err
	// }

	// TODO: implement
	// configs, err := convert.Configs(namespace, config.Configs)
	// if err != nil {
	// 	return err
	// }
	// if err := c.createConfigs(ctx, configs); err != nil {
	// 	return err
	// }

	services, err := convert.Services(namespace, config, c.Client)
	if err != nil {
		return err
	}
	added, updated, err := c.deployServices(ctx, services, namespace)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't deploy all services (added %+v; updated %+v)", added, updated,
		)
	}
	return nil
}
