package app_test

import (
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

func TestMediaCollectionIndexingWorker_KickoffWithTask(t *testing.T) {
	t.Run("starts the worker", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10)})

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName})
		if len(jobs) != 0 {
			t.Fatalf("expected 0 jobs initially, got %d", len(jobs))
		}

		task, err := ta.App.MediaCollectionIndexingWorkerKickoffWithTask(ta.Ctx, source, map[string]any{})
		must(t, err)
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs = ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(jobs))
		}
	})

	t.Run("attaches a task", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10)})

		task, err := ta.App.MediaCollectionIndexingWorkerKickoffWithTask(ta.Ctx, source, map[string]any{})
		must(t, err)

		if *task.SourceID != source.ID {
			t.Errorf("expected task.SourceID to be %d, got %d", source.ID, task.SourceID)
		}
	})

	t.Run("can be called with additional job arguments", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10)})
		jobArgs := map[string]any{"force": true}

		task, err := ta.App.MediaCollectionIndexingWorkerKickoffWithTask(ta.Ctx, source, jobArgs)
		must(t, err)
		if task == nil {
			t.Fatal("expected task, got nil")
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID, "force": true}})
		if len(jobs) == 0 {
			t.Fatal("expected job to be enqueued with force argument")
		}
	})
}

func TestMediaCollectionIndexingWorker_Perform(t *testing.T) {
	for _, c := range []struct {
		name string
		src  store.SourceParams
		args map[string]any
	}{
		{"indexes the source if it should be indexed", store.SourceParams{IndexFrequencyMinutes: store.Ptr(10)}, nil},
		{"indexes the source no matter what if the source has never been indexed before", store.SourceParams{IndexFrequencyMinutes: store.Ptr(0), Clear: store.ClearLastIndexedAt}, nil},
		{"indexes the source no matter what if the 'force' arg is passed", store.SourceParams{IndexFrequencyMinutes: store.Ptr(0), LastIndexedAt: store.Ptr(apptest.Now())}, map[string]any{"force": true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			source := apptest.SourceFixture(t, ta, c.src)
			ta.YtDlpMock.Run.Stub(ytReturns(""))
			args := map[string]any{"id": source.ID}
			for k, v := range c.args {
				args[k] = v
			}

			performIndexing(t, ta, args)
		})
	}

	t.Run("doesn't use a download archive if the index has been forced", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			CollectionType:        store.Ptr(store.SourceCollectionTypeChannel),
			IndexFrequencyMinutes: store.Ptr(0),
			LastIndexedAt:         store.Ptr(apptest.Now()),
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
			return "", nil
		})

		performIndexing(t, ta, map[string]any{"id": source.ID, "force": true})
	})

	t.Run("does not do any indexing if the source has been indexed and shouldn't be rescheduled", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			IndexFrequencyMinutes: store.Ptr(-1),
			LastIndexedAt:         store.Ptr(apptest.Now()),
		})

		// Intentionally not stubbing YtDlpMock.Run: any call is unexpected and
		// will fail the test.

		performIndexing(t, ta, map[string]any{"id": source.ID})
	})

	t.Run("does not reschedule if the source shouldn't be indexed", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(-1)})

		ta.YtDlpMock.Run.Stub(ytReturns(""))
		err := ta.Oban.PerformJob(ta.Ctx, app.MediaCollectionIndexingWorkerName, map[string]any{"id": source.ID})
		must(t, err)

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 0 {
			t.Errorf("expected no rescheduled jobs, got %d", len(jobs))
		}
	})

	for _, c := range []struct {
		name     string
		existing []store.MediaItemParams // pending items created before indexing
		want     int
	}{
		{"kicks off a download job for each pending media item", nil, 3},
		{"starts a job for any pending media item even if it's from another run", []store.MediaItemParams{{Clear: store.ClearMediaFilepath}}, 4},
		// only 3 jobs: the first video is a duplicate
		{"does not kick off a job for media items that could not be saved", []store.MediaItemParams{{MediaID: store.Ptr("video1"), Clear: store.ClearMediaFilepath}}, 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10)})
			for _, p := range c.existing {
				p.SourceID = store.Ptr(source.ID)
				apptest.MediaItemFixture(t, ta, p)
			}

			ta.YtDlpMock.Run.Expect(ytReturns(apptest.SourceAttributesReturnFixture()))
			performIndexing(t, ta, map[string]any{"id": source.ID})

			if n := enqueuedDownloads(t, ta); n != c.want {
				t.Errorf("expected %d download jobs, got %d", c.want, n)
			}
		})
	}

	t.Run("reschedules the job based on the index frequency", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10)})

		ta.YtDlpMock.Run.Stub(ytReturns(""))
		beforeTime := apptest.Now()
		err := ta.Oban.PerformJob(ta.Ctx, app.MediaCollectionIndexingWorkerName, map[string]any{"id": source.ID})
		must(t, err)

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaCollectionIndexingWorkerName, Args: map[string]any{"id": source.ID}})
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
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10)})

		ta.YtDlpMock.Run.Stub(ytReturns(""))
		before, err := ta.App.ListTasksFor(ta.Ctx, source, store.Ptr("MediaCollectionIndexingWorker"), nil)
		must(t, err)
		if len(before) != 0 {
			t.Fatalf("expected 0 tasks before, got %d", len(before))
		}

		err = ta.Oban.PerformJob(ta.Ctx, app.MediaCollectionIndexingWorkerName, map[string]any{"id": source.ID})
		must(t, err)

		after, err := ta.App.ListTasksFor(ta.Ctx, source, store.Ptr("MediaCollectionIndexingWorker"), nil)
		must(t, err)
		if len(after) != 1 {
			t.Errorf("expected 1 task after, got %d", len(after))
		}
	})

	t.Run("creates a future task for fast indexing if appropriate", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10), FastIndex: store.Ptr(true)})

		ta.YtDlpMock.Run.Stub(ytReturns(""))
		beforeTime := apptest.Now()

		performIndexing(t, ta, map[string]any{"id": source.ID})

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName, Args: map[string]any{"id": source.ID}})
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
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10), FastIndex: store.Ptr(true)})

		ta.YtDlpMock.Run.Stub(ytReturns(""))
		existingJob, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{
			Worker: app.FastIndexingWorkerName,
			Args:   map[string]any{"id": source.ID},
		})
		must(t, err)
		task := apptest.TaskFixture(t, ta, apptest.TaskParams{SourceID: store.Ptr(source.ID), JobID: store.Ptr(existingJob.ID)})

		performIndexing(t, ta, map[string]any{"id": source.ID})

		_, err = ta.App.GetTaskBang(ta.Ctx, task.ID)
		if err == nil {
			t.Fatal("expected error reloading task, but got nil")
		}
	})

	t.Run("does not create a task for fast indexing otherwise", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10), FastIndex: store.Ptr(false)})

		ta.YtDlpMock.Run.Stub(ytReturns(""))
		performIndexing(t, ta, map[string]any{"id": source.ID})

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
		if len(jobs) != 0 {
			t.Errorf("expected no fast indexing jobs, got %d", len(jobs))
		}
	})

	t.Run("creates the basic media_item records", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{IndexFrequencyMinutes: store.Ptr(10)})

		ta.YtDlpMock.Run.Expect(ytReturns(apptest.SourceAttributesReturnFixture()))
		mediaItemMediaIDs := func() []string {
			mediaItems, err := store.All[store.MediaItem](ta.Ctx, ta.App.Q(ta.Ctx), store.From[store.MediaItem]("mi").Where(map[string]interface{}{"mi.source_id": source.ID}))
			must(t, err)
			ids := make([]string, len(mediaItems))
			for i, mi := range mediaItems {
				ids[i] = mi.MediaID
			}
			return ids
		}

		if before := mediaItemMediaIDs(); len(before) != 0 {
			t.Fatalf("expected no media items before, got %v", before)
		}

		performIndexing(t, ta, map[string]any{"id": source.ID})

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
		ta := apptest.NewApp(t)

		performIndexing(t, ta, map[string]any{"id": 0})
	})
}

// performIndexing inserts an indexing job with args and performs it.
func performIndexing(t *testing.T, ta *apptest.TestApp, args map[string]any) {
	t.Helper()
	job, err := ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), obanlite.JobSpec{Worker: app.MediaCollectionIndexingWorkerName, Args: args})
	must(t, err)
	must(t, ta.App.MediaCollectionIndexingWorkerPerform(ta.Ctx, job))
}
