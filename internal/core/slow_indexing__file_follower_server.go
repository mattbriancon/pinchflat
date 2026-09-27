package core

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// FileFollowerServer watches a file for new lines and calls a handler for
// each. If there's no activity for a while it stops itself. The GenServer
// becomes a goroutine owned by this struct.
type FileFollowerServer struct {
	pollInterval    time.Duration
	activityTimeout time.Duration
	stopChan        chan struct{}
	done            chan struct{}
	handler         func(line string)
	file            *os.File
	reader          *bufio.Reader
	lastActivity    time.Time
	stopOnce        sync.Once
}

// start_link/0. pollInterval comes from Config.FileWatcherPollInterval.
func FileFollowerServerStartLink(ctx context.Context, pollInterval time.Duration) (*FileFollowerServer, error) {
	return &FileFollowerServer{
		pollInterval:    pollInterval,
		activityTimeout: 600_000 * time.Millisecond, // @activity_timeout_ms 600_000
		stopChan:        make(chan struct{}),
		done:            make(chan struct{}),
	}, nil
}

// watch_file/3: starts following filepath, calling handler with each new line.
func (f *FileFollowerServer) WatchFile(filepath string, handler func(line string)) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}

	f.file = file
	f.reader = bufio.NewReader(file)
	f.handler = handler
	f.lastActivity = time.Now().UTC()

	go f.readNewLines()

	return nil
}

// stop/1
func (f *FileFollowerServer) Stop() error {
	slog.Debug("Gracefully stopping file follower")
	f.stopOnce.Do(func() { close(f.stopChan) })
	// Wait for goroutine to finish
	<-f.done
	if f.file != nil {
		f.file.Close()
	}
	return nil
}

// GetDoneChan returns the done channel for testing purposes
func (f *FileFollowerServer) GetDoneChan() <-chan struct{} {
	return f.done
}

// readNewLines continuously reads lines from the file until stopped or timeout
func (f *FileFollowerServer) readNewLines() {
	defer close(f.done)

	for {
		select {
		case <-f.stopChan:
			return
		default:
		}

		// Check for activity timeout
		if time.Since(f.lastActivity) > f.activityTimeout {
			slog.Debug("No activity for 600000ms. Requesting stop.")
			f.stopOnce.Do(func() { close(f.stopChan) })
			return
		}

		if !f.attemptProcessNewLines() {
			// No new lines, wait for poll interval before trying again
			select {
			case <-f.stopChan:
				return
			case <-time.After(f.pollInterval):
				continue
			}
		}
	}
}

// attemptProcessNewLines reads one line and processes it
func (f *FileFollowerServer) attemptProcessNewLines() bool {
	line, err := f.reader.ReadString('\n')

	if err != nil && err != io.EOF {
		// Real error reading file
		slog.Error("Error reading file", "error", err)
		return false
	}

	if line != "" {
		// We got a line (might be followed by EOF if no trailing newline)
		f.handler(line)
		f.lastActivity = time.Now().UTC()
		return true
	}

	if err == io.EOF {
		// EOF without any line data
		slog.Debug("Current batch of media processed. Will check again in", "ms", f.pollInterval.Milliseconds())
		return false
	}

	return false
}
