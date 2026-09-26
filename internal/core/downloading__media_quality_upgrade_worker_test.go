package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestMediaQualityUpgradeWorker_Perform(t *testing.T) {
	t.Run("kicks off a task for redownloadable media items", func(t *testing.T) {
		t.Skip("BLOCKED: MediaListUpgradeableMediaItems, MediaDownloadWorkerKickoffWithTask unported")
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"redownload_delay_days": 4})
		source := coretest.SourceFixture(t, ta, core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"inserted_at":      coretest.NowMinus(10, "days"),
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{
			"source_id":           source.ID,
			"uploaded_at":         coretest.NowMinus(6, "days"),
			"media_downloaded_at": coretest.NowMinus(5, "days"),
		})

		err := ta.Oban.PerformJob(ctx, core.MediaQualityUpgradeWorkerName, map[string]any{})
		if err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{
			Worker: core.MediaDownloadWorkerName,
			Args:   map[string]any{"id": mediaItem.ID, "quality_upgrade?": true},
		})
		if len(enqueued) != 1 {
			t.Errorf("expected 1 enqueued job, got %d", len(enqueued))
		}
	})

	t.Run("does not kickoff a task for non-redownloadable media items", func(t *testing.T) {
		t.Skip("BLOCKED: MediaListUpgradeableMediaItems unported")
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"redownload_delay_days": 4})
		source := coretest.SourceFixture(t, ta, core.Attrs{
			"media_profile_id": mediaProfile.ID,
			"inserted_at":      coretest.NowMinus(10, "days"),
		})

		coretest.MediaItemFixture(t, ta, core.Attrs{
			"source_id":           source.ID,
			"uploaded_at":         coretest.NowMinus(6, "days"),
			"media_downloaded_at": coretest.NowMinus(1, "day"),
		})

		err := ta.Oban.PerformJob(ctx, core.MediaQualityUpgradeWorkerName, map[string]any{})
		if err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(enqueued) != 0 {
			t.Errorf("expected 0 enqueued jobs, got %d", len(enqueued))
		}
	})
}
