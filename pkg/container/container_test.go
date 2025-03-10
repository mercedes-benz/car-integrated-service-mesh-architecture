// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"gotest.tools/v3/assert"
	"os"
	"strings"
	"testing"
)

const (
	defaultDomain  = "customcr.io"
	testImageName1 = "envoyproxy/envoy"
	testImageName2 = "envoyproxy/envoy:distroless-v1.27-latest"
	testImageName3 = "docker.io/envoyproxy/envoy:distroless-v1.27-latest"
	testImageName4 = "gcr.io/envoyproxy/envoy:distroless-v1.27-latest"
	testImageName5 = "localhost/envoyproxy/envoy:distroless-v1.27-latest"
	testImageName6 = "nonlocalhost/envoyproxy/envoy:distroless-v1.27-latest"
)

func TestParseFQIN(t *testing.T) {
	fqin, err := ParseFQIN(testImageName1, defaultDomain)
	assert.NilError(t, err)
	assert.Equal(t, fqin, "customcr.io/envoyproxy/envoy")

	fqin, err = ParseFQIN(testImageName2, defaultDomain)
	assert.NilError(t, err)
	assert.Equal(t, fqin, "customcr.io/envoyproxy/envoy:distroless-v1.27-latest")

	fqin, err = ParseFQIN(testImageName3, defaultDomain)
	assert.NilError(t, err)
	assert.Equal(t, fqin, "docker.io/envoyproxy/envoy:distroless-v1.27-latest")

	fqin, err = ParseFQIN(testImageName4, defaultDomain)
	assert.NilError(t, err)
	assert.Equal(t, fqin, "gcr.io/envoyproxy/envoy:distroless-v1.27-latest")

	fqin, err = ParseFQIN(testImageName5, defaultDomain)
	assert.NilError(t, err)
	assert.Equal(t, fqin, "localhost/envoyproxy/envoy:distroless-v1.27-latest")

	fqin, err = ParseFQIN(testImageName6, defaultDomain)
	assert.NilError(t, err)
	assert.Equal(t, fqin, "customcr.io/nonlocalhost/envoyproxy/envoy:distroless-v1.27-latest")
}

func TestFormatPorts(t *testing.T) {
	ports := []container.Port{
		{"", 80, 8080, "tcp"},
		{"127.0.0.1", 80, 8080, "tcp"},
	}
	result := formatPorts(ports)

	assert.Equal(t, result, "0.0.0.0:8080->80/tcp, 127.0.0.1:8080->80/tcp")
}

func TestFormatPortBindings(t *testing.T) {
	portMap := make(map[nat.Port][]nat.PortBinding, 3)
	portMap[("8000/tcp")] = []nat.PortBinding{{HostPort: "8000"}, {HostPort: "8080"}}
	portMap[("9000/tcp")] = []nat.PortBinding{{HostPort: "9000"}}
	portMap[("9002/tcp")] = []nat.PortBinding{{HostPort: "9002"}}

	result := formatPortBindings(portMap)

	assert.Assert(t, strings.Contains(result, "0.0.0.0:8000->8000/tcp"))
	assert.Assert(t, strings.Contains(result, "0.0.0.0:8080->8000/tcp"))
	assert.Assert(t, strings.Contains(result, "0.0.0.0:9000->9000/tcp"))
	assert.Assert(t, strings.Contains(result, "0.0.0.0:9002->9002/tcp"))
}

func TestDebugContainerManager(t *testing.T) {
	containerManager := NewDebugContainerManager(os.Stdout)
	defer func(containerManager Manager) {
		err := containerManager.Close()
		if err != nil {
			t.Fail()
		}
	}(containerManager)

	bundleID, servicePort, _ := containerManager.PullImageAndCreateContainer(
		context.Background(),
		testImageName1,
		nil,
		nil,
		false,
	)
	assert.Equal(t, bundleID, fmt.Sprintf(bundleIDFormat, 1))
	assert.Equal(t, servicePort, int32(8080))

	containers, _ := containerManager.Containers(context.Background())

	assert.Equal(t, len(containers), 1)

	for idx, cnt := range containers {
		assert.Equal(t, cnt.Image, testImageName1)
		assert.Equal(t, cnt.Status, statusRunning)

		err := containerManager.StopContainer(context.Background(), cnt.ID)
		assert.NilError(t, err)

		containers, _ = containerManager.Containers(context.Background())
		assert.Equal(t, containers[idx].Status, statusExited)

		err = containerManager.StartContainer(context.Background(), cnt.ID)
		assert.NilError(t, err)

		containers, _ = containerManager.Containers(context.Background())
		assert.Equal(t, containers[idx].Status, statusRunning)
	}

	bundleID, servicePort, err := containerManager.RemoveImageAndContainer(
		context.Background(),
		testImageName1,
		false,
	)
	assert.NilError(t, err)

	assert.Equal(t, bundleID, fmt.Sprintf(bundleIDFormat, 1))
	assert.Equal(t, servicePort, int32(8080))

	containers, _ = containerManager.Containers(context.Background())
	assert.Equal(t, len(containers), 0)

	bundleID, servicePort, err = containerManager.PullImageAndCreateContainer(
		context.Background(),
		testImageName2,
		nil,
		nil,
		false,
	)
	assert.NilError(t, err)

	assert.Equal(t, bundleID, fmt.Sprintf(bundleIDFormat, 2))
	assert.Equal(t, servicePort, int32(8081))

	containers, _ = containerManager.Containers(context.Background())
	bundleID, servicePort, err = containerManager.RemoveImageAndContainer(
		context.Background(),
		containers[0].Image,
		false,
	)
	assert.NilError(t, err)

	assert.Equal(t, bundleID, fmt.Sprintf(bundleIDFormat, 2))
	assert.Equal(t, servicePort, int32(8081))
}
