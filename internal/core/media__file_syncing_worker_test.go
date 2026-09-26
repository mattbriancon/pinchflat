package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestFileSyncingWorker_KickoffWithTask(t *testing.T) {
	t.Run("starts the worker", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		source := coretest.SourceFixture(t, ta, core.Attrs{})

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FileSyncingWorkerName})
		if len(enqueued) != 0 {
			t.Errorf("expected 0 enqueued jobs initially, got %d", len(enqueued))
		}

		_, err := ta.FileSyncingWorkerKickoffWithTask(ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("KickoffWithTask failed: %v", err)
		}

		enqueued = ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FileSyncingWorkerName})
		if len(enqueued) != 1 {
			t.Errorf("expected 1 enqueued job, got %d", len(enqueued))
		}
	})

	t.Run("attaches a task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		source := coretest.SourceFixture(t, ta, core.Attrs{})

		task, err := ta.FileSyncingWorkerKickoffWithTask(ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("KickoffWithTask failed: %v", err)
		}

		if task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("expected task.SourceID to be %d, got %v", source.ID, task.SourceID)
		}
	})
}

func TestFileSyncingWorker_Perform(t *testing.T) {
	t.Run("syncs file presence on disk", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{
			"media_filepath": "/tmp/missing.mp4",
			"source_id":      source.ID,
		})

		if err := ta.Oban.PerformJob(ctx, core.FileSyncingWorkerName, map[string]any{"id": source.ID}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		updatedMediaItem, err := ta.MediaGetMediaItem(ctx, mediaItem.ID)
		if err != nil {
			t.Fatalf("reload media item: %v", err)
		}

		if updatedMediaItem.MediaFilepath != nil {
			t.Error("media_filepath should be nil after sync")
		}
	})
}
