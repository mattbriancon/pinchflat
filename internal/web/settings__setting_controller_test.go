package web_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestSettingController_ShowSettings(t *testing.T) {
	t.Run("renders the page", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/settings")

		if !strings.Contains(res.HTML(t, 200), "Settings") {
			t.Error("expected 'Settings' in HTML")
		}
	})
}

func TestSettingController_UpdateSettings(t *testing.T) {
	t.Run("saves and redirects when data is valid", func(t *testing.T) {
		c := webtest.New(t)

		updateAttrs := store.Attrs{"download_throughput_limit": "4.2M"}
		res := c.Patch("/settings", "setting", updateAttrs)

		if res.RedirectedTo(t) != "/settings" {
			t.Errorf("expected redirect to /settings, got %s", res.RedirectedTo(t))
		}

		// Verify the setting was saved
		res = c.Get("/settings")
		if !strings.Contains(res.HTML(t, 200), "4.2M") {
			t.Error("expected updated download_throughput_limit in HTML")
		}
	})
}

func TestSettingController_AppInfo(t *testing.T) {
	t.Run("renders the page", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/app_info")

		if !strings.Contains(res.HTML(t, 200), "App Info") {
			t.Error("expected 'App Info' in HTML")
		}
	})
}

func TestSettingController_DownloadLogs(t *testing.T) {
	t.Run("downloads logs", func(t *testing.T) {
		c := webtest.New(t)

		// Create a temporary log file
		logDir := filepath.Join(os.TempDir(), "pinchflat", "data")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			t.Fatalf("failed to create log dir: %v", err)
		}
		logPath := filepath.Join(logDir, "pinchflat.log")
		if err := os.WriteFile(logPath, []byte("test log data"), 0644); err != nil {
			t.Fatalf("failed to write log file: %v", err)
		}
		defer os.RemoveAll(logDir)

		// Set the log path in config
		oldLogPath := c.App.Config.LogPath
		c.App.Config.LogPath = logPath
		defer func() { c.App.Config.LogPath = oldLogPath }()

		res := c.Get("/download_logs")

		// Check that we got a 200 status
		if res.Status != 200 {
			t.Errorf("expected status 200, got %d", res.Status)
		}

		// Check that the response contains our log data
		if !strings.Contains(res.Body, "test log data") {
			t.Error("expected 'test log data' in response")
		}
	})

	t.Run("redirects when log file is not found", func(t *testing.T) {
		c := webtest.New(t)

		// Clear the log path
		oldLogPath := c.App.Config.LogPath
		c.App.Config.LogPath = ""
		defer func() { c.App.Config.LogPath = oldLogPath }()

		res := c.Get("/download_logs")

		if res.RedirectedTo(t) != "/app_info" {
			t.Errorf("expected redirect to /app_info, got %s", res.RedirectedTo(t))
		}

		if res.Flash("error") != "Log file couldn't be found" {
			t.Errorf("expected error flash, got %s", res.Flash("error"))
		}
	})
}
