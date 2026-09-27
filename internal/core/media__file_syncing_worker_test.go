package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestFileSyncingWorker_KickoffWithTask(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)

	if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FileSyncingWorkerName})) != 0 {
		t.Fatalf("expected 0 enqueued jobs initially")
	}

	source := coretest.SourceFixture(t, ta, core.Attrs{})
	task, err := ta.FileSyncingWorkerKickoffWithTask(ta.Ctx, source, core.KW{})
	if err != nil {
		t.Fatalf("KickoffWithTask failed: %v", err)
	}

	if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FileSyncingWorkerName})) != 1 {
		t.Errorf("expected 1 enqueued job")
	}

	if task.SourceID == nil || *task.SourceID != source.ID {
		t.Errorf("expected task.SourceID %d, got %v", source.ID, task.SourceID)
	}
}

func TestFileSyncingWorker_Perform(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)

	source := coretest.SourceFixture(t, ta, core.Attrs{})
	mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{
		"media_filepath": "/tmp/missing.mp4",
		"source_id":      source.ID,
	})

	if err := ta.Oban.PerformJob(ta.Ctx, core.FileSyncingWorkerName, map[string]any{"id": source.ID}); err != nil {
		t.Fatalf("PerformJob failed: %v", err)
	}

	updatedMediaItem, err := ta.MediaGetMediaItem(ta.Ctx, mediaItem.ID)
	if err != nil {
		t.Fatalf("reload media item failed: %v", err)
	}

	if updatedMediaItem.MediaFilepath != nil {
		t.Error("media_filepath should be nil after sync")
	}
}
