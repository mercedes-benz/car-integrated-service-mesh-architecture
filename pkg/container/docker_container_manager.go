// SPDX-License-Identifier: Apache-2.0

package container

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/distribution/reference"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"io"
	"strconv"
	"strings"
)

func formatPorts(ports []container.Port) string {
	var sb strings.Builder

	for idx, port := range ports {
		if idx != 0 {
			sb.WriteString(", ")
		}

		ip := wildcardIP
		if len(port.IP) > 0 {
			ip = port.IP
		}

		sb.WriteString(fmt.Sprintf("%s:%d->%d/%s", ip, port.PublicPort, port.PrivatePort, port.Type))
	}

	return sb.String()
}

// NewDockerContainerManager creates a Manager that interacts with Docker (compatible) engines.
func NewDockerContainerManager(ctx context.Context) (Manager, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	_, err = c.Info(ctx) // check if docker (compatible) engine is running
	if err != nil {
		return nil, err
	}

	return &dockerContainerManager{c}, nil
}

type dockerContainerManager struct {
	client *client.Client
}

func (d dockerContainerManager) bundleConfiguration(ctx context.Context, containerID string) (BundleConfig, error) {
	r, _, err := d.client.CopyFromContainer(ctx, containerID, bundleConfigFileName)
	if err != nil {
		return BundleConfig{}, err
	}
	defer func() {
		logging.LogErr(r.Close())
	}()

	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			return BundleConfig{}, err
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if header.Name != bundleConfigFileName {
				continue
			}

			b, err := io.ReadAll(tarReader)
			if err != nil {
				return BundleConfig{}, err
			}

			var c BundleConfig
			err = json.Unmarshal(b, &c)
			if err != nil {
				return BundleConfig{}, err
			}

			return c, nil
		}
	}

	return BundleConfig{}, fmt.Errorf("cannot extract deployment config from container %s\n", containerID)
}

func (d dockerContainerManager) containerServicePort(ctx context.Context, id string) (int32, error) {
	i, err := d.client.ContainerInspect(ctx, id)
	if err != nil {
		return -1, err
	}

	for _, binding := range i.HostConfig.PortBindings {
		servicePort, err := strconv.Atoi(binding[0].HostPort)
		if err != nil {
			return -1, err
		}

		return int32(servicePort), nil
	}

	return -1, nil
}

// ParseFQIN parses a string into a fully qualified reference.
func ParseFQIN(name, defaultDomain string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return "", err
	}

	normalizedImageName := ref.Name()
	idx := strings.IndexRune(normalizedImageName, '/')
	if idx == -1 {
		return "", errors.New("invalid reference format: missing '/'")
	}

	if len(normalizedImageName) != len(name) && normalizedImageName[:idx+1] != name[:idx+1] {
		normalizedImageName = defaultDomain + normalizedImageName[idx:]
	}

	if tagged, isTagged := ref.(reference.Tagged); isTagged {
		normalizedImageName += ":" + tagged.Tag()
	}

	return normalizedImageName, nil
}

func (d dockerContainerManager) Containers(ctx context.Context) ([]Container, error) {
	containerList, err := d.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var containers []Container
	for _, c := range containerList {
		c := Container{c.ID,
			c.Names[0],
			c.Image,
			formatPorts(c.Ports),
			c.Status,
		}
		containers = append(containers, c)
	}

	return containers, nil
}

func (d dockerContainerManager) StartContainer(ctx context.Context, id string) error {
	if err := d.client.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return err
	}

	return nil
}

func (d dockerContainerManager) StopContainer(ctx context.Context, id string) error {
	if err := d.client.ContainerStop(ctx, id, container.StopOptions{}); err != nil {
		return err
	}

	return nil
}

func (d dockerContainerManager) PullImageAndCreateContainer(ctx context.Context, name string, args strslice.StrSlice, portBindings map[nat.Port][]nat.PortBinding,
	verifyBundleConfig bool) (string, int32, error) {

	// download the image
	reader, err := d.client.ImagePull(ctx, name, image.PullOptions{})
	if err != nil {
		return "", -1, err
	} else {
		// We need to wait for the download to finish.
		// The indicator of choice is an EOF error issued by the reader returned by ImagePull.
		bufDontCare := make([]byte, 32*1024)
		for {
			if _, err := reader.Read(bufDontCare); err != nil {
				break
			}
		}
		_ = reader.Close()
	}

	// create container based on image
	r, err := d.client.ContainerCreate(
		ctx,
		&container.Config{Image: name, Cmd: args},
		&container.HostConfig{NetworkMode: "slirp4netns", PublishAllPorts: true, PortBindings: portBindings},
		nil,
		nil,
		"",
	)
	if err != nil {
		return "", -1, err
	}

	// start container
	if err = d.client.ContainerStart(
		ctx,
		r.ID,
		container.StartOptions{},
	); err != nil {
		return "", -1, err
	}

	// success but also retrieve bundleID and servicePort
	if verifyBundleConfig {
		bundleConfig, err := d.bundleConfiguration(ctx, r.ID)
		if err != nil {
			return "", -1, err
		}

		servicePort, err := d.containerServicePort(ctx, r.ID)
		if err != nil {
			return "", -1, err
		}

		return bundleConfig.BundleID, servicePort, nil
	}

	return "", -1, nil
}

func (d dockerContainerManager) RemoveImageAndContainer(ctx context.Context, imageName string, verifyBundleConfig bool) (string, int32, error) {
	containerList, err := d.client.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "ancestor", Value: imageName}),
	})
	if err != nil {
		return "", -1, err
	}

	if len(containerList) == 0 {
		return "", -1, errors.New("can not not find container(s) running the supplied image")
	}

	var (
		bundleID          = ""
		servicePort int32 = -1
	)

	if verifyBundleConfig {
		bundleConfig, err := d.bundleConfiguration(ctx, containerList[0].ID)
		if err != nil {
			return "", -1, err
		}

		bundleID = bundleConfig.BundleID

		servicePort, err = d.containerServicePort(ctx, containerList[0].ID)
		if err != nil {
			return "", -1, err
		}
	}

	if err := d.client.ContainerRemove(
		ctx,
		containerList[0].ID,
		container.RemoveOptions{Force: true},
	); err != nil {
		return "", -1, err
	}

	if _, err := d.client.ImageRemove(
		ctx,
		containerList[0].ImageID,
		image.RemoveOptions{Force: true},
	); err != nil {
		return "", -1, err
	}

	return bundleID, servicePort, nil
}

func (d dockerContainerManager) Close() error {
	if err := d.client.Close(); err != nil {
		return err
	}

	return nil
}
