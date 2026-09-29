package storetest

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// TaskParams overrides the defaults of TaskFixture.
type TaskParams struct {
	JobID       *int64
	SourceID    *int64
	MediaItemID *int64
}

// TaskFixture creates a Task with sensible defaults for whatever p leaves
// unset: a new job, and a new source.
func TaskFixture(t testing.TB, ts *TestStore, p TaskParams) *store.Task {
	t.Helper()
	if p.JobID == nil {
		p.JobID = store.Ptr(int64(JobFixture(t, ts).ID))
	}
	if p.SourceID == nil {
		p.SourceID = store.Ptr(SourceFixture(t, ts, store.SourceParams{}).ID)
	}

	task, err := ts.CreateTask(ts.Ctx, *p.JobID, p.SourceID, p.MediaItemID)
	if err != nil {
		t.Fatalf("TaskFixture: %v", err)
	}
	return task
}
