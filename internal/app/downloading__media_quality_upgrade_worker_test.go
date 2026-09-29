package app_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestMediaQualityUpgradeWorker_Perform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		downloadedDays  int
		wantEnqueuedJob bool
	}{
		{
			name:            "kicks off a task for redownloadable media items",
			downloadedDays:  5,
			wantEnqueuedJob: true,
		},
		{
			name:            "does not kickoff a task for non-redownloadable media items",
			downloadedDays:  1,
			wantEnqueuedJob: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			ctx := ta.Ctx

			mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"redownload_delay_days": 4})
			source := apptest.SourceFixture(t, ta, store.Attrs{
				"media_profile_id": mediaProfile.ID,
				"inserted_at":      apptest.NowMinus(10, "days"),
			})

			mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{
				SourceID:          store.Ptr(source.ID),
				UploadedAt:        store.Ptr(apptest.NowMinus(6, "days")),
				MediaDownloadedAt: store.Ptr(apptest.NowMinus(tt.downloadedDays, "days")),
			})

			err := ta.Oban.PerformJob(ctx, app.MediaQualityUpgradeWorkerName, map[string]any{})
			if err != nil {
				t.Fatalf("PerformJob failed: %v", err)
			}

			enqueued := ta.Oban.Enqueued(t, obanlite.Match{
				Worker: app.MediaDownloadWorkerName,
				Args:   map[string]any{"id": mediaItem.ID, "quality_upgrade?": true},
			})
			if tt.wantEnqueuedJob {
				if len(enqueued) != 1 {
					t.Errorf("expected 1 enqueued job, got %d", len(enqueued))
				}
			} else {
				if len(enqueued) != 0 {
					t.Errorf("expected 0 enqueued jobs, got %d", len(enqueued))
				}
			}
		})
	}
}
