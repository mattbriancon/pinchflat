package core_test

import (
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestMediaCollectionIndexingWorker_KickoffWithTask(t *testing.T) {
	t.Run("starts the worker", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName})
		if len(jobs) != 0 {
			t.Fatalf("expected 0 jobs initially, got %d", len(jobs))
		}

		task, err := ta.App.MediaCollectionIndexingWorkerKickoffWithTask(ta.Ctx, source, store.Attrs{}, store.KW{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs = ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(jobs))
		}
	})

	t.Run("attaches a task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})

		task, err := ta.App.MediaCollectionIndexingWorkerKickoffWithTask(ta.Ctx, source, store.Attrs{}, store.KW{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if *task.SourceID != source.ID {
			t.Errorf("expected task.SourceID to be %d, got %d", source.ID, task.SourceID)
		}
	})

	t.Run("can be called with additional job arguments", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})
		jobArgs := store.Attrs{"force": true}

		task, err := ta.App.MediaCollectionIndexingWorkerKickoffWithTask(ta.Ctx, source, jobArgs, store.KW{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID, "force": true}})
		if len(jobs) == 0 {
			t.Fatal("expected job to be enqueued with force argument")
		}
	})

	t.Run("can be called with additional job options", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})
		jobOpts := store.KW{store.Opt("max_attempts", 5)}

		task, err := ta.App.MediaCollectionIndexingWorkerKickoffWithTask(ta.Ctx, source, store.Attrs{}, jobOpts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(jobs))
		}
		if jobs[0].MaxAttempts != 5 {
			t.Errorf("expected MaxAttempts to be 5, got %d", jobs[0].MaxAttempts)
		}
	})
}

