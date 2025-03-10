// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	pb "github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/protobuf/carisma/service/v1"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"sync"
)

const (
	HeaderNodeID = "x-carisma-node-id"
)

// ServiceConfigSnapshot represents a mapping of service to nodes at one point in time.
type ServiceConfigSnapshot map[string]map[string][]int32

// ServiceRegistryServer implements the node registry server.
type ServiceRegistryServer struct {
	pb.UnimplementedServiceRegistryServiceServer

	mu       *sync.RWMutex // protects services
	services ServiceConfigSnapshot

	nodeRegistry *NodeRegistryServer

	updateChannel chan<- ServiceConfigSnapshot
}

// OpenChannel opens a gRPC channel that processes service announcements and updates the service registry accordingly.
func (s *ServiceRegistryServer) OpenChannel(stream pb.ServiceRegistryService_OpenChannelServer) error {
	// try to retrieve metadata
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.FailedPrecondition, errorMsgMissingHeader)
	}

	// check if CARISMA header is set and validate value
	nodeID := md.Get(HeaderNodeID)
	if _, err := s.nodeRegistry.ValidateNodeID(nodeID[0]); len(nodeID) < 1 || err != nil {
		return status.Errorf(codes.FailedPrecondition, errorMsgMissingHeader)
	}

	// process all announcement messages and invoke the respective register/unregister method
	for {
		announcement, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&emptypb.Empty{})
		}
		if err != nil {
			return err
		}

		if announcement.RegistrationState == pb.ServiceAnnouncement_REGISTRATION_STATE_REGISTERED {
			s.registerService(nodeID[0], announcement.BundleId, announcement.LocalPort)

			logging.DefaultLogger.Info().
				Str("Node-ID", nodeID[0]).
				Str("Bundle-ID", announcement.BundleId).
				Int32("Port", announcement.LocalPort).
				Msg("Registered service")
		} else {
			s.unregisterService(nodeID[0], announcement.BundleId, announcement.LocalPort)

			logging.DefaultLogger.Info().
				Str("Node-ID", nodeID[0]).
				Str("Bundle-ID", announcement.BundleId).
				Int32("Port", announcement.LocalPort).
				Msg("Unregistered service")
		}
	}
}

func (s *ServiceRegistryServer) registerService(nodeID string, bundleID string, port int32) {
	s.mu.Lock()

	if _, ok := s.services[nodeID]; !ok {
		s.services[nodeID] = make(map[string][]int32)
	}

	s.services[nodeID][bundleID] = append(s.services[nodeID][bundleID], port)

	// Remove duplicate ports
	slices.Sort(s.services[nodeID][bundleID])
	s.services[nodeID][bundleID] = slices.Compact(s.services[nodeID][bundleID])

	s.mu.Unlock()

	s.updateChannel <- s.services
}

func (s *ServiceRegistryServer) unregisterService(nodeID string, bundleID string, port int32) {
	s.mu.Lock()

	idxPort := slices.Index(s.services[nodeID][bundleID], port)
	if idxPort > -1 {
		s.services[nodeID][bundleID] = slices.Delete(s.services[nodeID][bundleID], idxPort, idxPort+1)
	}

	if len(s.services[nodeID][bundleID]) == 0 {
		delete(s.services[nodeID], bundleID)
	}

	s.mu.Unlock()

	s.updateChannel <- s.services
}

// NewServiceRegistryServer creates a new instance of the ServiceRegistryServer.
func NewServiceRegistryServer(mu *sync.RWMutex, nReg *NodeRegistryServer, uC chan<- ServiceConfigSnapshot) *ServiceRegistryServer {
	s := &ServiceRegistryServer{
		mu:            mu,
		services:      make(map[string]map[string][]int32),
		updateChannel: uC,
		nodeRegistry:  nReg,
	}

	return s
}
