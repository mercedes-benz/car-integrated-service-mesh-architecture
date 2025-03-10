// SPDX-License-Identifier: Apache-2.0

package logging

import "github.com/rs/zerolog"

// DefaultDebugLevelWriter is the debug level writer to be used.
var DefaultDebugLevelWriter = newDebugLevelWriter(DefaultLogger)

// DebugLevelWriter wraps a logger.
type DebugLevelWriter struct {
	logger zerolog.Logger
}

func newDebugLevelWriter(logger zerolog.Logger) DebugLevelWriter {
	return DebugLevelWriter{
		logger,
	}
}

// Write writes from a buffer to the logger using the debug log level.
func (d DebugLevelWriter) Write(b []byte) (n int, err error) {
	d.logger.Debug().Msg(string(b))

	return len(b), nil
}
