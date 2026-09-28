package storetest

import (
	"context"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// TestJobWorker is a simple test worker that performs no operation.
// It mirrors Pinchflat.JobFixtures.TestJobWorker from the Elixir tests.
type TestJobWorker struct{}

// Perform implements the Worker interface for TestJobWorker.
func (w TestJobWorker) Perform(ctx context.Context, job *obanlite.Job) error {
	return nil
}

// TestJobWorkerName is the registered name for the test worker.
const TestJobWorkerName = "Pinchflat.JobFixtures.TestJobWorker"

// JobFixture creates a Job using the TestJobWorker.
func JobFixture(t testing.TB, ts *TestStore) *obanlite.Job {
	t.Helper()

	// Each TestStore has its own Oban, so register on every call (idempotent).
	ts.Oban.Register(TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, TestJobWorker{})

	spec := obanlite.NewJob(TestJobWorkerName, map[string]any{})
	job, err := ts.Oban.Insert(ts.Ctx, ts.Q(ts.Ctx), spec)
	if err != nil {
		t.Fatalf("JobFixture: %v", err)
	}
	return job
}
