package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
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
			ta := coretest.NewApp(t)

			ta.YtDlpMock.Update.Expect(func() (string, error) {
				return "", nil
			})
			ta.YtDlpMock.Version.Expect(func() (string, error) {
				return tt.versionResp, nil
			})

			err := ta.Oban.PerformJob(ta.Ctx, core.UpdateWorkerName, map[string]any{})
			if err != nil {
				t.Fatalf("PerformJob failed: %v", err)
			}

			if tt.expectVer != "" {
				val, err := ta.SettingsGet(ta.Ctx, "yt_dlp_version")
				if err != nil {
					t.Fatalf("SettingsGet failed: %v", err)
				}

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
