package app_test

import (
	"net/http"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestFastIndexingWorker_KickoffWithTask(t *testing.T) {
	t.Parallel()
	t.Run("starts worker", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{FastIndex: store.Ptr(true)})

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
		if len(workers) != 0 {
			t.Errorf("workers=%d, want 0 initially", len(workers))
		}

		_, err := ta.App.FastIndexingWorkerKickoffWithTask(ta.Ctx, source, 0)
		if err != nil {
			t.Fatalf("kickoff failed: %v", err)
		}

		workers = ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
		if len(workers) != 1 {
			t.Errorf("workers=%d, want 1", len(workers))
		}
	})

	t.Run("creates task with source", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{FastIndex: store.Ptr(true)})

		task, err := ta.App.FastIndexingWorkerKickoffWithTask(ta.Ctx, source, 0)
		if err != nil {
			t.Fatalf("kickoff failed: %v", err)
		}

		if *task.SourceID != source.ID {
			t.Errorf("task.SourceID=%d, want %d", *task.SourceID, source.ID)
		}
	})
}

func TestFastIndexingWorker_Perform(t *testing.T) {
	t.Parallel()
	t.Run("calls RSS when enabled", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{FastIndex: store.Ptr(true)})

		ta.HTTPMock.Get.Expect(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(app.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("perform failed: %v", err)
		}
	})

	t.Run("reschedules when enabled", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{FastIndex: store.Ptr(true)})

		ta.HTTPMock.Get.Expect(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(app.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("perform failed: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
		if len(jobs) == 0 {
			t.Error("expected rescheduled job")
		}
	})

	t.Run("prevents duplicate reschedules", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{FastIndex: store.Ptr(true)})

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(app.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("first perform failed: %v", err)
		}

		job2, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job2); err != nil {
			t.Errorf("second perform failed: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName})
		if len(jobs) != 1 {
			t.Errorf("jobs=%d, want 1 (duplicate prevention)", len(jobs))
		}
	})

	t.Run("skips RSS when disabled", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{FastIndex: store.Ptr(false)})

		ta.HTTPMock.Get.ExpectN(0, func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(app.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("perform failed: %v", err)
		}
	})

	t.Run("does not reschedule when disabled", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{FastIndex: store.Ptr(false)})

		err := ta.Oban.PerformJob(ta.Ctx, app.FastIndexingWorkerName, map[string]any{"id": source.ID})
		if err != nil {
			t.Fatalf("perform failed: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.FastIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 0 {
			t.Errorf("jobs=%d, want 0", len(jobs))
		}
	})

	t.Run("tolerates missing source", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		spec := obanlite.NewJob(app.FastIndexingWorkerName, map[string]any{"id": 0})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("perform should not error for missing source, got %v", err)
		}
	})
}
