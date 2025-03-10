// SPDX-License-Identifier: Apache-2.0

package xds

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
)

func getConfigSource() *core.ConfigSource {
	return &core.ConfigSource{
		ResourceApiVersion: resource.DefaultAPIVersion,
		ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				TransportApiVersion:       resource.DefaultAPIVersion,
				ApiType:                   core.ApiConfigSource_GRPC,
				SetNodeOnFirstMessageOnly: true,
				GrpcServices: []*core.GrpcService{{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: config.XDSClusterName},
					},
				}},
			},
		},
	}
}
