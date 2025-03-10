// SPDX-License-Identifier: Apache-2.0

package app

import (
	"bytes"
	"context"
	"fmt"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/container"
	carismaIO "github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/io"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	pbNode "github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/protobuf/carisma/node/v1"
	pbService "github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/protobuf/carisma/service/v1"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// Default location of the file containing the desired deployment config.
	desiredDeploymentConfigFilePath = "/opt/carisma/conf/global_desired_state.json"

	// Default location of the file containing the actual deployment config.
	actualDeploymentConfigFilePath = "/opt/carisma/conf/global_actual_state.json"

	// Delay between the messages transmitting the actual state.
	refreshRate = 5 * time.Second
)

func Run() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logging.LogInfo("Starting CARISMA orchestrator 🚀")

	cfg, err := config.New()
	logging.LogErr(err)

	if err != nil {
		return
	}

	if cfg.EnableDebugMode {
		logging.EnableDebugLogs()
	}

	var containerManager container.Manager
	if cfg.EmulateContainerRuntime {
		containerManager = container.NewDebugContainerManager(logging.DefaultDebugLevelWriter)
	} else {
		containerManager, err = container.NewDockerContainerManager(ctx)
		logging.LogErr(err)

		if err != nil {
			return
		}
	}
	defer func() {
		logging.LogErr(containerManager.Close())
	}()

	// stop potentially running Envoy instance
	err = stopEnvoy(context.Background(), containerManager, cfg)
	logging.LogErr(err)

	if cfg.EnableDiscovery {
		handleDiscovery(ctx, cfg)
	}

	cpConn, err := grpc.NewClient(
		net.JoinHostPort(
			cfg.CentralNodeHostname,
			strconv.Itoa(cfg.GRPCPort),
		),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	logging.LogErr(err)

	if err != nil {
		return
	}

	defer func() {
		logging.LogErr(cpConn.Close())
	}()

	nodeRegClient := pbNode.NewNodeRegistryServiceClient(cpConn)
	r, err := nodeRegClient.Register(
		ctx,
		&pbNode.RegisterRequest{
			Address: cfg.NodeHostname,
			Port:    int32(cfg.IngressPort),
		},
		grpc.WaitForReady(true),
	)
	logging.LogErr(err)

	if err != nil {
		return
	}

	logging.LogInfo(fmt.Sprintf("Received node ID %s during node registration", r.Id))

	err = runEnvoy(context.Background(), containerManager, cfg, r.Id)
	logging.LogErr(err)

	if err != nil {
		return
	}

	defer func() {
		err = stopEnvoy(context.Background(), containerManager, cfg)
		logging.LogErr(err)
	}()

	serviceRegClient := pbService.NewServiceRegistryServiceClient(cpConn)
	serviceRegChanClient, err := serviceRegClient.OpenChannel(
		metadata.NewOutgoingContext(
			ctx,
			metadata.Pairs(registry.HeaderNodeID, r.Id),
		),
	)
	logging.LogErr(err)

	if err != nil {
		return
	}

	nodeRegChanClient, err := nodeRegClient.OpenChannel(
		metadata.NewOutgoingContext(
			ctx,
			metadata.Pairs(registry.HeaderNodeID, r.Id),
		),
	)

	orchestrator := newOrchestrator(
		cfg,
		containerManager,
		func(bundleID string, servicePort int32) {
			err := serviceRegChanClient.Send(&pbService.ServiceAnnouncement{
				BundleId:          bundleID,
				LocalPort:         servicePort,
				RegistrationState: pbService.ServiceAnnouncement_REGISTRATION_STATE_REGISTERED,
			})
			logging.LogErr(err)
		},
		func(bundleID string, servicePort int32) {
			err := serviceRegChanClient.Send(&pbService.ServiceAnnouncement{
				BundleId:          bundleID,
				LocalPort:         servicePort,
				RegistrationState: pbService.ServiceAnnouncement_REGISTRATION_STATE_UNREGISTERED,
			})
			logging.LogErr(err)
		},
	)

	if cfg.EnableCentralMode {
		desiredDeploymentConfigFile, err := carismaIO.NewFileWatcher(desiredDeploymentConfigFilePath)
		logging.LogErr(err)

		if err != nil {
			return
		}

		defer desiredDeploymentConfigFile.Close()

		desiredDeploymentConfigFile.HandleDiff(func(a, b []byte) {
			if bytes.Equal(a, b) {
				return
			}

			dplmCfg, err := config.DeploymentConfigFromJSON(b)
			logging.LogErr(err)

			if err != nil {
				return
			}

			err = nodeRegChanClient.Send(
				&pbNode.DeploymentConfiguration{
					Json:      string(b),
					StateType: pbNode.DeploymentConfiguration_STATE_TYPE_DESIRED,
				},
			)
			logging.LogErr(err)

			err = orchestrator.process(ctx, &dplmCfg)
			logging.LogErr(err)
		})

		go desiredDeploymentConfigFile.Watch(ctx)

		go func() {
			globalActualDplmCfg := make(config.DeploymentConfig)

			for {
				msgDplmCfg, err := nodeRegChanClient.Recv()

				if err == io.EOF {
					// do not log EOF
					break
				} else if err != nil {
					logging.LogErr(err)

					break
				}

				if msgDplmCfg.StateType != pbNode.DeploymentConfiguration_STATE_TYPE_ACTUAL {
					continue
				}

				dplmCfg, err := config.DeploymentConfigFromJSON([]byte(msgDplmCfg.Json))
				logging.LogErr(err)

				if err == nil {
					for hostname, node := range dplmCfg {
						switch node.State {
						// helper state that request transmission of deployment configuration upon node startup
						case config.NodeStateStarting:
							if len(dplmCfg) == 1 {
								desiredDeploymentConfigFile.Diff(true)
							}
						// update actual deployment configuration
						case config.NodeStateRunning:
							fallthrough
						case config.NodeStateStopped:
							globalActualDplmCfg[hostname] = node
						}
					}

					prevGlobalActualDplmCfg := make(config.DeploymentConfig)

					j, err := os.ReadFile(actualDeploymentConfigFilePath)
					logging.LogErr(err)

					if err == nil {
						prevGlobalActualDplmCfg, err = config.DeploymentConfigFromJSON(j)
						logging.LogErr(err)
					}

					for hostname, node := range globalActualDplmCfg {
						if prevGlobalActualDplmCfg[hostname].State == config.NodeStateStopping && node.State == config.NodeStateRunning {
							node.State = config.NodeStateStopping
						}

						if prevGlobalActualDplmCfg[hostname].State == config.NodeStateStarting && node.State == config.NodeStateStopped {
							node.State = config.NodeStateStarting
						}

						prevGlobalActualDplmCfg[hostname] = node
					}

					j, err = prevGlobalActualDplmCfg.JSON()
					logging.LogErr(err)

					if err == nil {
						err = os.WriteFile(actualDeploymentConfigFilePath, j, 0644)
						logging.LogErr(err)
					}
				}
			}
		}()

		// initially compare the system state to the desired state
		desiredDeploymentConfigFile.Diff(false)
	}

	ticker := time.NewTicker(refreshRate)
	go func() {
		for {
			select {
			case <-ticker.C:
				currContainers, err := containerManager.Containers(ctx)
				logging.LogErr(err)

				if err != nil {
					return
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

				images := make([]string, len(currDeploymentConfig))
				for idx, img := range currDeploymentConfig {
					images[idx] = fmt.Sprintf("%s:%s", img.Name, img.Version)
				}

				dplmCfg := config.DeploymentConfig{
					cfg.NodeHostname: {
						State:  config.NodeStateRunning,
						Images: images,
					},
				}

				j, err := dplmCfg.JSON()
				logging.LogErr(err)

				if err == nil {
					err = nodeRegChanClient.Send(
						&pbNode.DeploymentConfiguration{
							Json:      string(j),
							StateType: pbNode.DeploymentConfiguration_STATE_TYPE_ACTUAL,
						},
					)
					logging.LogErr(err)
				}
			}
		}
	}()

	for {
		msgDplmCfg, err := nodeRegChanClient.Recv()
		if err == io.EOF {
			// do not log EOF
			break
		}
		if err != nil {
			logging.LogErr(err)

			break
		}

		dplmCfg, err := config.DeploymentConfigFromJSON([]byte(msgDplmCfg.Json))
		logging.LogErr(err)

		if err != nil {
			continue
		}

		err = orchestrator.process(ctx, &dplmCfg)
		logging.LogErr(err)
	}

	logging.LogInfo("Shutting down CARISMA orchestrator")
	stop()
}
