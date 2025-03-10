// SPDX-License-Identifier: Apache-2.0

package xds

import (
	"context"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoveryservice "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/registry"
	"google.golang.org/grpc"
	"net"
	"sync"
)

// Server manages the currently registered nodes and services.
type Server struct {
	snapshotVersion int
	cache           cache.SnapshotCache

	channelNodes    chan net.Addr
	channelServices chan registry.ServiceConfigSnapshot

	mu       sync.RWMutex // protects nodes and services
	nodes    []net.Addr
	services registry.ServiceConfigSnapshot
}

// NewServer creates a new xDS server.
func NewServer() *Server {
	c := cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	s := &Server{
		snapshotVersion: -1,
		cache:           c,
		channelNodes:    make(chan net.Addr),
		channelServices: make(chan registry.ServiceConfigSnapshot),
		nodes:           make([]net.Addr, 0),
		services:        make(registry.ServiceConfigSnapshot),
	}

	return s
}

// RegisterServer attaches an xDS server to the provided gRPC server.
func (x *Server) RegisterServer(ctx context.Context, grpcSrv *grpc.Server, cfg *config.Config) {
	go func() {
		for {
			select {
			case newNode := <-x.channelNodes:
				x.mu.Lock()

				x.nodes = append(x.nodes, newNode)
				err := x.generateSnapshots(ctx, cfg)

				x.mu.Unlock()

				if err != nil {
					return
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case newServiceConfig := <-x.channelServices:
				x.mu.Lock()

				x.services = newServiceConfig

				err := x.generateSnapshots(ctx, cfg)

				x.mu.Unlock()

				if err != nil {
					return
				}
			}
		}
	}()

	// run the xDS server
	srv := server.NewServer(ctx, x.cache, nil)

	discoveryservice.RegisterAggregatedDiscoveryServiceServer(grpcSrv, srv)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcSrv, srv)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcSrv, srv)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcSrv, srv)
}

// RWMutex returns a pointer to the mutex used for protecting shared resources.
func (x *Server) RWMutex() *sync.RWMutex {
	return &x.mu
}

// ChannelNodes returns the channel that can be used to introduce new nodes.
func (x *Server) ChannelNodes() chan<- net.Addr {
	return x.channelNodes
}

// ChannelServices returns the channel that can be used to introduce new services.
func (x *Server) ChannelServices() chan<- registry.ServiceConfigSnapshot {
	return x.channelServices
}
