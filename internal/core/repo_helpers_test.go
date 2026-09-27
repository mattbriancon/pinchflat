package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestRepoHelpers_InsertUniqueJob(t *testing.T) {
	ta := coretest.NewApp(t)

	t.Run("returns {:ok, job} if there is no conflict", func(t *testing.T) {
		// Register the test worker
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{})
		job, duplicate, err := ta.InsertUniqueJob(ta.Ctx, spec)
		if err != nil {
			t.Fatalf("InsertUniqueJob failed: %v", err)
		}
		if job == nil {
			t.Fatalf("expected Job, got nil")
		}
		if duplicate {
			t.Errorf("expected duplicate=false, got true")
		}
	})

	t.Run("returns {:duplicate, original_job} if there is a conflict", func(t *testing.T) {
		// Register the test worker
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{"foo": "bar"})
		spec.Unique = &obanlite.UniqueOpts{Period: obanlite.Infinity}

		job1, duplicate1, err := ta.InsertUniqueJob(ta.Ctx, spec)
		if err != nil {
			t.Fatalf("first InsertUniqueJob failed: %v", err)
		}
		if job1 == nil {
			t.Fatalf("expected Job, got nil")
		}
		if duplicate1 {
			t.Errorf("expected duplicate=false for first insert, got true")
		}

		job2, duplicate2, err := ta.InsertUniqueJob(ta.Ctx, spec)
		if err != nil {
			t.Fatalf("second InsertUniqueJob failed: %v", err)
		}
		if job2 == nil {
			t.Fatalf("expected Job, got nil")
		}
		if !duplicate2 {
			t.Errorf("expected duplicate=true for second insert, got false")
		}
		if job1.ID != job2.ID {
			t.Errorf("expected same job ID %d, got %d", job1.ID, job2.ID)
		}
	})

	t.Run("returns the error if there is an error", func(t *testing.T) {
		// In Go, we pass an empty/invalid spec to Oban and expect an error
		spec := obanlite.JobSpec{Worker: ""} // Missing worker name should cause an error
		_, _, err := ta.InsertUniqueJob(ta.Ctx, spec)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}

func TestRepoHelpers_MaybeLimit(t *testing.T) {
	t.Run("applies a limit if provided", func(t *testing.T) {
		ta := coretest.NewApp(t)
		coretest.MediaProfileFixture(t, ta, core.Attrs{})
		coretest.MediaProfileFixture(t, ta, core.Attrs{})

		q := core.From[core.MediaProfile]()
		q = core.MaybeLimit(q, core.Ptr(1))
		count, err := core.Scalar[int](ta.Ctx, ta.Q(ta.Ctx), core.SQ.Select("COUNT(*)").FromSelect(q, "mp"))
		if err != nil {
			t.Fatalf("Scalar failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
	})

	t.Run("does not apply a limit if not provided", func(t *testing.T) {
		ta := coretest.NewApp(t)
		coretest.MediaProfileFixture(t, ta, core.Attrs{})
		coretest.MediaProfileFixture(t, ta, core.Attrs{})

		q := core.From[core.MediaProfile]()
		q = core.MaybeLimit(q, nil)
		count, err := core.Scalar[int](ta.Ctx, ta.Q(ta.Ctx), core.SQ.Select("COUNT(*)").FromSelect(q, "mp"))
		if err != nil {
			t.Fatalf("Scalar failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected count 2, got %d", count)
		}
	})
}
