// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"flag"
	bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"time"
)

const (
	// TimeUnit represents the default time unit within the config file.
	TimeUnit = time.Second
	// XDSClusterName represents the name of the cluster of xDS servers.
	XDSClusterName   = "xds-cluster"
	envoyClusterName = "envoy-cluster"
	configFilePath   = "/opt/carisma/conf/carisma.json"
)

// Config encodes all supported configuration parameters.
type Config struct {
	EnableDebugMode                bool
	EmulateContainerRuntime        bool
	EnableCentralMode              bool   `json:"enableCentralMode"`
	EnableDiscovery                bool   `json:"enableDiscovery"`
	CentralNodeHostname            string `json:"centralNode"`
	NodeHostname                   string `json:"node"`
	StatusMgrPort                  int    `json:"statusMgrPort"`
	GRPCPort                       int    `json:"gRPCPort"`
	UDPPort                        int    `json:"udpPort"`
	UDPDelay                       int    `json:"udpDelay"`
	UDPTimeout                     int    `json:"udpTimeout"`
	IngressPort                    int    `json:"ingressPort"`
	EgressPort                     int    `json:"egressPort"`
	AdminPort                      int    `json:"adminPort"`
	DefaultContainerRegistryDomain string `json:"defaultContainerRegistryDomain"`
}

// New creates a new instance of Config based on default values. The default values can be overwritten by actual values specified in a file representation of the struct or by
// command line flags.
func New() (*Config, error) {
	cfg := Default()

	// try to read the local config file and construct a configuration with it
	if _, err := os.Stat(configFilePath); err == nil {
		if cfgFileContent, err := os.ReadFile(configFilePath); err == nil {
			if err := cfg.applyArgumentsFromConfigFileContent(cfgFileContent); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	cfg.applyCommandlineFlags()

	cfg.fix()

	return cfg, nil
}

// Default creates a new configuration based on default values.
func Default() *Config {
	hostname, err := os.Hostname()
	if err != nil {
		logging.DefaultLogger.Error().Err(err).Msg("could not determine hostname")
	}

	return &Config{
		EnableDebugMode:                false,
		EmulateContainerRuntime:        false,
		EnableCentralMode:              false,
		EnableDiscovery:                false,
		CentralNodeHostname:            "carisma-central",
		NodeHostname:                   hostname,
		StatusMgrPort:                  8010,
		GRPCPort:                       8016,
		UDPPort:                        8829,
		UDPDelay:                       5,
		UDPTimeout:                     60,
		IngressPort:                    8000,
		EgressPort:                     9000,
		AdminPort:                      9901,
		DefaultContainerRegistryDomain: "docker.io",
	}
}

func (c *Config) applyArgumentsFromConfigFileContent(fileContent []byte) error {
	err := json.Unmarshal(fileContent, c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) applyCommandlineFlags() {
	flag.BoolVar(&c.EnableDebugMode, "enable-debug-mode", c.EnableDebugMode, "Enable the debug mode")
	flag.BoolVar(&c.EmulateContainerRuntime, "emulate-container-runtime", c.EmulateContainerRuntime, "Run package manager with a dummy container runtime")
	flag.BoolVar(&c.EnableCentralMode, "enable-central-mode", c.EnableCentralMode, "Enable the central mode for the current node")
	flag.BoolVar(&c.EnableDiscovery, "enable-discovery", c.EnableDiscovery, "Enable UPD-based discovery of the nodes")
	flag.StringVar(&c.CentralNodeHostname, "central-node", c.CentralNodeHostname, "The hostname of the central node")
	flag.StringVar(&c.NodeHostname, "node", c.NodeHostname, "The hostname of the current node")
	flag.IntVar(&c.StatusMgrPort, "status-manager-port", c.StatusMgrPort, "The port for the status manager to listen on")
	flag.IntVar(&c.GRPCPort, "grpc-port", c.GRPCPort, "The port for the gRPC server to listens on")
	flag.IntVar(&c.UDPPort, "udp-port", c.UDPPort, " The UDP port for the package manager to listen on")
	flag.IntVar(&c.UDPDelay, "udp-delay", c.UDPDelay, "The delay between the UPD messages")
	flag.IntVar(&c.UDPTimeout, "udp-timeout", c.UDPTimeout, "The maximum time to wait for an UDP broadcast message")
	flag.IntVar(&c.IngressPort, "ingress-port", c.IngressPort, "The ingress port for Envoy to listen on")
	flag.IntVar(&c.EgressPort, "egress-port", c.EgressPort, "The egress port for Envoy to listen on")
	flag.StringVar(&c.DefaultContainerRegistryDomain, "default-container-registry-domain", c.DefaultContainerRegistryDomain,
		"The default container registry domain to be used for normalizing image names")

	flag.Parse()
}

func (c *Config) fix() {
	if c.EnableCentralMode {
		c.CentralNodeHostname = c.NodeHostname
	}
}

// EnvoyBootstrapConfigWithNodeID creates a configuration for the Envoy proxy based on the encoded configuration values.
func (c *Config) EnvoyBootstrapConfigWithNodeID(nodeID string) string {
	bootstrapCfg := bootstrap.Bootstrap{
		Node: &core.Node{
			Id:      nodeID,
			Cluster: envoyClusterName,
		},
		DynamicResources: &bootstrap.Bootstrap_DynamicResources{
			LdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
				ResourceApiVersion: core.ApiVersion_V3,
			},
			CdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
				ResourceApiVersion: core.ApiVersion_V3,
			},
			AdsConfig: &core.ApiConfigSource{
				ApiType: core.ApiConfigSource_GRPC,
				GrpcServices: []*core.GrpcService{
					{
						TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
							EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
								ClusterName: XDSClusterName,
							},
						},
					},
				},
				TransportApiVersion: core.ApiVersion_V3,
			},
		},
		StaticResources: &bootstrap.Bootstrap_StaticResources{
			Clusters: []*cluster.Cluster{
				{
					Name: XDSClusterName,
					ClusterDiscoveryType: &cluster.Cluster_Type{
						Type: cluster.Cluster_STRICT_DNS,
					},
					DnsLookupFamily:               cluster.Cluster_V4_ONLY,
					TypedExtensionProtocolOptions: HTTP2ProtocolOptions(),
					LoadAssignment: &endpoint.ClusterLoadAssignment{
						ClusterName: XDSClusterName,
						Endpoints: []*endpoint.LocalityLbEndpoints{
							{
								LbEndpoints: []*endpoint.LbEndpoint{{
									HostIdentifier: &endpoint.LbEndpoint_Endpoint{
										Endpoint: &endpoint.Endpoint{
											Address: &core.Address{
												Address: &core.Address_SocketAddress{
													SocketAddress: &core.SocketAddress{
														Protocol: core.SocketAddress_TCP,
														Address:  c.CentralNodeHostname,
														PortSpecifier: &core.SocketAddress_PortValue{
															PortValue: uint32(c.GRPCPort),
														},
													},
												},
											},
										},
									},
								}},
							},
						},
					},
				},
			},
		},
	}

	if c.EnableDebugMode {
		bootstrapCfg.Admin = &bootstrap.Admin{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: uint32(c.AdminPort),
						},
					},
				},
			},
		}
	}

	return protojson.Format(&bootstrapCfg)
}
