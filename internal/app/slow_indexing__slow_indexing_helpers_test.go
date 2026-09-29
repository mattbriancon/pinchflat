package app_test

import (
	"os"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

func TestSlowIndexingHelpers_KickoffIndexingTask(t *testing.T) {
	t.Run("schedules a job", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(1)})

		task, err := ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, map[string]any{})
		must(t, err)
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{
			Worker: app.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if len(jobs) == 0 {
			t.Fatal("expected job to be enqueued")
		}
	})

	t.Run("schedules a job for the future based on when the source was last indexed", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			IndexFrequencyMinutes: store.Ptr(30),
			LastIndexedAt:         store.Ptr(apptest.NowMinus(5, "minutes")),
		})

		task, err := ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, map[string]any{})
		must(t, err)
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{
			Worker: app.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(jobs))
		}

		// Should be scheduled approximately 25 minutes in the future
		diff := jobs[0].ScheduledAt.Sub(apptest.Now())
		if diff < 24*time.Minute || diff > 26*time.Minute {
			t.Errorf("expected scheduled in ~25 minutes, got %v", diff)
		}
	})

	t.Run("schedules a job immediately if the source was indexed far in the past", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			IndexFrequencyMinutes: store.Ptr(30),
			LastIndexedAt:         store.Ptr(apptest.NowMinus(60, "minutes")),
		})

		task, err := ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, map[string]any{})
		must(t, err)
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(jobs))
		}

		// Should be scheduled immediately (within 1 second)
		diff := jobs[0].ScheduledAt.Sub(apptest.Now())
		if diff < -1*time.Second || diff > 1*time.Second {
			t.Errorf("expected scheduled immediately, got %v", diff)
		}
	})

	t.Run("schedules a job immediately if the source has never been indexed", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			IndexFrequencyMinutes: store.Ptr(30),
			Clear:                 store.ClearLastIndexedAt,
		})

		task, err := ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, map[string]any{})
		must(t, err)
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(jobs))
		}

		// Should be scheduled immediately
		diff := jobs[0].ScheduledAt.Sub(apptest.Now())
		if diff < -1*time.Second || diff > 1*time.Second {
			t.Errorf("expected scheduled immediately, got %v", diff)
		}
	})

	t.Run("schedules a job immediately if the user is forcing an index", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			IndexFrequencyMinutes: store.Ptr(30),
			LastIndexedAt:         store.Ptr(apptest.NowMinus(5, "minutes")),
		})

		task, err := ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, map[string]any{"force": true})
		must(t, err)
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(jobs))
		}

		// Should be scheduled immediately
		diff := jobs[0].ScheduledAt.Sub(apptest.Now())
		if diff < -1*time.Second || diff > 1*time.Second {
			t.Errorf("expected scheduled immediately, got %v", diff)
		}
	})

	t.Run("creates and attaches a task", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(1)})

		task, err := ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, map[string]any{})
		must(t, err)

		if *task.SourceID != source.ID {
			t.Errorf("expected task.SourceID to be %d, got %d", source.ID, task.SourceID)
		}
	})

	t.Run("deletes any pending media collection tasks for the source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: app.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		must(t, err)
		task := apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(source.ID), JobID: store.Ptr(job.ID)})

		_, err = ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, map[string]any{})
		must(t, err)

		// The old task should be deleted
		_, err = ta.App.GetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Fatal("expected error reloading task, but got nil")
		}
	})

	t.Run("deletes any executing media collection tasks for the source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: app.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		must(t, err)
		task := apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(source.ID), JobID: store.Ptr(job.ID)})

		// Mark the job as executing
		mustOK(t)(ta.DB.ExecContext(ta.Ctx, `UPDATE oban_jobs SET state = 'executing' WHERE id = ?`, job.ID))

		_, err = ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, map[string]any{})
		must(t, err)

		// The task should be deleted
		_, err = ta.App.GetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Fatal("expected error reloading task, but got nil")
		}
	})

	t.Run("can be called with additional job arguments", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(1)})
		jobArgs := map[string]any{"force": true}

		_, err := ta.App.SlowIndexingHelpersKickoffIndexingTask(ta.Ctx, source, jobArgs)
		must(t, err)

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID, "force": true}})
		if len(jobs) == 0 {
			t.Fatal("expected job to be enqueued with force argument")
		}
	})
}

