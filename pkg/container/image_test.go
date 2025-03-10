// SPDX-License-Identifier: Apache-2.0

package container

import (
	"gotest.tools/v3/assert"
	"testing"
)

func TestExtractImage(t *testing.T) {
	i1 := Image{Name: "envoyproxy/envoy", Version: "distroless-v1.27-latest"}
	i2 := ParseImageName("envoyproxy/envoy:distroless-v1.27-latest")
	assert.DeepEqual(t, i1, i2)

	i1 = Image{Name: "envoyproxy/envoy", Version: ""}
	i2 = ParseImageName("envoyproxy/envoy:")
	assert.DeepEqual(t, i1, i2)

	i1 = Image{Name: "envoyproxy/envoy", Version: "latest"}
	i2 = ParseImageName("envoyproxy/envoy")
	assert.DeepEqual(t, i1, i2)
}

func TestExtractImageList(t *testing.T) {
	containers := []Container{
		{ID: "4b0240ee1570", Image: "envoyproxy/envoy:distroless-v1.27-latest", Ports: "8010", Status: statusRunning},
		{ID: "5af73b532a06", Image: "envoyproxy/envoy:", Ports: "8000", Status: statusExited},
		{ID: "4b0240ee1570", Image: "envoyproxy/envoy:distroless-v1.27-latest", Ports: "8042", Status: statusRunning},
		{ID: "5af73b532a06", Image: "envoyproxy/envoy", Ports: "8080", Status: statusExited},
	}
	images := ExtractImageList(containers)

	i0 := Image{Name: "envoyproxy/envoy", Version: "distroless-v1.27-latest"}
	assert.DeepEqual(t, images[0], i0)
	assert.DeepEqual(t, images[2], i0)

	i1 := Image{Name: "envoyproxy/envoy", Version: ""}
	assert.DeepEqual(t, images[1], i1)

	i3 := Image{Name: "envoyproxy/envoy", Version: "latest"}
	assert.DeepEqual(t, images[3], i3)
}
