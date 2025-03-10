// SPDX-License-Identifier: Apache-2.0

package version

// Injected upon build
var (
	version = "DEVELOPMENT"

	commit string
)

// Version returns either the CARISMA version or DEVELOPMENT in case of unreleased code.
func Version() string {
	return version
}

// Commit returns the commit the build is based on.
func Commit() string {
	return commit
}
