package apptest

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// TaskFixture creates a Task with sensible defaults, merging in attrs.
func TaskFixture(t testing.TB, ta *TestApp, attrs store.Attrs) *store.Task {
	t.Helper()
	sourceID := int64(SourceFixture(t, ta, store.Attrs{}).ID)
	jobID := int64(JobFixture(t, ta).ID)

	// Check if attrs override defaults
	if v, ok := attrs["job_id"]; ok {
		jobID = v.(int64)
	}

	var finalSourceID, finalMediaItemID *int64
	if v, ok := attrs["source_id"]; ok {
		if id, ok := v.(int64); ok {
			finalSourceID = &id
		} else {
			finalSourceID = &sourceID
		}
	} else {
		finalSourceID = &sourceID
	}

	if v, ok := attrs["media_item_id"]; ok {
		if id, ok := v.(int64); ok {
			finalMediaItemID = &id
		}
	}

	task, err := ta.CreateTask(ta.Ctx, jobID, finalSourceID, finalMediaItemID)
	if err != nil {
		t.Fatalf("TaskFixture: %v", err)
	}
	return task
}
