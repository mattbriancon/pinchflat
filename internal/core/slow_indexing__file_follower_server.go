package core

import (
	"context"
)

// FileFollowerServer watches a file for new lines and processes them as they come in.
// If there's no activity for a certain amount of time, the server stops itself.

type FileFollowerServer struct {
	// GenServer state will be added during implementation
}

const fileFollowerServerPollIntervalMs = 0 // will be set to compile-time config value

// FileFollowerServerStartLink/0
func FileFollowerServerStartLink(ctx context.Context) (*FileFollowerServer, error) {
	panic("unported: Pinchflat.SlowIndexing.FileFollowerServer.start_link/0")
}

// FileFollowerServerWatchFile/3
func (f *FileFollowerServer) FileFollowerServerWatchFile(filepath string, handler func(string) error) error {
	panic("unported: Pinchflat.SlowIndexing.FileFollowerServer.watch_file/3")
}

// FileFollowerServerStop/0
func (f *FileFollowerServer) FileFollowerServerStop() error {
	panic("unported: Pinchflat.SlowIndexing.FileFollowerServer.stop/1")
}
