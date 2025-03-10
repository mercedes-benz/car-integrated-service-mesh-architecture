// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/cmd/carisma-control-plane/app/xds"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	pbNode "github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/protobuf/carisma/node/v1"
	pbService "github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/protobuf/carisma/service/v1"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"os/signal"
	"strconv"
	"syscall"
)

func Run() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logging.DefaultLogger.Info().Msg("Starting CARISMA Control Plane 🚀")

	cfg, err := config.New()
	if err != nil {
		logging.DefaultLogger.Error().Err(err).Msg("")

		return
	}

	if cfg.EnableDebugMode {
		logging.EnableDebugLogs()
	}

	grpcServer := grpc.NewServer([]grpc.ServerOption{grpc.MaxConcurrentStreams(1000000)}...)

	xdsServer := xds.NewServer()
	xdsServer.RegisterServer(ctx, grpcServer, cfg)

	nodeRegistryServer := registry.NewNodeRegistryServer(xdsServer.RWMutex(), xdsServer.ChannelNodes())
	pbNode.RegisterNodeRegistryServiceServer(grpcServer, nodeRegistryServer)

	serviceRegSrv := registry.NewServiceRegistryServer(xdsServer.RWMutex(), nodeRegistryServer, xdsServer.ChannelServices())
	pbService.RegisterServiceRegistryServiceServer(grpcServer, serviceRegSrv)

	lis, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(cfg.GRPCPort)))
	logging.LogErr(err)

	if cfg.EnableDebugMode {
		reflection.Register(grpcServer)
	}

	go func() {
		err = grpcServer.Serve(lis)
		logging.LogErr(err)
	}()

	select {
	case <-ctx.Done():
		grpcServer.Stop()

		stop()

		logging.DefaultLogger.Info().Msg("Shutting down CARISMA Control Plane")
	}
}
