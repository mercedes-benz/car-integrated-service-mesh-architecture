// SPDX-License-Identifier: Apache-2.0

package web

import "embed"

//go:embed all:template
var Templates embed.FS

//go:embed all:static
var Assets embed.FS
