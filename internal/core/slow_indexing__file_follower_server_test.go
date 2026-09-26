package core_test

import (
	"os"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestFileFollowerServer_WatchFile(t *testing.T) {
	t.Run("calls the handler for each existing line in the file", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		// Create a temp file
		tmpfile, err := os.CreateTemp(ta.Config.TmpfileDirectory, "*.txt")
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		// Write initial content
		if _, err := tmpfile.WriteString("line1\nline2"); err != nil {
			t.Fatalf("write file: %v", err)
		}
		tmpfile.Close()

		// Start the server
		server, err := core.FileFollowerServerStartLink(ta.Ctx, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("start server: %v", err)
		}

		// Collect lines
		lines := make(chan string, 10)
		handler := func(line string) {
			lines <- line
		}

		// Watch the file
		if err := server.WatchFile(tmpfile.Name(), handler); err != nil {
			t.Fatalf("watch file: %v", err)
		}

		// Read the expected lines with timeout
		timeout := time.NewTimer(2 * time.Second)
		defer timeout.Stop()

		receivedLines := []string{}
		expectedLines := []string{"line1\n", "line2"}

		for i := 0; i < len(expectedLines); i++ {
			select {
			case line := <-lines:
				receivedLines = append(receivedLines, line)
			case <-timeout.C:
				t.Fatalf("timeout waiting for line %d, got %d: %v", i, len(receivedLines), receivedLines)
			}
		}

		// Verify the lines
		for i, expected := range expectedLines {
			if i < len(receivedLines) && receivedLines[i] != expected {
				t.Errorf("line %d: got %q, want %q", i, receivedLines[i], expected)
			}
		}

		// Stop the server
		server.Stop()
	})

	t.Run("calls the handler for each new line in the file", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		// Create a temp file
		tmpfile, err := os.CreateTemp(ta.Config.TmpfileDirectory, "*.txt")
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())
		tmpfile.Close()

		// Start the server
		server, err := core.FileFollowerServerStartLink(ta.Ctx, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("start server: %v", err)
		}

		// Collect lines
		lines := make(chan string, 10)
		handler := func(line string) {
			lines <- line
		}

		// Watch the file
		if err := server.WatchFile(tmpfile.Name(), handler); err != nil {
			t.Fatalf("watch file: %v", err)
		}

		// Open file for writing
		file, err := os.OpenFile(tmpfile.Name(), os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatalf("open file for append: %v", err)
		}

		timeout := time.NewTimer(5 * time.Second)
		defer timeout.Stop()

		// Write first line
		if _, err := file.WriteString("line1\n"); err != nil {
			t.Fatalf("write line1: %v", err)
		}

		select {
		case line := <-lines:
			if line != "line1\n" {
				t.Errorf("expected %q, got %q", "line1\n", line)
			}
		case <-timeout.C:
			t.Fatal("timeout waiting for line1")
		}

		// Write second line (without newline)
		if _, err := file.WriteString("line2"); err != nil {
			t.Fatalf("write line2: %v", err)
		}

		select {
		case line := <-lines:
			if line != "line2" {
				t.Errorf("expected %q, got %q", "line2", line)
			}
		case <-timeout.C:
			t.Fatal("timeout waiting for line2")
		}

		file.Close()
		server.Stop()
	})
}

func TestFileFollowerServer_Stop(t *testing.T) {
	t.Run("stops the watcher", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		// Create a temp file
		tmpfile, err := os.CreateTemp(ta.Config.TmpfileDirectory, "*.txt")
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())
		tmpfile.Close()

		// Start the server
		server, err := core.FileFollowerServerStartLink(ta.Ctx, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("start server: %v", err)
		}

		// Watch the file
		handler := func(line string) {}
		if err := server.WatchFile(tmpfile.Name(), handler); err != nil {
			t.Fatalf("watch file: %v", err)
		}

		// Give it a moment to start
		time.Sleep(50 * time.Millisecond)

		// Stop the server
		server.Stop()

		// After stopping, the Done channel should be closed
		select {
		case <-server.GetDoneChan():
			// Expected: the goroutine has finished
		case <-time.After(1 * time.Second):
			t.Fatal("server did not stop in time")
		}
	})
}
