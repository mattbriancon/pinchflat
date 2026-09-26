package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// MediaCollectionIndexingWorker indexes media collections for sources.

const MediaCollectionIndexingWorkerName = "Pinchflat.SlowIndexing.MediaCollectionIndexingWorker"

var mediaCollectionIndexingWorkerOpts = obanlite.WorkerOpts{
	Queue:    "media_collection_indexing",
	Priority: 0,
	Unique: &obanlite.UniqueOpts{
		Period: obanlite.Infinity,
		States: []string{"available", "scheduled", "retryable"},
	},
	Tags: []string{"media_source", "media_collection_indexing", "show_in_dashboard"},
}

// MediaCollectionIndexingWorkerKickoffWithTask/3
func (a *App) MediaCollectionIndexingWorkerKickoffWithTask(ctx context.Context, source *Source, jobArgs Attrs, jobOpts KW) (*Task, error) {
	panic("unported: Pinchflat.SlowIndexing.MediaCollectionIndexingWorker.kickoff_with_task/3")
}

// MediaCollectionIndexingWorkerPerform/1
func (a *App) MediaCollectionIndexingWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.SlowIndexing.MediaCollectionIndexingWorker.perform/1")
}
