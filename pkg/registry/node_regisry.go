// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"errors"
	"fmt"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/channel"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	pb "github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/protobuf/carisma/node/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
)

// NodeAddr encodes the hostname and port of an endpoint.
type NodeAddr struct {
	Host string
	Port int
}

// Network returns the address's network name, "tcp".
func (a *NodeAddr) Network() string { return "tcp" }

// String returns the address's string representation.
func (a *NodeAddr) String() string {
	if a == nil {
		return "<nil>"
	}

	return net.JoinHostPort(a.Host, strconv.Itoa(a.Port))
}

// GetNodeIdx extracts the node's index out of a node ID.
func GetNodeIdx(nodeID string) (int, error) {
	s := strings.Split(nodeID, "-")

	return strconv.Atoi(s[len(s)-1])
}

// NodeRegistryServer implements the node registry server.
type NodeRegistryServer struct {
	pb.UnimplementedNodeRegistryServiceServer

	mu    *sync.RWMutex // protects nodes
	nodes []net.Addr

	broker *channel.Broker[*pb.DeploymentConfiguration]

	chanNodes chan<- net.Addr
}

// Register receives a RegisterRequest containing the IP address and port of the node and replies with assigned the node id after registration.
func (s *NodeRegistryServer) Register(_ context.Context, addr *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	address := addr.Address
	port := int(addr.Port)

	s.mu.Lock()

	s.nodes = append(s.nodes, &NodeAddr{
		Host: address,
		Port: port,
	})

	newNodeIdx := len(s.nodes) - 1
	newNode := s.nodes[newNodeIdx]

	s.mu.Unlock()

	s.chanNodes <- newNode

	logging.DefaultLogger.Debug().
		Str("Node", fmt.Sprintf("node-%v", newNodeIdx)).
		Str("Address", address).
		Int("Port", port).
		Msg("Registering node")

	nodeID := fmt.Sprintf("node-%v", newNodeIdx)

	return &pb.RegisterResponse{Id: nodeID}, nil
}

// OpenChannel opens a gRPC channel that processes deployment configurations.
func (s *NodeRegistryServer) OpenChannel(stream pb.NodeRegistryService_OpenChannelServer) error {
	// try to retrieve metadata
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.FailedPrecondition, errorMsgMissingHeader)
	}

	// check if CARISMA header is set
	nodeID := md.Get(HeaderNodeID)
	if len(nodeID) < 1 {
		return status.Errorf(codes.FailedPrecondition, errorMsgMissingHeader)
	}

	// validate provided node ID
	nodeIdx, err := s.ValidateNodeID(nodeID[0])
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, err.Error())
	}

	nodeHostname := s.nodes[nodeIdx].(*NodeAddr).Host

	// Processes all deployment configuration messages.
	chRead := s.broker.Read()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-stream.Context().Done():
				return
			case <-ctx.Done():
				return
			case msgDplmCfg, ok := <-chRead:
				if ok {
					err := stream.Send(msgDplmCfg)
					logging.LogErr(err)
				}
			}
		}
	}()

	// Trigger initial distribution of deployment configuration for the node that opened the channel.
	dplmCfg := config.DeploymentConfigForStartingNode(nodeHostname)
	if j, err := dplmCfg.JSON(); err != nil {
		s.broker.Write(&pb.DeploymentConfiguration{
			Json:      string(j),
			StateType: pb.DeploymentConfiguration_STATE_TYPE_ACTUAL,
		})
	}

	for {
		msgDplmCfg, err := stream.Recv()
		if err != nil {
			dplmCfg := config.DeploymentConfigForStoppedNode(nodeHostname)
			if j, err := dplmCfg.JSON(); err != nil {
				s.broker.Write(&pb.DeploymentConfiguration{
					Json:      string(j),
					StateType: pb.DeploymentConfiguration_STATE_TYPE_ACTUAL,
				})
			}

			cancel()

			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}

		s.broker.Write(msgDplmCfg)
	}
}

// ValidateNodeID checks whether the node with the provided ID has been registered.
func (s *NodeRegistryServer) ValidateNodeID(nodeID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodeIdx, err := GetNodeIdx(nodeID)
	if err != nil {
		return -1, err
	}

	if nodeIdx < 0 || nodeIdx > len(s.nodes)-1 {
		return -1, errors.New(errorInvalidNodeID)
	}

	return nodeIdx, nil
}

// NewNodeRegistryServer creates a new instance of the NodeRegistryServer.
func NewNodeRegistryServer(mu *sync.RWMutex, channelNodes chan<- net.Addr) *NodeRegistryServer {
	s := &NodeRegistryServer{
		mu:        mu,
		nodes:     make([]net.Addr, 0),
		broker:    channel.NewBroker[*pb.DeploymentConfiguration](),
		chanNodes: channelNodes,
	}

	go s.broker.Listen()

	return s
}
