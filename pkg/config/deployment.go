// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
)

// NodeState encodes the state of a node.
type NodeState string

const (
	// NodeStateStarting represents the state of a node that is starting.
	NodeStateStarting NodeState = "starting"
	// NodeStateRunning represents the state of a node that is running.
	NodeStateRunning NodeState = "running"
	// NodeStateStopping represents the state of a node that is about to shut down.
	NodeStateStopping NodeState = "stopping"
	// NodeStateStopped represents the state of a node that is not running.
	NodeStateStopped NodeState = "stopped"
)

// NodeConfig encodes the state of a node and the containerized software that (shall) run(s) on it.
type NodeConfig struct {
	State  NodeState `json:"state"`
	Images []string  `json:"container"`
}

// DeploymentConfig encodes a mapping of NodeConfig instances to the hostnames of the respective nodes.
type DeploymentConfig map[string]NodeConfig

// DeploymentConfigFromJSON parses JSON into an instance of DeploymentConfig.
func DeploymentConfigFromJSON(str []byte) (DeploymentConfig, error) {
	var dplmCfg DeploymentConfig

	if err := json.Unmarshal(str, &dplmCfg); err != nil {
		return dplmCfg, err
	}

	return dplmCfg, nil
}

// JSON returns the JSON representation of a DeploymentConfig instance.
func (d DeploymentConfig) JSON() ([]byte, error) {
	j, err := json.MarshalIndent(d, "", "    ")

	if err != nil {
		return []byte{}, err
	}

	return j, nil
}

// DeploymentConfigForStartingNode constructs a DeploymentConfig instance that represents a starting node.
func DeploymentConfigForStartingNode(hostname string) DeploymentConfig {
	dplmCfg := DeploymentConfig{
		hostname: {
			State:  NodeStateStarting,
			Images: []string{},
		},
	}

	return dplmCfg
}

// DeploymentConfigForStoppedNode constructs a DeploymentConfig instance that represents a node that is about to shut
// down.
func DeploymentConfigForStoppedNode(hostname string) DeploymentConfig {
	dplmCfg := DeploymentConfig{
		hostname: {
			State:  NodeStateStopped,
			Images: []string{},
		},
	}

	return dplmCfg
}
