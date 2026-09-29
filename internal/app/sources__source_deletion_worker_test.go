package app_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestSourceDeletionWorker_Kickoff(t *testing.T) {
	t.Run("starts worker", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: app.SourceDeletionWorkerName})) != 0 {
			t.Errorf("expected 0 enqueued initially")
		}

		job, err := ta.App.SourceDeletionWorkerKickoff(ta.Ctx, source, map[string]any{})
		if err != nil {
			t.Errorf("kickoff failed: %v", err)
		}
		if job == nil {
			t.Error("expected job")
		}

		if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: app.SourceDeletionWorkerName})) != 1 {
			t.Errorf("expected 1 enqueued job")
		}
	})

	t.Run("passes job arguments", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		jobArgs := map[string]any{"delete_files": true}

		job, err := ta.App.SourceDeletionWorkerKickoff(ta.Ctx, source, jobArgs)
		if err != nil {
			t.Errorf("kickoff failed: %v", err)
		}
		if job == nil {
			t.Error("expected job")
		}

		enqueued := ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: app.SourceDeletionWorkerName,
			Args:   map[string]any{"id": source.ID, "delete_files": true},
		})
		if enqueued == nil {
			t.Error("job not enqueued as expected")
		}
	})
}

func TestSourceDeletionWorker_Perform(t *testing.T) {
	t.Run("without deleting files", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		err := ta.Oban.PerformJob(ta.Ctx, app.SourceDeletionWorkerName, map[string]any{"id": source.ID})
		if err != nil {
			t.Errorf("perform job failed: %v", err)
		}

		_, err = ta.App.GetSource(ta.Ctx, source.ID)
		if err != store.ErrNotFound {
			t.Errorf("source should be deleted")
		}

		_, err = ta.App.GetMediaItem(ta.Ctx, mediaItem.ID)
		if err != store.ErrNotFound {
			t.Errorf("media_item should be deleted")
		}

		if _, err := os.Stat(*mediaItem.MediaFilepath); os.IsNotExist(err) {
			t.Error("media file should exist")
		}
	})

	t.Run("with deleting files", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		mediaFilepath := *mediaItem.MediaFilepath

		err := ta.Oban.PerformJob(ta.Ctx, app.SourceDeletionWorkerName, map[string]any{
			"id":           source.ID,
			"delete_files": true,
		})
		if err != nil {
			t.Errorf("perform job failed: %v", err)
		}

		_, err = ta.App.GetSource(ta.Ctx, source.ID)
		if err != store.ErrNotFound {
			t.Errorf("source should be deleted")
		}

		_, err = ta.App.GetMediaItem(ta.Ctx, mediaItem.ID)
		if err != store.ErrNotFound {
			t.Errorf("media_item should be deleted")
		}

		if _, err := os.Stat(mediaFilepath); !os.IsNotExist(err) {
			t.Error("media file should not exist")
		}
	})
}
