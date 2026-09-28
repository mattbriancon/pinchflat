package storetest

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// TaskFixture creates a Task with sensible defaults, merging in attrs.
func TaskFixture(t testing.TB, ts *TestStore, attrs store.Attrs) *store.Task {
	t.Helper()
	defaults := store.Attrs{
		"source_id": SourceFixture(t, ts, store.Attrs{}).ID,
		"job_id":    JobFixture(t, ts).ID,
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	task, err := ts.CreateTask(ts.Ctx, defaults)
	if err != nil {
		t.Fatalf("TaskFixture: %v", err)
	}
	return task
}
