package app_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestFileSyncingWorker_KickoffWithTask(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)

	if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FileSyncingWorkerName})) != 0 {
		t.Fatalf("expected 0 enqueued jobs initially")
	}

	source := apptest.SourceFixture(t, ta, store.Attrs{})
	task, err := ta.FileSyncingWorkerKickoffWithTask(ta.Ctx, source)
	if err != nil {
		t.Fatalf("KickoffWithTask failed: %v", err)
	}

	if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FileSyncingWorkerName})) != 1 {
		t.Errorf("expected 1 enqueued job")
	}

	if task.SourceID == nil || *task.SourceID != source.ID {
		t.Errorf("expected task.SourceID %d, got %v", source.ID, task.SourceID)
	}
}

func TestFileSyncingWorker_Perform(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)

	source := apptest.SourceFixture(t, ta, store.Attrs{})
	mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{
		"media_filepath": "/tmp/missing.mp4",
		"source_id":      source.ID,
	})

	if err := ta.Oban.PerformJob(ta.Ctx, app.FileSyncingWorkerName, map[string]any{"id": source.ID}); err != nil {
		t.Fatalf("PerformJob failed: %v", err)
	}

	updatedMediaItem, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID)
	if err != nil {
		t.Fatalf("reload media item failed: %v", err)
	}

	if updatedMediaItem.MediaFilepath != nil {
		t.Error("media_filepath should be nil after sync")
	}
}
