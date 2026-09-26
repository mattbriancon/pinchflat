package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

const FastIndexingWorkerName = "Pinchflat.FastIndexing.FastIndexingWorker"

var fastIndexingWorkerOpts = obanlite.WorkerOpts{
	Queue:    "fast_indexing",
	Priority: 0,
	Unique: &obanlite.UniqueOpts{
		Period: obanlite.Infinity,
		States: []string{"available", "scheduled", "retryable"},
	},
	Tags: []string{"media_source", "fast_indexing", "show_in_dashboard"},
}

// FastIndexingWorkerKickoffWithTask/2
func (a *App) FastIndexingWorkerKickoffWithTask(ctx context.Context, source *Source, opts KW) (*Task, error) {
	panic("unported: Pinchflat.FastIndexing.FastIndexingWorker.kickoff_with_task/2")
}

// FastIndexingWorkerPerform/1
func (a *App) FastIndexingWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.FastIndexing.FastIndexingWorker.perform/1")
}
