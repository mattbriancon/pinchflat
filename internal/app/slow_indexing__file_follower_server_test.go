package app_test

import (
	"os"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
)

func TestFileFollowerServer_WatchFile(t *testing.T) {
	t.Run("handles existing lines", func(t *testing.T) {
		ta := apptest.NewApp(t)
		defer ta.App.DB.Close()

		tmpfile, err := os.CreateTemp(ta.Config.TmpfileDirectory, "*.txt")
		if err != nil {
			t.Fatalf("create file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.WriteString("line1\nline2"); err != nil {
			t.Fatalf("write: %v", err)
		}
		tmpfile.Close()

		server, err := app.FileFollowerServerStartLink(ta.Ctx, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("start server: %v", err)
		}

		lines := make(chan string, 10)
		if err := server.WatchFile(tmpfile.Name(), func(line string) {
			lines <- line
		}); err != nil {
			t.Fatalf("watch: %v", err)
		}

		timeout := time.NewTimer(2 * time.Second)
		defer timeout.Stop()

		receivedLines := []string{}
		expectedLines := []string{"line1\n", "line2"}

		for i := 0; i < len(expectedLines); i++ {
			select {
			case line := <-lines:
				receivedLines = append(receivedLines, line)
			case <-timeout.C:
				t.Fatalf("timeout waiting for line %d, got %d", i, len(receivedLines))
			}
		}

		for i, expected := range expectedLines {
			if i < len(receivedLines) && receivedLines[i] != expected {
				t.Errorf("line[%d]=%q, want %q", i, receivedLines[i], expected)
			}
		}

		server.Stop()
	})

	t.Run("handles new lines appended", func(t *testing.T) {
		ta := apptest.NewApp(t)
		defer ta.App.DB.Close()

		tmpfile, err := os.CreateTemp(ta.Config.TmpfileDirectory, "*.txt")
		if err != nil {
			t.Fatalf("create file: %v", err)
		}
		defer os.Remove(tmpfile.Name())
		tmpfile.Close()

		server, err := app.FileFollowerServerStartLink(ta.Ctx, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("start server: %v", err)
		}

		lines := make(chan string, 10)
		if err := server.WatchFile(tmpfile.Name(), func(line string) {
			lines <- line
		}); err != nil {
			t.Fatalf("watch: %v", err)
		}

		file, err := os.OpenFile(tmpfile.Name(), os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatalf("open: %v", err)
		}

		timeout := time.NewTimer(5 * time.Second)
		defer timeout.Stop()

		if _, err := file.WriteString("line1\n"); err != nil {
			t.Fatalf("write1: %v", err)
		}

		select {
		case line := <-lines:
			if line != "line1\n" {
				t.Errorf("line=%q, want line1\\n", line)
			}
		case <-timeout.C:
			t.Fatal("timeout on line1")
		}

		if _, err := file.WriteString("line2"); err != nil {
			t.Fatalf("write2: %v", err)
		}

		select {
		case line := <-lines:
			if line != "line2" {
				t.Errorf("line=%q, want line2", line)
			}
		case <-timeout.C:
			t.Fatal("timeout on line2")
		}

		file.Close()
		server.Stop()
	})
}

func TestFileFollowerServer_Stop(t *testing.T) {
	t.Run("stops watching", func(t *testing.T) {
		ta := apptest.NewApp(t)
		defer ta.App.DB.Close()

		tmpfile, err := os.CreateTemp(ta.Config.TmpfileDirectory, "*.txt")
		if err != nil {
			t.Fatalf("create file: %v", err)
		}
		defer os.Remove(tmpfile.Name())
		tmpfile.Close()

		server, err := app.FileFollowerServerStartLink(ta.Ctx, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("start server: %v", err)
		}

		if err := server.WatchFile(tmpfile.Name(), func(line string) {}); err != nil {
			t.Fatalf("watch: %v", err)
		}

		time.Sleep(50 * time.Millisecond)
		server.Stop()

		select {
		case <-server.GetDoneChan():
		case <-time.After(1 * time.Second):
			t.Fatal("server did not stop")
		}
	})
}