func TestSlowIndexingHelpers_DeleteIndexingTasks(t *testing.T) {
	t.Run("deletes slow indexing tasks for the source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: app.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		must(t, err)
		_ = apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(source.ID), JobID: store.Ptr(job.ID)})

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName})
		if len(jobs) == 0 {
			t.Fatal("expected job to be enqueued")
		}

		err = ta.App.SlowIndexingHelpersDeleteIndexingTasks(ta.Ctx, source, false)
		must(t, err)

		jobs = ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName})
		if len(jobs) != 0 {
			t.Fatalf("expected no jobs, got %d", len(jobs))
		}
	})

	t.Run("deletes fast indexing tasks for the source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: app.FastIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		must(t, err)
		_ = apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(source.ID), JobID: store.Ptr(job.ID)})

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
		if len(jobs) == 0 {
			t.Fatal("expected job to be enqueued")
		}

		err = ta.App.SlowIndexingHelpersDeleteIndexingTasks(ta.Ctx, source, false)
		must(t, err)

		jobs = ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
		if len(jobs) != 0 {
			t.Fatalf("expected no jobs, got %d", len(jobs))
		}
	})

	t.Run("doesn't normally delete currently executing tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: app.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		must(t, err)
		task := apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(source.ID), JobID: store.Ptr(job.ID)})

		mustOK(t)(ta.DB.ExecContext(ta.Ctx, `UPDATE oban_jobs SET state = 'executing' WHERE id = ?`, job.ID))

		reloadedTask, err := ta.App.GetTaskBang(ta.Ctx, task.ID)
		must(t, err)

		err = ta.App.SlowIndexingHelpersDeleteIndexingTasks(ta.Ctx, source, false)
		must(t, err)

		reloadedTask, err = ta.App.GetTaskBang(ta.Ctx, reloadedTask.ID)
		must(t, err)
	})

	t.Run("can optionally delete currently executing tasks", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: app.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		must(t, err)
		task := apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(source.ID), JobID: store.Ptr(job.ID)})

		mustOK(t)(ta.DB.ExecContext(ta.Ctx, `UPDATE oban_jobs SET state = 'executing' WHERE id = ?`, job.ID))

		reloadedTask, err := ta.App.GetTaskBang(ta.Ctx, task.ID)
		must(t, err)

		err = ta.App.SlowIndexingHelpersDeleteIndexingTasks(ta.Ctx, source, true)
		must(t, err)

		_, err = ta.App.GetTaskBang(ta.Ctx, reloadedTask.ID)
		if err == nil {
			t.Fatal("expected error reloading task, but got nil")
		}
	})
}

