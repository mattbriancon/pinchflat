package app_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestDownloadingHelpers_EnqueuePendingDownloadTasks(t *testing.T) {
	t.Parallel()

	t.Run("enqueues a job for each pending media item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		if err := ta.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, nil); err != nil {
			t.Fatalf("DownloadingHelpersEnqueuePendingDownloadTasks failed: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})
	})

	t.Run("does not enqueue a job for media items with a filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		_ = apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), MediaFilepath: store.Ptr("some/filepath.mp4")})

		if err := ta.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, nil); err != nil {
			t.Fatalf("DownloadingHelpersEnqueuePendingDownloadTasks failed: %v", err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
	})

	t.Run("attaches a task to each enqueued job", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		tasks, err := ta.ListTasksFor(ctx, mediaItem, nil, []string{obanlite.StateAvailable})
		if err != nil {
			t.Fatalf("TasksListTasksFor failed: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}

		if err := ta.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, nil); err != nil {
			t.Fatalf("DownloadingHelpersEnqueuePendingDownloadTasks failed: %v", err)
		}

		tasks, err = ta.ListTasksFor(ctx, mediaItem, nil, []string{obanlite.StateAvailable})
		if err != nil {
			t.Fatalf("TasksListTasksFor failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("expected 1 task, got %d", len(tasks))
		}
	})

	t.Run("does not create a job if the source is set to not download", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{DownloadMedia: store.Ptr(false)})

		if err := ta.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, nil); err != nil {
			t.Fatalf("DownloadingHelpersEnqueuePendingDownloadTasks failed: %v", err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
	})

	t.Run("does not attach tasks if the source is set to not download", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{DownloadMedia: store.Ptr(false)})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		if err := ta.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, nil); err != nil {
			t.Fatalf("DownloadingHelpersEnqueuePendingDownloadTasks failed: %v", err)
		}

		tasks, err := ta.ListTasksFor(ctx, mediaItem, nil, []string{obanlite.StateAvailable})
		if err != nil {
			t.Fatalf("TasksListTasksFor failed: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("can pass job options", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		priority := 1
		if err := ta.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, &priority); err != nil {
			t.Fatalf("DownloadingHelpersEnqueuePendingDownloadTasks failed: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}, Priority: &priority})
	})
}

func TestDownloadingHelpers_DequeuePendingDownloadTasks(t *testing.T) {
	t.Parallel()

	t.Run("deletes all pending tasks for a source's media items", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		if err := ta.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, nil); err != nil {
			t.Fatalf("DownloadingHelpersEnqueuePendingDownloadTasks failed: %v", err)
		}
		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})

		if err := ta.DownloadingHelpersDequeuePendingDownloadTasks(ctx, source); err != nil {
			t.Fatalf("DownloadingHelpersDequeuePendingDownloadTasks failed: %v", err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		tasks, err := ta.ListTasksFor(ctx, mediaItem, nil, []string{obanlite.StateAvailable})
		if err != nil {
			t.Fatalf("TasksListTasksFor failed: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})
}

func TestDownloadingHelpers_KickoffDownloadIfPending(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*apptest.TestApp, *store.MediaItem) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		return ta, mediaItem
	}

	t.Run("enqueues a download job", func(t *testing.T) {
		ta, mediaItem := setup(t)
		ctx := ta.Ctx

		_, err := ta.DownloadingHelpersKickoffDownloadIfPending(ctx, mediaItem, nil)
		if err != nil {
			t.Fatalf("DownloadingHelpersKickoffDownloadIfPending failed: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})
	})

	t.Run("creates and returns a download task record", func(t *testing.T) {
		ta, mediaItem := setup(t)
		ctx := ta.Ctx

		task, err := ta.DownloadingHelpersKickoffDownloadIfPending(ctx, mediaItem, nil)
		if err != nil {
			t.Fatalf("DownloadingHelpersKickoffDownloadIfPending failed: %v", err)
		}

		tasks, err := ta.ListTasksFor(ctx, mediaItem, nil, []string{obanlite.StateAvailable})
		if err != nil {
			t.Fatalf("TasksListTasksFor failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("expected 1 task, got %d", len(tasks))
		}
		foundTask := tasks[0]
		if task.ID != foundTask.ID {
			t.Errorf("expected task ID %d, got %d", task.ID, foundTask.ID)
		}
	})

	t.Run("does not enqueue a download job if the source does not allow it", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{DownloadMedia: store.Ptr(false)})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		_, err := ta.DownloadingHelpersKickoffDownloadIfPending(ctx, mediaItem, nil)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != app.ErrShouldNotDownload.Error() {
			t.Errorf("expected error %v, got %v", app.ErrShouldNotDownload, err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
	})

	t.Run("does not enqueue a download job if the media item does not match the format rules", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{LivestreamBehaviour: store.Ptr(store.MediaProfileLivestreamBehaviourExclude)})
		source := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(profile.ID)})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Livestream: store.Ptr(true), Clear: store.ClearMediaFilepath})

		_, err := ta.DownloadingHelpersKickoffDownloadIfPending(ctx, mediaItem, nil)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != app.ErrShouldNotDownload.Error() {
			t.Errorf("expected error %v, got %v", app.ErrShouldNotDownload, err)
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
	})

	t.Run("can pass job options", func(t *testing.T) {
		ta, mediaItem := setup(t)
		ctx := ta.Ctx

		priority := 1
		_, err := ta.DownloadingHelpersKickoffDownloadIfPending(ctx, mediaItem, &priority)
		if err != nil {
			t.Fatalf("DownloadingHelpersKickoffDownloadIfPending failed: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}, Priority: &priority})
	})
}

func TestDownloadingHelpers_KickoffRedownloadForExistingMedia(t *testing.T) {
	t.Run("enqueues a download job for each downloaded media item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), MediaFilepath: store.Ptr("some/filepath.mp4")})

		results, err := ta.DownloadingHelpersKickoffRedownloadForExistingMedia(ctx, source)
		if err != nil {
			t.Fatalf("DownloadingHelpersKickoffRedownloadForExistingMedia failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})
	})

	t.Run("doesn't enqueue jobs for media that should be ignored", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		otherSource := apptest.SourceFixture(t, ta, store.SourceParams{})
		_ = apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		_ = apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(otherSource.ID), MediaFilepath: store.Ptr("some/filepath.mp4")})
		_ = apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), MediaFilepath: store.Ptr("some/filepath.mp4"), PreventDownload: store.Ptr(true)})

		results, err := ta.DownloadingHelpersKickoffRedownloadForExistingMedia(ctx, source)
		if err != nil {
			t.Fatalf("DownloadingHelpersKickoffRedownloadForExistingMedia failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}

		ta.Oban.RefuteEnqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
	})
}
