// SPDX-License-Identifier: Apache-2.0

package xds

import (
	"context"
	"fmt"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	gRPCRouteName       = "grpc_route"
	localGRPCRouteName  = "local_grpc_route"
	gRPCVHostName       = "grpc_vhost"
	localGRPCVHostName  = "local_grpc_vhost"
	ingressStatPrefix   = "ingress_http"
	egressStatPrefix    = "egress_http"
	ingressListenerName = "ingress_listener"
	egressListenerName  = "egress_listener"
)

func generateNodeID(id int) string {
	return fmt.Sprintf("node-%v", id)
}

func makeHTTPConnectionManager() ([]*anypb.Any, error) {
	routerConfig, _ := anypb.New(&router.Router{})
	manager := []*hcm.HttpConnectionManager{
		{
			CodecType:  hcm.HttpConnectionManager_AUTO,
			StatPrefix: ingressStatPrefix,
			RouteSpecifier: &hcm.HttpConnectionManager_Rds{
				Rds: &hcm.Rds{
					ConfigSource:    getConfigSource(),
					RouteConfigName: localGRPCRouteName,
				},
			},
			HttpFilters: []*hcm.HttpFilter{{
				Name:       wellknown.Router,
				ConfigType: &hcm.HttpFilter_TypedConfig{TypedConfig: routerConfig},
			}},
		},
		{
			CodecType:  hcm.HttpConnectionManager_AUTO,
			StatPrefix: egressStatPrefix,
			RouteSpecifier: &hcm.HttpConnectionManager_Rds{
				Rds: &hcm.Rds{
					ConfigSource:    getConfigSource(),
					RouteConfigName: gRPCRouteName,
				},
			},
			HttpFilters: []*hcm.HttpFilter{{
				Name:       wellknown.Router,
				ConfigType: &hcm.HttpFilter_TypedConfig{TypedConfig: routerConfig},
			}},
		},
	}

	marshalledIngressConnectionManagerConfig, err := anypb.New(manager[0])
	if err != nil {
		return []*anypb.Any{}, err
	}

	marshalledEgressConnectionManagerConfig, err := anypb.New(manager[1])
	if err != nil {
		return []*anypb.Any{}, err
	}

	return []*anypb.Any{marshalledIngressConnectionManagerConfig, marshalledEgressConnectionManagerConfig}, err
}

func (x *Server) makeRoutes(localNodeID string) []types.Resource {
	routes := make([]*route.Route, 0)
	localRoutes := make([]*route.Route, 0)

	for nodeID, serviceConfig := range x.services {
		for bundleID := range serviceConfig {
			clusterID := generateClusterName(bundleID, nodeID == localNodeID)

			r := &route.Route{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: fmt.Sprintf("/%v", bundleID),
					},
					Grpc: &route.RouteMatch_GrpcRouteMatchOptions{},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: clusterID,
						},
					},
				},
			}

			e := logging.DefaultLogger.Debug().
				Str("Node", localNodeID).
				Str("Bundle", bundleID).
				Str("Cluster", clusterID).
				Str("Route", fmt.Sprintf("/%v", bundleID)).
				Str("Domain", "*")

			if nodeID != localNodeID {
				e.
					Str("gRPCRoute", gRPCRouteName).
					Str("vHost", gRPCVHostName).
					Msg("Registering route")

				routes = append(routes, r)
			} else {
				e.
					Str("gRPCRoute", fmt.Sprintf("%s,%s", gRPCRouteName, localGRPCRouteName)).
					Str("vHost", localGRPCVHostName).
					Msg("Registering local route")

				localRoutes = append(localRoutes, r)
			}
		}
	}

	return []types.Resource{
		&route.RouteConfiguration{
			Name: gRPCRouteName,
			VirtualHosts: []*route.VirtualHost{{
				Name:    gRPCVHostName,
				Domains: []string{"*"},
				Routes:  append(routes, localRoutes...),
			}},
		},
		&route.RouteConfiguration{
			Name: localGRPCRouteName,
			VirtualHosts: []*route.VirtualHost{{
				Name:    localGRPCVHostName,
				Domains: []string{"*"},
				Routes:  localRoutes,
			}},
		},
	}
}

func (x *Server) makeHTTPListener(cfg *config.Config) ([]types.Resource, error) {
	httpConnectionManager, err := makeHTTPConnectionManager()
	if err != nil {
		return []types.Resource{}, err
	}

	logging.DefaultLogger.Debug().
		Str("gRPCRoute", localGRPCRouteName).
		Uint32("Port", uint32(cfg.IngressPort)).
		Msg("Registering ingress listener")

	logging.DefaultLogger.Debug().
		Str("gRPCRoute", gRPCRouteName).
		Uint32("Port", uint32(cfg.EgressPort)).
		Msg("Registering egress listener")

	return []types.Resource{
		&listener.Listener{
			Name: ingressListenerName,
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: uint32(cfg.IngressPort),
						},
					},
				},
			},
			TrafficDirection: core.TrafficDirection_INBOUND,
			FilterChains: []*listener.FilterChain{{
				Filters: []*listener.Filter{{
					Name: wellknown.Router,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: httpConnectionManager[0],
					},
				}},
			}},
		},
		&listener.Listener{
			Name: egressListenerName,
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: uint32(cfg.EgressPort),
						},
					},
				},
			},
			TrafficDirection: core.TrafficDirection_OUTBOUND,
			FilterChains: []*listener.FilterChain{{
				Filters: []*listener.Filter{{
					Name: wellknown.Router,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: httpConnectionManager[1],
					},
				}},
			}},
		},
	}, nil
}

func (x *Server) generateSnapshots(ctx context.Context, cfg *config.Config) error {
	x.snapshotVersion++
	for i := range x.nodes {
		nodeID := generateNodeID(i)

		logging.DefaultLogger.Debug().Msgf("Generating snapshot for %s", nodeID)

		httpListener, err := x.makeHTTPListener(cfg)
		if err != nil {
			return err
		}

		clusters, err := x.makeClusters(nodeID, int32(cfg.IngressPort))
		if err != nil {
			return err
		}
		snapshot, err := cache.NewSnapshot(
			fmt.Sprintf("%v.0", x.snapshotVersion),
			map[resource.Type][]types.Resource{
				resource.ClusterType:  clusters,
				resource.RouteType:    x.makeRoutes(nodeID),
				resource.ListenerType: httpListener,
			},
		)
		if err != nil {
			return err
		}

		if err := x.cache.SetSnapshot(ctx, nodeID, snapshot); err != nil {
			logging.DefaultLogger.Panic().Err(err).Msg("")
		} else {
			logging.DefaultLogger.Debug().
				Str("Node", nodeID).
				Str("Snapshot", fmt.Sprintf("%v.0", x.snapshotVersion)).
				Msg("Setting node-specific snapshot")
		}
	}
	return nil
}