func TestSlowIndexingHelpers_IndexAndEnqueueDownloadForMediaItems(t *testing.T) {
	t.Run("creates a media_item record for each media ID returned", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return apptest.SourceAttributesReturnFixture(), nil
		})

		result, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		mediaItems := 0
		for _, item := range result {
			if mi, ok := item.(*store.MediaItem); ok {
				mediaItems++
				_ = mi
			}
		}

		if mediaItems != 3 {
			t.Errorf("expected 3 media items, got %d", mediaItems)
		}
	})

	t.Run("attaches all media_items to the given source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return apptest.SourceAttributesReturnFixture(), nil
		})

		result, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		for _, item := range result {
			if mi, ok := item.(*store.MediaItem); ok {
				if mi.SourceID != source.ID {
					t.Errorf("expected source_id to be %d, got %d", source.ID, mi.SourceID)
				}
			}
		}
	})

	t.Run("won't duplicate media_items based on media_id and source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		_, err = ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		// Check that we still have only 3 media items
		q := store.From[store.MediaItem]("mi").Where(map[string]interface{}{"mi.source_id": source.ID})
		items, err := store.All[store.MediaItem](ta.Ctx, ta.App.Q(ta.Ctx), q)
		must(t, err)

		if len(items) != 3 {
			t.Errorf("expected 3 media items, got %d", len(items))
		}
	})

	t.Run("can duplicate media_ids for different sources", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source1 := apptest.SourceFixture(t, ta, store.SourceParams{})
		source2 := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return apptest.SourceAttributesReturnFixture(), nil
		})

		result1, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source1, false)
		must(t, err)

		result2, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source2, false)
		must(t, err)

		mediaItems1 := 0
		mediaItems2 := 0
		for _, item := range result1 {
			if _, ok := item.(*store.MediaItem); ok {
				mediaItems1++
			}
		}
		for _, item := range result2 {
			if _, ok := item.(*store.MediaItem); ok {
				mediaItems2++
			}
		}

		if mediaItems1 != 3 || mediaItems2 != 3 {
			t.Errorf("expected 3 media items for each source, got %d and %d", mediaItems1, mediaItems2)
		}
	})

	t.Run("returns a list of media_items", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return apptest.SourceAttributesReturnFixture(), nil
		})

		result1, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		result2, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		ids1 := make([]int64, 0)
		ids2 := make([]int64, 0)

		for _, item := range result1 {
			if mi, ok := item.(*store.MediaItem); ok {
				ids1 = append(ids1, mi.ID)
			}
		}

		for _, item := range result2 {
			if mi, ok := item.(*store.MediaItem); ok {
				ids2 = append(ids2, mi.ID)
			}
		}

		if len(ids1) != len(ids2) {
			t.Errorf("expected same number of items, got %d and %d", len(ids1), len(ids2))
		}

		for i := 0; i < len(ids1); i++ {
			if ids1[i] != ids2[i] {
				t.Errorf("expected ids to match at index %d, got %d != %d", i, ids1[i], ids2[i])
			}
		}
	})

	t.Run("updates the source's last_indexed_at field", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		if source.LastIndexedAt != nil {
			t.Errorf("expected last_indexed_at to be nil initially, got %v", source.LastIndexedAt)
		}

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		reloadedSource, err := ta.App.GetSource(ta.Ctx, source.ID)
		must(t, err)

		if reloadedSource.LastIndexedAt == nil {
			t.Fatal("expected last_indexed_at to be set")
		}

		diff := time.Now().UTC().Sub(reloadedSource.LastIndexedAt.Time)
		if diff < 0 || diff > 2*time.Second {
			t.Errorf("expected last_indexed_at to be recent, got diff %v", diff)
		}
	})

	t.Run("enqueues a job for each pending media item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})
		if len(jobs) == 0 {
			t.Fatal("expected job to be enqueued for pending media item")
		}
	})

	t.Run("does not attach tasks if the source is set to not download", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{DownloadMedia: store.Ptr(false)})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		tasks, err := ta.App.ListTasksFor(ta.Ctx, mediaItem, nil, nil)
		must(t, err)
		if len(tasks) != 0 {
			t.Errorf("expected no tasks for media item, got %d", len(tasks))
		}
	})

	t.Run("doesn't blow up if a media item cannot be coerced into a struct", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		response := `{"id":"video3","title":"Video 3","live_status":"not_live","description":"desc3","original_url":null,"aspect_ratio":null,"duration":null,"upload_date":null}`

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return response, nil
		})

		result, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		if _, ok := result[0].(store.ValidationErrors); !ok {
			t.Errorf("expected a store.ValidationErrors result, got %T", result[0])
		}
	})

	t.Run("doesn't blow up if the media item cannot be saved", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		// This is a disallowed title - see store.MediaItem changeset or issue #549
		response := `{"id":"video1","title":"youtube video #123","original_url":"https://example.com/video1","live_status":"not_live","description":"desc1","aspect_ratio":1.67,"duration":12.34,"upload_date":"20210101"}`

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return response, nil
		})

		result, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		if _, ok := result[0].(store.ValidationErrors); !ok {
			t.Errorf("expected a store.ValidationErrors result, got %T", result[0])
		}
	})

	t.Run("passes the source's download options to the yt-dlp runner", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			output, hasOutput := opts.Get("output")
			if !hasOutput {
				t.Error("expected output option to be set")
			} else if s, _ := output.(string); s == "" {
				t.Error("expected output option to be a non-empty path")
			}

			remux, hasRemux := opts.Get("remux_video")
			if !hasRemux || remux != "mp4" {
				t.Errorf("expected remux_video to be mp4, got %v (present=%v)", remux, hasRemux)
			}

			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)
	})
}

func TestSlowIndexingHelpers_IndexAndEnqueueDownloadForMediaItems_Cookies(t *testing.T) {
	t.Run("sets use_cookies if the source uses cookies", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{CookieBehaviour: store.Ptr(store.SourceCookieBehaviourAllOperations)})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if !addl.UseCookies {
				t.Error("expected use_cookies to be true in addl opts")
			}
			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)
	})

	t.Run("sets use_cookies if the source uses cookies when needed", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{CookieBehaviour: store.Ptr(store.SourceCookieBehaviourWhenNeeded)})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if !addl.UseCookies {
				t.Error("expected use_cookies to be true in addl opts")
			}
			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)
	})

	t.Run("doesn't set use_cookies if the source doesn't use cookies", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{CookieBehaviour: store.Ptr(store.SourceCookieBehaviourDisabled)})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if addl.UseCookies {
				t.Error("expected use_cookies to be false in addl opts")
			}
			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)
	})
}

