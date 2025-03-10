// SPDX-License-Identifier: Apache-2.0

package xds

import (
	"errors"
	"fmt"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/registry"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/durationpb"
	"time"
)

func generateClusterName(bundleID string, isLocal bool) string {
	if isLocal {
		return fmt.Sprintf("local_%v_cluster", bundleID)
	} else {
		return fmt.Sprintf("%v_cluster", bundleID)
	}
}

func (x *Server) makeEndpoints(nodeID, bundleID string, ports []int32) ([]*endpoint.LocalityLbEndpoints, error) {
	nodeIdx, err := registry.GetNodeIdx(nodeID)
	if err != nil {
		return nil, err
	}
	if nodeIdx >= len(x.nodes) || nodeIdx < len(x.nodes) {
		return nil, errors.New("invalid node ID")
	}

	nodeAddress := x.nodes[nodeIdx].(*registry.NodeAddr).Host

	endpoints := make([]*endpoint.LocalityLbEndpoints, 0)
	for _, port := range ports {
		localityEndpoints := endpoint.LocalityLbEndpoints{
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  nodeAddress,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: uint32(port),
									},
								},
							},
						},
					},
				},
			}},
		}

		e := logging.DefaultLogger.Debug().
			Str("Node", nodeID).
			Str("Bundle", bundleID).
			Str("Address", nodeAddress).
			Int("Port", int(port))

		e.Msg("Registering service endpoint")

		endpoints = append(endpoints, &localityEndpoints)
	}

	return endpoints, nil
}

func (x *Server) makeClusters(localNodeID string, ingressPort int32) ([]types.Resource, error) {
	clusters := make([]types.Resource, 0)

	seen := make([]string, 0)
	for nodeID, serviceConfig := range x.services {
		for bundleID, instances := range serviceConfig {
			clusterID := generateClusterName(bundleID, false)
			ports := []int32{ingressPort}
			if nodeID == localNodeID {
				clusterID = generateClusterName(bundleID, true)
				ports = instances
			}

			if slices.Contains(seen, clusterID) {
				continue
			}

			seen = append(seen, clusterID)

			logging.DefaultLogger.Debug().
				Str("Cluster", clusterID).
				Msg("Registering cluster")

			endpoints, err := x.makeEndpoints(nodeID, bundleID, ports)
			if err != nil {
				return []types.Resource{}, err
			}

			clusters = append(clusters, &cluster.Cluster{
				Name:                          clusterID,
				DnsLookupFamily:               cluster.Cluster_V4_ONLY,
				ConnectTimeout:                durationpb.New(1 * time.Second),
				ClusterDiscoveryType:          &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
				LbPolicy:                      cluster.Cluster_ROUND_ROBIN,
				TypedExtensionProtocolOptions: config.HTTP2ProtocolOptions(),
				LoadAssignment: &endpoint.ClusterLoadAssignment{
					ClusterName: clusterID,
					Endpoints:   endpoints,
				},
			})
		}
	}

	return clusters, nil
}
