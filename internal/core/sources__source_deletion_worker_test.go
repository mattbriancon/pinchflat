package core_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestSourceDeletionWorker_Kickoff(t *testing.T) {
	t.Run("starts the worker", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		source := coretest.SourceFixture(t, ta, core.Attrs{})

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.SourceDeletionWorkerName})
		if len(enqueued) != 0 {
			t.Errorf("expected 0 enqueued jobs, got %d", len(enqueued))
		}

		job, err := ta.App.SourceDeletionWorkerKickoff(ta.Ctx, source, core.Attrs{}, core.KW{})
		if err != nil {
			t.Errorf("kickoff failed: %v", err)
		}
		if job == nil {
			t.Error("expected job, got nil")
		}

		enqueued = ta.Oban.Enqueued(t, obanlite.Match{Worker: core.SourceDeletionWorkerName})
		if len(enqueued) != 1 {
			t.Errorf("expected 1 enqueued job, got %d", len(enqueued))
		}
	})

	t.Run("can be called with additional job arguments", func(t *testing.T) {
		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		source := coretest.SourceFixture(t, ta, core.Attrs{})

		jobArgs := core.Attrs{"delete_files": true}

		job, err := ta.App.SourceDeletionWorkerKickoff(ta.Ctx, source, jobArgs, core.KW{})
		if err != nil {
			t.Errorf("kickoff failed: %v", err)
		}
		if job == nil {
			t.Error("expected job, got nil")
		}

		enqueued := ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.SourceDeletionWorkerName,
			Args:   map[string]any{"id": source.ID, "delete_files": true},
		})
		if enqueued == nil {
			t.Error("expected enqueued job not found")
		}
	})
}

func TestSourceDeletionWorker_Perform(t *testing.T) {
	t.Run("deletes the source but leaves the files", func(t *testing.T) {
		t.Skip("BLOCKED: MediaCreateMediaItem unported")

		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{
			"source_id": source.ID,
		})

		err := ta.Oban.PerformJob(ta.Ctx, core.SourceDeletionWorkerName, map[string]any{"id": source.ID})
		if err != nil {
			t.Errorf("perform job failed: %v", err)
		}

		// Check that source is deleted
		_, err = ta.App.SourcesGetSource(ta.Ctx, source.ID)
		if err != core.ErrNotFound {
			t.Errorf("expected ErrNotFound for deleted source, got: %v", err)
		}

		// Check that media_item is deleted
		_, err = ta.App.MediaGetMediaItem(ta.Ctx, mediaItem.ID)
		if err != core.ErrNotFound {
			t.Errorf("expected ErrNotFound for deleted media_item, got: %v", err)
		}

		// Check that media file still exists
		if _, err := os.Stat(*mediaItem.MediaFilepath); os.IsNotExist(err) {
			t.Error("expected media file to exist")
		}
	})

	t.Run("deletes the source and files if specified", func(t *testing.T) {
		t.Skip("BLOCKED: MediaCreateMediaItem unported")

		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{
			"source_id": source.ID,
		})

		mediaFilepath := *mediaItem.MediaFilepath

		err := ta.Oban.PerformJob(ta.Ctx, core.SourceDeletionWorkerName, map[string]any{
			"id":           source.ID,
			"delete_files": true,
		})
		if err != nil {
			t.Errorf("perform job failed: %v", err)
		}

		// Check that source is deleted
		_, err = ta.App.SourcesGetSource(ta.Ctx, source.ID)
		if err != core.ErrNotFound {
			t.Errorf("expected ErrNotFound for deleted source, got: %v", err)
		}

		// Check that media_item is deleted
		_, err = ta.App.MediaGetMediaItem(ta.Ctx, mediaItem.ID)
		if err != core.ErrNotFound {
			t.Errorf("expected ErrNotFound for deleted media_item, got: %v", err)
		}

		// Check that media file is deleted
		if _, err := os.Stat(mediaFilepath); !os.IsNotExist(err) {
			t.Error("expected media file to not exist")
		}
	})
}