func TestSlowIndexingHelpers_IndexAndEnqueueDownloadForMediaItems_FileWatcher(t *testing.T) {
	t.Run("creates a new media item for everything already in the file", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		pollInterval := ta.App.Config.FileWatcherPollInterval

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			filepath := addl.OutputFilepath
			must(t, os.WriteFile(filepath, []byte(apptest.SourceAttributesReturnFixture()), 0644))
			time.Sleep(pollInterval * 2)
			// We know we're testing the file watcher since the synchronous call
			// only returns an empty string (creating no records)
			return "", nil
		})

		countBefore, err := store.All[store.MediaItem](ta.Ctx, ta.App.Q(ta.Ctx), store.From[store.MediaItem]("mi"))
		must(t, err)
		if len(countBefore) != 0 {
			t.Fatalf("expected 0 media items before indexing, got %d", len(countBefore))
		}

		_, err = ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		countAfter, err := store.All[store.MediaItem](ta.Ctx, ta.App.Q(ta.Ctx), store.From[store.MediaItem]("mi"))
		must(t, err)
		if len(countAfter) != 3 {
			t.Errorf("expected 3 media items after indexing, got %d", len(countAfter))
		}
	})

	t.Run("enqueues a download for everything already in the file", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		pollInterval := ta.App.Config.FileWatcherPollInterval

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			filepath := addl.OutputFilepath
			must(t, os.WriteFile(filepath, []byte(apptest.SourceAttributesReturnFixture()), 0644))
			time.Sleep(pollInterval * 2)
			return "", nil
		})

		before := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		if len(before) != 0 {
			t.Fatalf("expected no download jobs enqueued before indexing, got %d", len(before))
		}

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		after := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		if len(after) == 0 {
			t.Fatal("expected download job to be enqueued")
		}
	})

	t.Run("does not enqueue downloads if the source is set to not download", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{DownloadMedia: store.Ptr(false)})
		pollInterval := ta.App.Config.FileWatcherPollInterval

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			filepath := addl.OutputFilepath
			must(t, os.WriteFile(filepath, []byte(apptest.SourceAttributesReturnFixture()), 0644))
			time.Sleep(pollInterval * 2)
			return "", nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		if len(jobs) != 0 {
			t.Errorf("expected no download jobs enqueued, got %d", len(jobs))
		}
	})

	t.Run("does not enqueue downloads for media that doesn't match the profile's format options", func(t *testing.T) {
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{ShortsBehaviour: store.Ptr(store.MediaProfileShortsBehaviourExclude)})
		source := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(profile.ID)})
		pollInterval := ta.App.Config.FileWatcherPollInterval

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			filepath := addl.OutputFilepath
			contents := `{"id":"video2","title":"Video 2","original_url":"https://example.com/shorts/video2","live_status":"is_live","description":"desc2","aspect_ratio":1.67,"duration":345.67,"upload_date":"20210101"}`
			must(t, os.WriteFile(filepath, []byte(contents), 0644))
			time.Sleep(pollInterval * 2)
			return "", nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		if len(jobs) != 0 {
			t.Errorf("expected no download jobs enqueued, got %d", len(jobs))
		}
	})

	t.Run("does not enqueue multiple download jobs for the same media items", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		pollInterval := ta.App.Config.FileWatcherPollInterval

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			filepath := addl.OutputFilepath
			must(t, os.WriteFile(filepath, []byte(apptest.SourceAttributesReturnFixture()), 0644))
			time.Sleep(pollInterval * 2)
			// This also returns the final result to the yt-dlp call (like the real
			// usage actually would do) so it'll attempt to create the media items and
			// enqueue the download jobs based on this as well
			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)

		items, err := store.All[store.MediaItem](ta.Ctx, ta.App.Q(ta.Ctx), store.From[store.MediaItem]("mi"))
		must(t, err)
		if len(items) != 3 {
			t.Errorf("expected 3 media items, got %d", len(items))
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
		if len(jobs) != 3 {
			t.Errorf("expected 3 download jobs enqueued, got %d", len(jobs))
		}
	})

	t.Run("does not blow up if the file returns invalid json", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		pollInterval := ta.App.Config.FileWatcherPollInterval

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			filepath := addl.OutputFilepath
			must(t, os.WriteFile(filepath, []byte("INVALID"), 0644))
			time.Sleep(pollInterval * 2)
			return "", nil
		})

		result, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)
		if len(result) != 0 {
			t.Errorf("expected no results, got %d", len(result))
		}
	})
}

