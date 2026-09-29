package app_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestMediaProfileDeletionWorker_Kickoff(t *testing.T) {
	t.Run("starts worker", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})

		if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaProfileDeletionWorkerName})) != 0 {
			t.Errorf("expected 0 enqueued initially")
		}

		job, err := ta.App.MediaProfileDeletionWorkerKickoff(ta.Ctx, profile, store.Attrs{}, store.KW{})
		if err != nil {
			t.Errorf("kickoff failed: %v", err)
		}
		if job == nil {
			t.Error("expected job")
		}

		if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaProfileDeletionWorkerName})) != 1 {
			t.Errorf("expected 1 enqueued job")
		}
	})

	t.Run("passes job arguments", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		jobArgs := store.Attrs{"delete_files": true}

		job, err := ta.App.MediaProfileDeletionWorkerKickoff(ta.Ctx, profile, jobArgs, store.KW{})
		if err != nil {
			t.Errorf("kickoff failed: %v", err)
		}
		if job == nil {
			t.Error("expected job")
		}

		enqueued := ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: app.MediaProfileDeletionWorkerName,
			Args:   map[string]any{"id": profile.ID, "delete_files": true},
		})
		if enqueued == nil {
			t.Error("job not enqueued as expected")
		}
	})
}

func TestMediaProfileDeletionWorker_Perform(t *testing.T) {
	t.Run("without deleting files", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		source := apptest.SourceFixture(t, ta, store.Attrs{
			"media_profile_id": profile.ID,
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{
			"source_id": source.ID,
		})

		err := ta.Oban.PerformJob(ta.Ctx, app.MediaProfileDeletionWorkerName, map[string]any{"id": profile.ID})
		if err != nil {
			t.Errorf("perform job failed: %v", err)
		}

		_, err = ta.App.GetMediaProfile(ta.Ctx, profile.ID)
		if err != store.ErrNotFound {
			t.Errorf("profile should be deleted")
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

		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		source := apptest.SourceFixture(t, ta, store.Attrs{
			"media_profile_id": profile.ID,
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.Attrs{
			"source_id": source.ID,
		})

		mediaFilepath := *mediaItem.MediaFilepath

		err := ta.Oban.PerformJob(ta.Ctx, app.MediaProfileDeletionWorkerName, map[string]any{
			"id":           profile.ID,
			"delete_files": true,
		})
		if err != nil {
			t.Errorf("perform job failed: %v", err)
		}

		_, err = ta.App.GetMediaProfile(ta.Ctx, profile.ID)
		if err != store.ErrNotFound {
			t.Errorf("profile should be deleted")
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
