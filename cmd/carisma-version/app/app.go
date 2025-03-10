// SPDX-License-Identifier: Apache-2.0

package app

import (
	"fmt"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/version"
)

func Run() {
	if version.Version() == "DEVELOPMENT" {
		fmt.Printf("Running CARISMA built from source based on %v\n", version.Commit())
	} else {
		fmt.Printf("Running CARISMA %v\n", version.Version())
	}

}
