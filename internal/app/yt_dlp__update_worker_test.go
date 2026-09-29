package app_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
)

func TestUpdateWorker_Perform(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		versionResp string
		expectVer   string
	}{
		{
			name:        "calls yt-dlp runner to update",
			versionResp: "",
			expectVer:   "",
		},
		{
			name:        "saves new version to database",
			versionResp: "1.2.3",
			expectVer:   "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ta := apptest.NewApp(t)

			ta.YtDlpMock.Update.Expect(func() (string, error) {
				return "", nil
			})
			ta.YtDlpMock.Version.Expect(func() (string, error) {
				return tt.versionResp, nil
			})

			err := ta.Oban.PerformJob(ta.Ctx, app.UpdateWorkerName, map[string]any{})
			must(t, err)

			if tt.expectVer != "" {
				val, err := ta.GetSetting(ta.Ctx, "yt_dlp_version")
				must(t, err)

				var version string
				if v, ok := val.(*string); ok && v != nil {
					version = *v
				} else if v, ok := val.(string); ok {
					version = v
				}
				if version != tt.expectVer {
					t.Errorf("expected version %q, got %q", tt.expectVer, version)
				}
			}
		})
	}
}
