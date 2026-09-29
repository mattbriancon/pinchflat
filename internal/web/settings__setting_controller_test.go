package web_test

import (
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

		updateAttrs := map[string]any{"download_throughput_limit": "4.2M"}
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

// A browser posts setting[...] keys, the hidden "false" before a checked
// checkbox's "true", and every field of the form.
func TestSettingController_UpdateSettingsFromBrowserForm(t *testing.T) {
	c := webtest.New(t)
	mustOK(t)(c.App.SetSetting(c.Ctx, "onboarding", false))

	form := url.Values{
		"_method":                                   {"patch"},
		"setting[restrict_filenames]":               {"false", "true"},
		"setting[video_codec_preference]":           {"av01"},
		"setting[audio_codec_preference]":           {"opus"},
		"setting[extractor_sleep_interval_seconds]": {"3"},
		"setting[download_throughput_limit]":        {""},
		"setting[youtube_api_key]":                  {""},
	}
	req := httptest.NewRequest("POST", "/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if got := c.Do(req).RedirectedTo(t); got != "/settings" {
		t.Fatalf("redirect = %q, want /settings", got)
	}

	s, err := c.App.GetSettingsRecord(c.Ctx)
	must(t, err)
	if s.RestrictFilenames == nil || !*s.RestrictFilenames {
		t.Errorf("restrict_filenames = %v, want true", s.RestrictFilenames)
	}
	if s.VideoCodecPreference != "av01" || s.AudioCodecPreference != "opus" || s.ExtractorSleepIntervalSeconds != 3 {
		t.Errorf("got %q %q %d", s.VideoCodecPreference, s.AudioCodecPreference, s.ExtractorSleepIntervalSeconds)
	}
	if s.Onboarding {
		t.Error("onboarding should stay false")
	}
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
		must(t, os.MkdirAll(logDir, 0755))
		logPath := filepath.Join(logDir, "pinchflat.log")
		must(t, os.WriteFile(logPath, []byte("test log data"), 0644))
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
