// SPDX-License-Identifier: Apache-2.0

package config

import (
	"gotest.tools/v3/assert"
	"os"
	"testing"
)

func TestConfigCentralMode(t *testing.T) {
	expectation := Default()
	expectation.NodeHostname = "localhost"
	expectation.GRPCPort = 8042

	configFileContent := `{
	"EnableCentralMode": false,
	"Node": "localhost",
    "GRPCPort": 8042
}`

	cfg := Default()
	err := cfg.applyArgumentsFromConfigFileContent([]byte(configFileContent))
	assert.NilError(t, err)

	cfg.fix()

	assert.DeepEqual(t, *cfg, *expectation)
}

func TestConfigSatelliteMode(t *testing.T) {
	expectation := Default()
	expectation.NodeHostname = "localhost"
	expectation.GRPCPort = 8042

	configFileContent := `{
	"EnableCentralMode": false,
	"Node": "localhost",
    "GRPCPort": 8042
}`

	cfg := Default()
	err := cfg.applyArgumentsFromConfigFileContent([]byte(configFileContent))
	assert.NilError(t, err)

	cfg.fix()

	assert.DeepEqual(t, *cfg, *expectation)
}

func TestCommandLineArgumentsForCentralNode(t *testing.T) {
	os.Args = []string{"cmd", "-enable-central-mode=true"}

	expectation := Default()
	expectation.EnableCentralMode = true
	expectation.CentralNodeHostname = "localhost"
	expectation.NodeHostname = "localhost"
	expectation.GRPCPort = 8042

	configFileContent := `{
	"EnableCentralMode": false,
	"Node": "localhost",
    "GRPCPort": 8042
}`

	cfg := Default()
	err := cfg.applyArgumentsFromConfigFileContent([]byte(configFileContent))
	assert.NilError(t, err)

	cfg.applyCommandlineFlags()

	cfg.fix()

	assert.DeepEqual(t, *cfg, *expectation)
}

func TestCommandLineArgumentsForSatelliteNode(t *testing.T) {
	expectation := Default()
	expectation.EnableCentralMode = false
	expectation.CentralNodeHostname = "carisma-central"
	expectation.NodeHostname = "localhost"
	expectation.GRPCPort = 8042

	configFileContent := `{
	"Node": "localhost",
    "GRPCPort": 8042
}`

	cfg := Default()
	err := cfg.applyArgumentsFromConfigFileContent([]byte(configFileContent))
	assert.NilError(t, err)

	cfg.applyCommandlineFlags()

	cfg.fix()

	assert.DeepEqual(t, *cfg, *expectation)
}
