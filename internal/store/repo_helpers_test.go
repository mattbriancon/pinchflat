package store_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/store/storetest"
)

func TestRepoHelpers_InsertUniqueJob(t *testing.T) {
	t.Run("no conflict", func(t *testing.T) {
		t.Parallel()
		ts := storetest.NewStore(t)
		ts.Oban.Register(storetest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, storetest.TestJobWorker{})

		spec := obanlite.NewJob(storetest.TestJobWorkerName, map[string]any{})
		job, duplicate, err := ts.InsertUniqueJob(ts.Ctx, spec)
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
		ts := storetest.NewStore(t)
		ts.Oban.Register(storetest.TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, storetest.TestJobWorker{})

		spec := obanlite.NewJob(storetest.TestJobWorkerName, map[string]any{"foo": "bar"})
		spec.Unique = &obanlite.UniqueOpts{Period: obanlite.Infinity}

		job1, duplicate1, err := ts.InsertUniqueJob(ts.Ctx, spec)
		if err != nil {
			t.Fatalf("first InsertUniqueJob failed: %v", err)
		}
		if job1 == nil {
			t.Fatalf("expected Job")
		}
		if duplicate1 {
			t.Errorf("expected duplicate=false for first insert")
		}

		job2, duplicate2, err := ts.InsertUniqueJob(ts.Ctx, spec)
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
		ts := storetest.NewStore(t)
		spec := obanlite.JobSpec{Worker: ""}
		_, _, err := ts.InsertUniqueJob(ts.Ctx, spec)
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
		{"with limit", store.Ptr(1), 1},
		{"without limit", nil, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := storetest.NewStore(t)
			storetest.MediaProfileFixture(t, ts, store.Attrs{})
			storetest.MediaProfileFixture(t, ts, store.Attrs{})

			q := store.From[store.MediaProfile]()
			q = store.MaybeLimit(q, tt.limit)
			count, err := store.Scalar[int](ts.Ctx, ts.Q(ts.Ctx), store.SQ.Select("COUNT(*)").FromSelect(q, "mp"))
			if err != nil {
				t.Fatalf("Scalar failed: %v", err)
			}
			if count != tt.expectedCount {
				t.Errorf("expected count %d, got %d", tt.expectedCount, count)
			}
		})
	}
}
