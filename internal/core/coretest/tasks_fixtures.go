package coretest

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// TaskFixture creates a Task with sensible defaults, merging in attrs.
func TaskFixture(t testing.TB, ta *TestApp, attrs core.Attrs) *core.Task {
	t.Helper()
	defaults := core.Attrs{
		"source_id": SourceFixture(t, ta, core.Attrs{}).ID,
		"job_id":    JobFixture(t, ta).ID,
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	task, err := ta.TasksCreateTask(ta.Ctx, defaults)
	if err != nil {
		t.Fatalf("TaskFixture: %v", err)
	}
	return task
}
