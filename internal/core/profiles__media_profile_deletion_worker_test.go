package core_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestMediaProfileDeletionWorker_Kickoff(t *testing.T) {
	t.Run("starts worker", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{})

		if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaProfileDeletionWorkerName})) != 0 {
			t.Errorf("expected 0 enqueued initially")
		}

		job, err := ta.App.MediaProfileDeletionWorkerKickoff(ta.Ctx, profile, core.Attrs{}, core.KW{})
		if err != nil {
			t.Errorf("kickoff failed: %v", err)
		}
		if job == nil {
			t.Error("expected job")
		}

		if len(ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaProfileDeletionWorkerName})) != 1 {
			t.Errorf("expected 1 enqueued job")
		}
	})

	t.Run("passes job arguments", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		jobArgs := core.Attrs{"delete_files": true}

		job, err := ta.App.MediaProfileDeletionWorkerKickoff(ta.Ctx, profile, jobArgs, core.KW{})
		if err != nil {
			t.Errorf("kickoff failed: %v", err)
		}
		if job == nil {
			t.Error("expected job")
		}

		enqueued := ta.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: core.MediaProfileDeletionWorkerName,
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
		ta := coretest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		source := coretest.SourceFixture(t, ta, core.Attrs{
			"media_profile_id": profile.ID,
		})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{
			"source_id": source.ID,
		})

		err := ta.Oban.PerformJob(ta.Ctx, core.MediaProfileDeletionWorkerName, map[string]any{"id": profile.ID})
		if err != nil {
			t.Errorf("perform job failed: %v", err)
		}

		_, err = ta.App.ProfilesGetMediaProfile(ta.Ctx, profile.ID)
		if err != core.ErrNotFound {
			t.Errorf("profile should be deleted")
		}

		_, err = ta.App.SourcesGetSource(ta.Ctx, source.ID)
		if err != core.ErrNotFound {
			t.Errorf("source should be deleted")
		}

		_, err = ta.App.MediaGetMediaItem(ta.Ctx, mediaItem.ID)
		if err != core.ErrNotFound {
			t.Errorf("media_item should be deleted")
		}

		if _, err := os.Stat(*mediaItem.MediaFilepath); os.IsNotExist(err) {
			t.Error("media file should exist")
		}
	})

	t.Run("with deleting files", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{})
		source := coretest.SourceFixture(t, ta, core.Attrs{
			"media_profile_id": profile.ID,
		})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{
			"source_id": source.ID,
		})

		mediaFilepath := *mediaItem.MediaFilepath

		err := ta.Oban.PerformJob(ta.Ctx, core.MediaProfileDeletionWorkerName, map[string]any{
			"id":           profile.ID,
			"delete_files": true,
		})
		if err != nil {
			t.Errorf("perform job failed: %v", err)
		}

		_, err = ta.App.ProfilesGetMediaProfile(ta.Ctx, profile.ID)
		if err != core.ErrNotFound {
			t.Errorf("profile should be deleted")
		}

		_, err = ta.App.SourcesGetSource(ta.Ctx, source.ID)
		if err != core.ErrNotFound {
			t.Errorf("source should be deleted")
		}

		_, err = ta.App.MediaGetMediaItem(ta.Ctx, mediaItem.ID)
		if err != core.ErrNotFound {
			t.Errorf("media_item should be deleted")
		}

		if _, err := os.Stat(mediaFilepath); !os.IsNotExist(err) {
			t.Error("media file should not exist")
		}
	})
}
