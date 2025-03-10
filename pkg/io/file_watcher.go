// SPDX-License-Identifier: Apache-2.0

package io

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"os"
)

// A Handler responds to changes regarding the content of a file.
type Handler interface {
	// ProcessDiff invokes the function wrapped inside the HandlerFunc.
	ProcessDiff(a, b []byte)
}

// HandlerFunc is an adapter to allow the use of an appropriate function as a Handler for a change regarding the content of a file.
type HandlerFunc func(a, b []byte)

func (f HandlerFunc) ProcessDiff(a, b []byte) {
	f(a, b)
}

// FileWatcher wraps file that can be watched for changes.
type FileWatcher struct {
	filePath string
	lastRead []byte
	watcher  *fsnotify.Watcher
	handler  Handler
}

// NewFileWatcher obtains a handle to the provided file and creates a new FileWatcher based on that handle.
func NewFileWatcher(filePath string) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err = watcher.Add(filePath); err != nil {
		return nil, err
	}

	return &FileWatcher{filePath: filePath, watcher: watcher}, nil
}

// HandleDiff sets the handler that processes a change regarding the content of the file.
func (c *FileWatcher) HandleDiff(handler func(a, b []byte)) {
	c.handler = HandlerFunc(handler)
}

// Watch blocks the caller and starts watching the file for changes.
func (c *FileWatcher) Watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-c.watcher.Events:
			if err := waitUntilFind(c.filePath); err != nil {
				logging.LogErr(err)

				continue
			}

			content, err := os.ReadFile(c.filePath)
			if err != nil {
				logging.LogErr(err)

				continue
			}

			if c.handler != nil {
				c.handler.ProcessDiff(c.lastRead, content)
			}

			c.lastRead = content

			if event.Op&fsnotify.Rename == fsnotify.Rename {
				err := c.watcher.Add(c.filePath)
				logging.LogErr(err)
			}
		case err := <-c.watcher.Errors:
			logging.LogErr(err)
		}
	}
}

func waitUntilFind(filename string) error {
	for {
		_, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				return err
			}
		}
		break
	}
	return nil
}

// Diff manually triggers a diff.
func (c *FileWatcher) Diff(force bool) {
	if force {
		c.lastRead = []byte{}
	}

	c.watcher.Events <- fsnotify.Event{Name: c.filePath}
}

// Close frees the resources associated with a FileWatcher instance.
func (c *FileWatcher) Close() {
	_ = c.watcher.Close()
}