func TestSlowIndexingHelpers_IndexAndEnqueueDownloadForMediaItems_DownloadArchive(t *testing.T) {
	t.Run("a download archive is used if the source is a channel that has been indexed before", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			CollectionType: store.Ptr(store.SourceCollectionTypeChannel),
			LastIndexedAt:  store.Ptr(apptest.Now()),
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			hasBreakOnExisting := false
			hasDownloadArchive := false

			for _, opt := range opts {
				if opt.Key == "break_on_existing" {
					hasBreakOnExisting = true
				}
				if opt.Key == "download_archive" {
					hasDownloadArchive = true
				}
			}

			if !hasBreakOnExisting {
				t.Error("expected break_on_existing in opts")
			}
			if !hasDownloadArchive {
				t.Error("expected download_archive in opts")
			}

			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)
	})

	t.Run("a download archive is not used if the source is not a channel", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			CollectionType: store.Ptr(store.SourceCollectionTypePlaylist),
			LastIndexedAt:  store.Ptr(apptest.Now()),
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			for _, opt := range opts {
				if opt.Key == "break_on_existing" {
					t.Error("expected no break_on_existing in opts for playlist")
				}
				if opt.Key == "download_archive" {
					t.Error("expected no download_archive in opts for playlist")
				}
			}

			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)
	})

	t.Run("a download archive is not used if the source has never been indexed before", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			CollectionType: store.Ptr(store.SourceCollectionTypeChannel),
			Clear:          store.ClearLastIndexedAt,
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			for _, opt := range opts {
				if opt.Key == "break_on_existing" {
					t.Error("expected no break_on_existing in opts")
				}
				if opt.Key == "download_archive" {
					t.Error("expected no download_archive in opts")
				}
			}

			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)
	})

	t.Run("a download archive is not used if the index has been forced to run", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			CollectionType: store.Ptr(store.SourceCollectionTypeChannel),
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			for _, opt := range opts {
				if opt.Key == "break_on_existing" {
					t.Error("expected no break_on_existing in opts when forced")
				}
				if opt.Key == "download_archive" {
					t.Error("expected no download_archive in opts when forced")
				}
			}

			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, true)
		must(t, err)
	})

	t.Run("the download archive is formatted correctly and contains the right video", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			CollectionType: store.Ptr(store.SourceCollectionTypeChannel),
			LastIndexedAt:  store.Ptr(apptest.Now()),
		})

		var mediaItems []*store.MediaItem
		for n := 1; n <= 21; n++ {
			mediaItems = append(mediaItems, apptest.MediaItemFixture(t, ta, store.MediaItemParams{
				SourceID:   store.Ptr(source.ID),
				UploadedAt: store.Ptr(apptest.NowMinus(n, "days")),
			}))
		}
		lastMediaItem := mediaItems[len(mediaItems)-1]

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			archiveFile, _ := opts.Get("download_archive")
			contents, err := os.ReadFile(archiveFile.(string))
			must(t, err)
			expected := "youtube " + lastMediaItem.MediaID
			if string(contents) != expected {
				t.Errorf("expected archive contents %q, got %q", expected, string(contents))
			}

			return apptest.SourceAttributesReturnFixture(), nil
		})

		_, err := ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false)
		must(t, err)
	})
}

// Found by running the Go binary against a real database: with an output
// path override, nothing else preloads media_profile, and building quality
// options dereferenced a nil profile.
func TestSlowIndexingHelpers_IndexAndEnqueuePreloadsMediaProfile(t *testing.T) {
	t.Run("works for a freshly loaded source with an output path override", func(t *testing.T) {
		ta := apptest.NewApp(t)
		fixture := apptest.SourceFixture(t, ta, store.SourceParams{OutputPathTemplateOverride: store.Ptr("/{{ title }}.{{ ext }}")})
		source, err := ta.App.GetSource(ta.Ctx, fixture.ID)
		must(t, err)
		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if _, ok := opts.Get("format_sort"); !ok {
				t.Error("expected the media profile's quality options")
			}
			return "", nil
		})

		mustOK(t)(ta.App.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ta.Ctx, source, false))
	})
}