func TestMediaCollectionIndexingWorker_Perform(t *testing.T) {
	t.Run("indexes the source if it should be indexed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})

		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("indexes the source no matter what if the source has never been indexed before", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{
			"index_frequency_minutes": 0,
			"last_indexed_at":         nil,
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})

		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("indexes the source no matter what if the 'force' arg is passed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{
			"index_frequency_minutes": 0,
			"last_indexed_at":         coretest.Now(),
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})

		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID, "force": true},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("doesn't use a download archive if the index has been forced", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{
			"collection_type":         "channel",
			"index_frequency_minutes": 0,
			"last_indexed_at":         coretest.Now(),
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			for _, opt := range opts {
				if opt.Key == "break_on_existing" {
					t.Error("expected no break_on_existing in opts")
				}
				if opt.Key == "download_archive" {
					t.Error("expected no download_archive in opts")
				}
			}
			return "", nil
		})

		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID, "force": true},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("does not do any indexing if the source has been indexed and shouldn't be rescheduled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{
			"index_frequency_minutes": -1,
			"last_indexed_at":         coretest.Now(),
		})

		// Intentionally not stubbing YtDlpMock.Run: any call is unexpected and
		// will fail the test.

		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("does not reschedule if the source shouldn't be indexed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": -1})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})
		err := ta.Oban.PerformJob(ta.Ctx, core.MediaCollectionIndexingWorkerName, map[string]any{"id": source.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 0 {
			t.Errorf("expected no rescheduled jobs, got %d", len(jobs))
		}
	})

	t.Run("kicks off a download job for each pending media item", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return coretest.SourceAttributesReturnFixture(), nil
		})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(jobs) != 3 {
			t.Errorf("expected 3 download jobs, got %d", len(jobs))
		}
	})

	t.Run("starts a job for any pending media item even if it's from another run", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})
		coretest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return coretest.SourceAttributesReturnFixture(), nil
		})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(jobs) != 4 {
			t.Errorf("expected 4 download jobs, got %d", len(jobs))
		}
	})

	t.Run("does not kick off a job for media items that could not be saved", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})
		coretest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID, "media_filepath": nil, "media_id": "video1"})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return coretest.SourceAttributesReturnFixture(), nil
		})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Only 3 jobs should be enqueued, since the first video is a duplicate
		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(jobs) != 3 {
			t.Errorf("expected 3 download jobs, got %d", len(jobs))
		}
	})

	t.Run("reschedules the job based on the index frequency", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})
		beforeTime := coretest.Now()
		err := ta.Oban.PerformJob(ta.Ctx, core.MediaCollectionIndexingWorkerName, map[string]any{"id": source.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 rescheduled job, got %d", len(jobs))
		}

		expected := beforeTime.Add(time.Duration(source.IndexFrequencyMinutes) * time.Minute)
		diff := jobs[0].ScheduledAt.Sub(expected)
		if diff < -5*time.Second || diff > 5*time.Second {
			t.Errorf("expected scheduled at ~%v, got %v", expected, jobs[0].ScheduledAt)
		}
	})

	t.Run("creates a task for the rescheduled job", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})
		before, err := ta.App.ListTasksFor(ta.Ctx, source, store.Ptr("MediaCollectionIndexingWorker"), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(before) != 0 {
			t.Fatalf("expected 0 tasks before, got %d", len(before))
		}

		err = ta.Oban.PerformJob(ta.Ctx, core.MediaCollectionIndexingWorkerName, map[string]any{"id": source.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		after, err := ta.App.ListTasksFor(ta.Ctx, source, store.Ptr("MediaCollectionIndexingWorker"), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(after) != 1 {
			t.Errorf("expected 1 task after, got %d", len(after))
		}
	})

	t.Run("creates a future task for fast indexing if appropriate", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10, "fast_index": true})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})
		beforeTime := coretest.Now()

		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 fast indexing job, got %d", len(jobs))
		}

		expected := beforeTime.Add(time.Duration(store.SourceFastIndexFrequency()) * time.Minute)
		diff := jobs[0].ScheduledAt.Sub(expected)
		if diff < -5*time.Second || diff > 5*time.Second {
			t.Errorf("expected scheduled at ~%v, got %v", expected, jobs[0].ScheduledAt)
		}
	})

	t.Run("deletes existing fast indexing tasks if a new one is created", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10, "fast_index": true})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})
		existingJob, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.FastIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}
		task := coretest.TaskFixture(t, ta, store.Attrs{"source_id": source.ID, "job_id": existingJob.ID})

		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = ta.App.GetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Fatal("expected error reloading task, but got nil")
		}
	})

	t.Run("does not create a task for fast indexing otherwise", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10, "fast_index": false})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})
		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(jobs) != 0 {
			t.Errorf("expected no fast indexing jobs, got %d", len(jobs))
		}
	})

	t.Run("creates the basic media_item records", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, store.Attrs{"index_frequency_minutes": 10})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return coretest.SourceAttributesReturnFixture(), nil
		})
		mediaItemMediaIDs := func() []string {
			mediaItems, err := store.All[store.MediaItem](ta.Ctx, ta.App.Q(ta.Ctx), store.From[store.MediaItem]("mi").Where(map[string]interface{}{"mi.source_id": source.ID}))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			ids := make([]string, len(mediaItems))
			for i, mi := range mediaItems {
				ids[i] = mi.MediaID
			}
			return ids
		}

		if before := mediaItemMediaIDs(); len(before) != 0 {
			t.Fatalf("expected no media items before, got %v", before)
		}

		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		after := mediaItemMediaIDs()
		expected := []string{"video1", "video2", "video3"}
		if len(after) != len(expected) {
			t.Fatalf("expected %v, got %v", expected, after)
		}
		for i, id := range expected {
			if after[i] != id {
				t.Errorf("expected media_id %q at index %d, got %q", id, i, after[i])
			}
		}
	})

	t.Run("does not blow up if the record doesn't exist", func(t *testing.T) {
		ta := coretest.NewApp(t)

		job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: core.MediaCollectionIndexingWorkerName,
			Args:   map[string]any{"id": 0},
		})
		if err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}

		err = ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
