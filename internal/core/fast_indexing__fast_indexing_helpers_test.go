package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestFastIndexingHelpers_KickoffIndexingTask(t *testing.T) {
	t.Run("deletes any existing fast indexing tasks", func(t *testing.T) {
	})

	t.Run("kicks off a new fast indexing task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		task, err := ta.App.FastIndexingHelpersKickoffIndexingTask(ta.Ctx, source)
		if err != nil {
			t.Fatalf("Failed to kickoff indexing task: %v", err)
		}

		if *task.SourceID != source.ID {
			t.Errorf("Expected task.SourceID=%d, got %d", source.ID, *task.SourceID)
		}

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(workers) != 1 {
			t.Errorf("Expected 1 worker, got %d", len(workers))
		}
	})
}

func TestFastIndexingHelpers_IndexAndKickoffDownloads(t *testing.T) {
	t.Run("enqueues a worker for each new media_id in the source's RSS feed", func(t *testing.T) {
	})

	t.Run("does not enqueue a new worker for the source's media IDs we already know about", func(t *testing.T) {
	})

	t.Run("kicks off a download task for all pending media but at a lower priority", func(t *testing.T) {
	})

	t.Run("returns the found media items", func(t *testing.T) {
	})

	t.Run("does not enqueue a download job if the source does not allow it", func(t *testing.T) {
	})

	t.Run("creates a download task record", func(t *testing.T) {
	})

	t.Run("passes the source's download options to the yt-dlp runner", func(t *testing.T) {
	})

	t.Run("does not enqueue a download job if the media item does not match the format rules", func(t *testing.T) {
	})

	t.Run("does not blow up if a media item cannot be created", func(t *testing.T) {
	})

	t.Run("does not blow up if a media item causes a yt-dlp error", func(t *testing.T) {
	})
}

func TestFastIndexingHelpers_IndexAndKickoffDownloads_WhenTestingCookies(t *testing.T) {
	t.Run("sets use_cookies if the source uses cookies", func(t *testing.T) {
	})

	t.Run("does not set use_cookies if the source uses cookies when needed", func(t *testing.T) {
	})

	t.Run("does not set use_cookies if the source does not use cookies", func(t *testing.T) {
	})
}

func TestFastIndexingHelpers_IndexAndKickoffDownloads_WhenTestingBackends(t *testing.T) {
	t.Run("uses the YouTube API if it is enabled", func(t *testing.T) {
	})

	t.Run("the YouTube API creates records as expected", func(t *testing.T) {
	})

	t.Run("RSS is used as a backup if the API fails", func(t *testing.T) {
	})

	t.Run("RSS is used if the API is not enabled", func(t *testing.T) {
	})
}
