// SPDX-License-Identifier: Apache-2.0

package container

import "strings"

// Image represents a container image with a specific version.
type Image struct {
	Name    string
	Version string
}

// ParseImageName turns a string into an instance of Image.
func ParseImageName(str string) Image {
	idx := strings.LastIndex(str, ":")

	if idx != -1 {
		return Image{Name: str[:idx], Version: str[idx+1:]}
	}

	return Image{Name: str, Version: "latest"}
}

// ExtractImageList transforms a slice of Container instances into a slice of Image instances.
func ExtractImageList(containers []Container) []Image {
	images := make([]Image, len(containers))
	for idx, c := range containers {
		images[idx] = ParseImageName(c.Image)
	}

	return images
}
