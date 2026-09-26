package core

import (
	"context"
	"time"
)

// FileFollowerServer watches a file for new lines and calls a handler for
// each. If there's no activity for a while it stops itself. The GenServer
// becomes a goroutine owned by this struct.
type FileFollowerServer struct {
	pollInterval    time.Duration
	activityTimeout time.Duration
	// implementation state goes here
}

// start_link/0. pollInterval comes from Config.FileWatcherPollInterval.
func FileFollowerServerStartLink(ctx context.Context, pollInterval time.Duration) (*FileFollowerServer, error) {
	panic("unported: Pinchflat.SlowIndexing.FileFollowerServer.start_link/0")
}

// watch_file/3: starts following filepath, calling handler with each new line.
func (f *FileFollowerServer) WatchFile(filepath string, handler func(line string)) error {
	panic("unported: Pinchflat.SlowIndexing.FileFollowerServer.watch_file/3")
}

// stop/1
func (f *FileFollowerServer) Stop() error {
	panic("unported: Pinchflat.SlowIndexing.FileFollowerServer.stop/1")
}
