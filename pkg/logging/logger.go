// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"github.com/rs/zerolog"
	"os"
	"strconv"
	"time"
)

// DefaultLogger is the logger to be used.
var DefaultLogger = newLogger()

type levelWriter struct {
	infOut zerolog.ConsoleWriter
	errOut zerolog.ConsoleWriter
}

// Write should not be called, see WriteLevel.
func (l levelWriter) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

// WriteLevel writes to the appropriate output stream.
func (l levelWriter) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	if level <= zerolog.WarnLevel {
		return l.infOut.Write(p)
	} else {
		return l.errOut.Write(p)
	}
}

func newLogger() zerolog.Logger {
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		return file + ":" + strconv.Itoa(line)
	}

	return zerolog.New(levelWriter{
		infOut: zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC1123},
		errOut: zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC1123},
	}).With().Timestamp().Caller().Logger()
}

// EnableDebugLogs enables the output of debug log messages.
func EnableDebugLogs() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

// LogInfo logs a message to the info log.
func LogInfo(msg string) {
	DefaultLogger.Info().Msg(msg)
}

// LogDbg logs a message to the debug log.
func LogDbg(msg string) {
	DefaultLogger.Debug().Msg(msg)
}

// LogErr logs a potential error to the appropriate log if the error is present.
func LogErr(err error) {
	if err != nil {
		DefaultLogger.Error().Err(err).Msg("")
	}
}
