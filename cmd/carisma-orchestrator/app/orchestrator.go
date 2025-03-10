// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/container"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"strings"
)

type regHandler func(string, int32)

const (
	// Name prefix of containers that are not managed by the CARISMA orchestrator.
	unmanagedContainerNamePrefix = "/carisma-keep-"
)

func diff(a, b []container.Image) []container.Image {
	memory := make(map[string]string, len(a))
	for _, i := range a {
		memory[i.Name] = i.Version
	}

	var missing []container.Image
	for _, i := range b {
		if version, found := memory[i.Name]; !found || version != i.Version {
			missing = append(missing, i)
		}
	}

	return missing
}

type orchestrator struct {
	cfg    *config.Config
	cntMgr container.Manager
	hReg   regHandler
	hUnreg regHandler
}

func newOrchestrator(cfg *config.Config, cntMgr container.Manager, hReg regHandler, hUnreg regHandler) *orchestrator {
	return &orchestrator{
		cfg:    cfg,
		cntMgr: cntMgr,
		hReg:   hReg,
		hUnreg: hUnreg,
	}
}

func (o *orchestrator) process(ctx context.Context, dplmCfg *config.DeploymentConfig) error {
	if node, ok := (*dplmCfg)[o.cfg.NodeHostname]; ok {
		currContainers, err := o.cntMgr.Containers(ctx)

		if err != nil {
			return err
		}

		// filter container slice
		idx := 0
		for _, c := range currContainers {
			if !strings.HasPrefix(c.FirstName, unmanagedContainerNamePrefix) && strings.HasPrefix(c.Status, "Up") {
				currContainers[idx] = c
				idx++
			}
		}
		currContainers = currContainers[:idx]

		currDeploymentConfig := container.ExtractImageList(currContainers)

		newDeploymentConfig := make([]container.Image, len(node.Images))
		for idx, img := range node.Images {
			newDeploymentConfig[idx] = container.ParseImageName(img)
		}

		// We need to turn the user supplied image names into fully qualified
		// image names for comparison with the current deployment configuration.
		for idx, i := range newDeploymentConfig {
			if fqin, err := container.ParseFQIN(i.Name, o.cfg.DefaultContainerRegistryDomain); err != nil {
				return err
			} else {
				newDeploymentConfig[idx].Name = fqin
			}
		}

		removedImages := diff(newDeploymentConfig, currDeploymentConfig)
		for _, i := range removedImages {
			// do not remove communication middleware
			if strings.HasSuffix(i.Name, container.ParseImageName(envoyContainerImageName).Name) {
				continue
			}

			bundleID, servicePort, err := o.cntMgr.RemoveImageAndContainer(
				ctx,
				fmt.Sprintf("%s:%s", i.Name, i.Version),
				true,
			)

			if err != nil {
				// Do not abort execution here, but still dump the error.
				logging.DefaultLogger.Error().Err(err).
					Str("image identifier", fmt.Sprintf("%s:%s", i.Name, i.Version)).
					Msg("could not remove container image, retrying without bundle descriptor")

				_, _, err = o.cntMgr.RemoveImageAndContainer(
					ctx,
					fmt.Sprintf("%s:%s", i.Name, i.Version),
					false,
				)

				if err != nil {
					// Do not abort execution here, but still dump the error.
					logging.DefaultLogger.Error().Err(err).
						Str("image identifier", fmt.Sprintf("%s:%s", i.Name, i.Version)).
						Msg("could not remove container image, still unsuccessful")
				}

				continue
			}

			o.hUnreg(bundleID, servicePort)
		}

		newImages := diff(currDeploymentConfig, newDeploymentConfig)
		for _, i := range newImages {
			bundleID, servicePort, err := o.cntMgr.PullImageAndCreateContainer(
				ctx,
				fmt.Sprintf("%s:%s", i.Name, i.Version),
				nil,
				nil,
				true,
			)

			if err != nil {
				// Do not abort execution here, but still dump the error.
				logging.DefaultLogger.Error().Err(err).
					Str("image identifier", fmt.Sprintf("%s:%s", i.Name, i.Version)).
					Msg("could not install container image")

				continue
			}

			o.hReg(bundleID, servicePort)
		}
	}

	return nil
}
