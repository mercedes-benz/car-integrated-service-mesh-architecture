// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
)

const (
	bundleConfigFileName = "info.json"
	wildcardIP           = "0.0.0.0"
)

// Manager represents the common container manager interface.
type Manager interface {
	// Containers returns the list of containers present on the machine.
	Containers(ctx context.Context) ([]Container, error)
	// StartContainer starts a container identified by its ID.
	StartContainer(ctx context.Context, id string) error
	// StopContainer stops a container identified by its ID.
	StopContainer(ctx context.Context, id string) error
	// PullImageAndCreateContainer pulls the requested image from the registry and creates a container with all ports published and the supplied port bindings created.
	PullImageAndCreateContainer(ctx context.Context, name string, args strslice.StrSlice, portBindings map[nat.Port][]nat.PortBinding, verifyBundleConfig bool) (string, int32,
		error)
	// RemoveImageAndContainer removes the specified image and all associated containers.
	RemoveImageAndContainer(ctx context.Context, name string, verifyBundleConfig bool) (string, int32, error)
	// Close closes the connection to the underlying container engine.
	Close() error
}

// Container encodes information related to a concrete container instance.
type Container struct {
	ID        string
	FirstName string
	Image     string
	Ports     string
	Status    string
}

// BundleConfig encodes a configuration file that shall be present in every in-car app.
type BundleConfig struct {
	BundleID string `json:"bundle_id"`
}
