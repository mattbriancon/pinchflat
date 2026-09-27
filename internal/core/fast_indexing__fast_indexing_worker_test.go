package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestFastIndexingWorker_KickoffWithTask(t *testing.T) {
	t.Run("starts the worker", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		workers := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(workers) != 0 {
			t.Errorf("Expected 0 workers initially, got %d", len(workers))
		}

		_, err := ta.App.FastIndexingWorkerKickoffWithTask(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("Failed to kickoff: %v", err)
		}

		workers = ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(workers) != 1 {
			t.Errorf("Expected 1 worker, got %d", len(workers))
		}
	})

	t.Run("attaches a task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		task, err := ta.App.FastIndexingWorkerKickoffWithTask(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("Failed to kickoff: %v", err)
		}

		if *task.SourceID != source.ID {
			t.Errorf("Expected task.SourceID=%d, got %d", source.ID, *task.SourceID)
		}
	})
}

func TestFastIndexingWorker_Perform(t *testing.T) {
	t.Run("calls out to Youtube RSS if enabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}
	})

	t.Run("reschedules itself if fast indexing is enabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}

		// Should reschedule itself
		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(jobs) == 0 {
			t.Error("Expected a scheduled job after perform")
		}
	})

	t.Run("does not reschedule if that would create a duplicate job", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": true})

		ta.HTTPMock.Get.Stub(func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform first time: %v", err)
		}

		// Perform again
		job2, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job2); err != nil {
			t.Errorf("Failed to perform second time: %v", err)
		}

		// Should only have 1 job (duplicate prevention)
		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName})
		if len(jobs) != 1 {
			t.Errorf("Expected 1 job (duplicate prevention), got %d", len(jobs))
		}
	})

	t.Run("does not call out to Youtube RSS if disabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": false})

		ta.HTTPMock.Get.ExpectN(0, func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Failed to perform: %v", err)
		}
	})

	t.Run("does not reschedule itself if fast indexing is disabled", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"fast_index": false})

		err := ta.Oban.PerformJob(ta.Ctx, core.FastIndexingWorkerName, map[string]any{"id": source.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.FastIndexingWorkerName, Args: map[string]any{"id": source.ID}})
		if len(jobs) != 0 {
			t.Errorf("expected no rescheduled jobs, got %d", len(jobs))
		}
	})

	t.Run("does not blow up if the record doesn't exist", func(t *testing.T) {
		ta := coretest.NewApp(t)

		spec := obanlite.NewJob(core.FastIndexingWorkerName, map[string]any{"id": 0})
		job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
		if err != nil {
			t.Fatalf("Failed to insert job: %v", err)
		}
		if err := ta.App.FastIndexingWorkerPerform(ta.Ctx, job); err != nil {
			t.Errorf("Expected no error for missing source, got %v", err)
		}
	})
}
