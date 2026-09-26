package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestUpdateWorker_Perform(t *testing.T) {
	t.Run("calls the yt-dlp runner to update yt-dlp", func(t *testing.T) {
		ta := coretest.NewApp(t)

		ta.YtDlpMock.Update.Expect(func() (string, error) {
			return "", nil
		})
		ta.YtDlpMock.Version.Expect(func() (string, error) {
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.UpdateWorkerName, map[string]any{})
	})

	t.Run("saves the new version to the database", func(t *testing.T) {
		ta := coretest.NewApp(t)

		ta.YtDlpMock.Update.Expect(func() (string, error) {
			return "", nil
		})
		ta.YtDlpMock.Version.Expect(func() (string, error) {
			return "1.2.3", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.UpdateWorkerName, map[string]any{})

		val, err := ta.SettingsGet(ta.Ctx, "yt_dlp_version")
		if err != nil {
			t.Errorf("Failed to get yt_dlp_version: %v", err)
		}

		var version string
		if v, ok := val.(*string); ok && v != nil {
			version = *v
		} else if v, ok := val.(string); ok {
			version = v
		}
		if version != "1.2.3" {
			t.Errorf("Expected yt_dlp_version to be '1.2.3', got %v (type: %T)", val, val)
		}
	})
}
