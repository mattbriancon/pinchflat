package coretest

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// TaskFixture creates a Task with sensible defaults, merging in attrs.
func TaskFixture(t testing.TB, ta *TestApp, attrs store.Attrs) *store.Task {
	t.Helper()
	defaults := store.Attrs{
		"source_id": SourceFixture(t, ta, store.Attrs{}).ID,
		"job_id":    JobFixture(t, ta).ID,
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	task, err := ta.CreateTask(ta.Ctx, defaults)
	if err != nil {
		t.Fatalf("TaskFixture: %v", err)
	}
	return task
}
