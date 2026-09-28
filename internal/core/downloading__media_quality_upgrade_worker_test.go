package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
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
			ta := coretest.NewApp(t)
			ctx := ta.Ctx

			mediaProfile := coretest.MediaProfileFixture(t, ta, store.Attrs{"redownload_delay_days": 4})
			source := coretest.SourceFixture(t, ta, store.Attrs{
				"media_profile_id": mediaProfile.ID,
				"inserted_at":      coretest.NowMinus(10, "days"),
			})

			mediaItem := coretest.MediaItemFixture(t, ta, store.Attrs{
				"source_id":           source.ID,
				"uploaded_at":         coretest.NowMinus(6, "days"),
				"media_downloaded_at": coretest.NowMinus(tt.downloadedDays, "days"),
			})

			err := ta.Oban.PerformJob(ctx, core.MediaQualityUpgradeWorkerName, map[string]any{})
			if err != nil {
				t.Fatalf("PerformJob failed: %v", err)
			}

			enqueued := ta.Oban.Enqueued(t, obanlite.Match{
				Worker: core.MediaDownloadWorkerName,
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
