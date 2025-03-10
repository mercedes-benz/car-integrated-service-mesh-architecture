// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"gotest.tools/v3/assert"
	"testing"
)

func TestFromJSON(t *testing.T) {
	expectation := DeploymentConfig{
		"host-1": {
			State: NodeStateRunning,
			Images: []string{
				"localhost/image1:latest",
				"localhost/image2:v1",
			},
		},
		"host-2": {
			State: NodeStateRunning,
			Images: []string{
				"localhost/image3:latest",
			},
		},
	}

	fileContent := `{
	"host-1": {
		"state": "running",
		"container": [
			"localhost/image1:latest",
			"localhost/image2:v1"
		]
	},
	"host-2": {
		"state": "running",
		"container": [
			"localhost/image3:latest"
		]
	}
}`

	dplmCfg, err := DeploymentConfigFromJSON([]byte(fileContent))
	assert.NilError(t, err)

	assert.DeepEqual(t, dplmCfg, expectation)
}

func TestJSON(t *testing.T) {
	dplmCfg := DeploymentConfig{
		"host-1": {
			State: NodeStateRunning,
			Images: []string{
				"localhost/image1:latest",
				"localhost/image2:v1",
			},
		},
		"host-2": {
			State: NodeStateRunning,
			Images: []string{
				"localhost/image3:latest",
			},
		},
	}

	j, err := dplmCfg.JSON()
	assert.NilError(t, err)

	var dat DeploymentConfig

	err = json.Unmarshal(j, &dat)
	assert.NilError(t, err)

	for hostname, nodeConfig := range dat {
		assert.DeepEqual(t, dplmCfg[hostname], nodeConfig)
	}
}

func TestFromJSONInvalid(t *testing.T) {
	fileContent := `[
    {
        "name": "host-1",
        "state": "running",
        "container": [
			"localhost/image1:latest",
            "localhost/image2:v1"
		]
    },
	{
        "name": "host-1",
        "state": "running",
        "container": [
			"localhost/image3:latest"
		]
    }
]`

	_, err := DeploymentConfigFromJSON([]byte(fileContent))

	assert.Assert(t, err != nil)
}

func TestDeploymentConfigForStartingNode(t *testing.T) {
	expectation := DeploymentConfig{
		"host-1": {
			State:  NodeStateStarting,
			Images: []string{},
		},
	}

	dplmCfg := DeploymentConfigForStartingNode("host-1")

	assert.DeepEqual(t, dplmCfg, expectation)
}

func TestDeploymentConfigForStoppedNode(t *testing.T) {
	expectation := DeploymentConfig{
		"host-1": {
			State:  NodeStateStopped,
			Images: []string{},
		},
	}

	dplmCfg := DeploymentConfigForStoppedNode("host-1")

	assert.DeepEqual(t, dplmCfg, expectation)
}
