package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestRepoHelpers_InsertUniqueJob(t *testing.T) {
	t.Run("no conflict", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{})
		job, duplicate, err := ta.InsertUniqueJob(ta.Ctx, spec)
		if err != nil {
			t.Fatalf("InsertUniqueJob failed: %v", err)
		}
		if job == nil {
			t.Fatalf("expected Job")
		}
		if duplicate {
			t.Errorf("expected duplicate=false")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ta.Oban.Register(coretest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, coretest.TestJobWorker{})

		spec := obanlite.NewJob(coretest.TestJobWorkerName, map[string]any{"foo": "bar"})
		spec.Unique = &obanlite.UniqueOpts{Period: obanlite.Infinity}

		job1, duplicate1, err := ta.InsertUniqueJob(ta.Ctx, spec)
		if err != nil {
			t.Fatalf("first InsertUniqueJob failed: %v", err)
		}
		if job1 == nil {
			t.Fatalf("expected Job")
		}
		if duplicate1 {
			t.Errorf("expected duplicate=false for first insert")
		}

		job2, duplicate2, err := ta.InsertUniqueJob(ta.Ctx, spec)
		if err != nil {
			t.Fatalf("second InsertUniqueJob failed: %v", err)
		}
		if job2 == nil {
			t.Fatalf("expected Job")
		}
		if !duplicate2 {
			t.Errorf("expected duplicate=true for second insert")
		}
		if job1.ID != job2.ID {
			t.Errorf("expected same job ID %d, got %d", job1.ID, job2.ID)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		spec := obanlite.JobSpec{Worker: ""}
		_, _, err := ta.InsertUniqueJob(ta.Ctx, spec)
		if err == nil {
			t.Errorf("expected error for invalid job spec")
		}
	})
}

func TestRepoHelpers_MaybeLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		limit         *int
		expectedCount int
	}{
		{"with limit", core.Ptr(1), 1},
		{"without limit", nil, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ta := coretest.NewApp(t)
			coretest.MediaProfileFixture(t, ta, core.Attrs{})
			coretest.MediaProfileFixture(t, ta, core.Attrs{})

			q := core.From[core.MediaProfile]()
			q = core.MaybeLimit(q, tt.limit)
			count, err := core.Scalar[int](ta.Ctx, ta.Q(ta.Ctx), core.SQ.Select("COUNT(*)").FromSelect(q, "mp"))
			if err != nil {
				t.Fatalf("Scalar failed: %v", err)
			}
			if count != tt.expectedCount {
				t.Errorf("expected count %d, got %d", tt.expectedCount, count)
			}
		})
	}
}
