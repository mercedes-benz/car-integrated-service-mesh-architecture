// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"io"
	"math/rand"
	"strings"
)

const (
	statusRunning   = "running"
	statusExited    = "exited"
	bundleIDFormat  = "com.mercedes_benz.app_%d"
	servicePortBase = 8080
	appIdxBase      = 1
)

// NewDebugContainerManager creates a Manager that prints to the command line for debugging purposes.
func NewDebugContainerManager(writer io.Writer) Manager {
	_, err := fmt.Fprintln(writer, "Starting debug container manager")
	logging.LogErr(err)

	return &debugContainerManager{
		writer,
		make(map[string]virtualContainer),
		servicePortBase,
		appIdxBase,
	}
}

type virtualContainer struct {
	appIdx    int32
	container *Container
}

type debugContainerManager struct {
	writer          io.Writer
	container       map[string]virtualContainer
	nextServicePort int32
	nextAppIdx      int32
}

func generateRandomContainerId() string {
	var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, 64)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}

	return string(b)
}

func formatPortBindings(portBindings map[nat.Port][]nat.PortBinding) string {
	var sb strings.Builder

	first := true
	for containerPort, hostPorts := range portBindings {
		if !first && len(portBindings) > 1 {
			sb.WriteString(", ")
		}

		// usually there is only one
		for idx, port := range hostPorts {
			if idx != 0 {
				sb.WriteString(", ")
			}

			ip := wildcardIP
			if len(port.HostIP) > 0 {
				ip = port.HostIP
			}

			sb.WriteString(fmt.Sprintf("%s:%s->%s", ip, port.HostPort, containerPort))
		}

		first = false
	}

	return sb.String()
}

func (d *debugContainerManager) printContainerTable() {
	var sb strings.Builder
	sb.WriteString("currently deployed containers:\n")

	const format = "|%-64s|%-64s|%-10s|%-10s\n"
	sb.WriteString(fmt.Sprintf(format, "id", "image", "status", "port(s)"))

	for id, vc := range d.container {
		sb.WriteString(fmt.Sprintf(format, id, vc.container.Image, vc.container.Status, vc.container.Ports))
	}

	_, err := fmt.Fprint(d.writer, sb.String())
	logging.LogErr(err)
}

func (d *debugContainerManager) Containers(_ context.Context) ([]Container, error) {
	a := make([]Container, 0, len(d.container))

	for _, vc := range d.container {
		a = append(a, *vc.container)
	}

	return a, nil
}

func (d *debugContainerManager) StartContainer(_ context.Context, id string) error {
	vc, ok := d.container[id]
	if ok {
		vc.container.Status = statusRunning

		_, err := fmt.Fprintf(d.writer, "starting container with ID %s\n", id)
		logging.LogErr(err)

		d.printContainerTable()

		return nil
	}

	d.printContainerTable()

	return fmt.Errorf("container not found: %v", id)
}

func (d *debugContainerManager) StopContainer(_ context.Context, id string) error {
	vc, ok := d.container[id]
	if ok {
		vc.container.Status = statusExited

		_, err := fmt.Fprintf(d.writer, "stopping container with ID %s\n", id)
		logging.LogErr(err)

		d.printContainerTable()

		return nil
	}

	d.printContainerTable()

	return fmt.Errorf("container not found: %v", id)
}

func (d *debugContainerManager) PullImageAndCreateContainer(_ context.Context, name string, _ strslice.StrSlice, portBindings map[nat.Port][]nat.PortBinding, _ bool) (string,
	int32, error) {
	id := generateRandomContainerId()
	d.container[id] = virtualContainer{
		d.nextAppIdx,
		&Container{
			ID:        id,
			FirstName: id,
			Image:     name,
			Ports:     formatPortBindings(portBindings),
			Status:    statusRunning,
		},
	}

	_, err := fmt.Fprintf(d.writer, "deploying image %s into container with ID %s\n", name, id)
	logging.LogErr(err)

	d.printContainerTable()

	bundleID := fmt.Sprintf("com.mercedes_benz.app_%d", d.nextAppIdx)
	servicePort := d.nextServicePort

	d.nextServicePort += 1
	d.nextAppIdx += 1

	return bundleID, servicePort, nil
}

func (d *debugContainerManager) RemoveImageAndContainer(_ context.Context, name string, _ bool) (string, int32, error) {
	for id, vc := range d.container {
		if vc.container.Image == name {
			delete(d.container, id)

			_, err := fmt.Fprintf(d.writer, "removing container with ID %s based on image with name %s\n", id, name)
			logging.LogErr(err)

			bundleID := fmt.Sprintf(bundleIDFormat, vc.appIdx)
			servicePort := servicePortBase - 1 + vc.appIdx

			d.printContainerTable()

			return bundleID, servicePort, nil
		}
	}

	d.printContainerTable()

	return "", -1, nil
}

func (d *debugContainerManager) Close() error {
	_, err := fmt.Fprintln(d.writer, "shutting down container manager")
	logging.LogErr(err)

	return nil
}
