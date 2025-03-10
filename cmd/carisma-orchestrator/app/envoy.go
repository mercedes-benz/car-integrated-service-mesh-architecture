// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"github.com/docker/go-connections/nat"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/container"
	"strconv"
)

const (
	// Envoy container image name.
	envoyContainerImageName = "envoyproxy/envoy:distroless-v1.27-latest"
)

func runEnvoy(ctx context.Context, containerManager container.Manager, cfg *config.Config, nodeID string) error {
	envoyBootstrapCfg := cfg.EnvoyBootstrapConfigWithNodeID(nodeID)

	portMap := make(map[nat.Port][]nat.PortBinding, 3)
	portMap[nat.Port(strconv.Itoa(cfg.IngressPort))] = []nat.PortBinding{
		{HostPort: strconv.Itoa(cfg.IngressPort)},
	}
	portMap[nat.Port(strconv.Itoa(cfg.EgressPort))] = []nat.PortBinding{
		{HostPort: strconv.Itoa(cfg.EgressPort)},
	}
	portMap[nat.Port(strconv.Itoa(cfg.AdminPort))] = []nat.PortBinding{
		{HostPort: strconv.Itoa(cfg.AdminPort)},
	}

	imgName, err := container.ParseFQIN(envoyContainerImageName, cfg.DefaultContainerRegistryDomain)
	if err != nil {
		return err
	}

	_, _, err = containerManager.PullImageAndCreateContainer(
		ctx,
		imgName,
		[]string{
			"--config-yaml",
			envoyBootstrapCfg,
		},
		portMap,
		false,
	)
	if err != nil {
		return err
	}

	return nil
}

func stopEnvoy(ctx context.Context, containerManager container.Manager, cfg *config.Config) error {
	imgName, err := container.ParseFQIN(envoyContainerImageName, cfg.DefaultContainerRegistryDomain)
	if err != nil {
		return err
	}

	_, _, err = containerManager.RemoveImageAndContainer(ctx, imgName, false)

	return err
}
