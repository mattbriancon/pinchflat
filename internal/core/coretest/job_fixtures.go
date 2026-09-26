package coretest

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
func JobFixture(t testing.TB, ta *TestApp) *obanlite.Job {
	t.Helper()

	// Register TestJobWorker with this test app's Oban instance
	// (each TestApp has its own Oban, so we need to register with each one)
	ta.Oban.Register(TestJobWorkerName, obanlite.WorkerOpts{Queue: "default"}, TestJobWorker{})

	spec := obanlite.NewJob(TestJobWorkerName, map[string]any{})
	job, err := ta.Oban.Insert(ta.Ctx, ta.Q(ta.Ctx), spec)
	if err != nil {
		t.Fatalf("JobFixture: %v", err)
	}
	return job
}
