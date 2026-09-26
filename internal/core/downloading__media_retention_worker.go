package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// MediaRetentionWorker handles deletion of media items past retention date.

const MediaRetentionWorkerName = "Pinchflat.Downloading.MediaRetentionWorker"

var mediaRetentionWorkerOpts = obanlite.WorkerOpts{
	Queue: "local_data",
	Unique: &obanlite.UniqueOpts{
		Period: obanlite.Infinity,
		States: []string{"available", "scheduled", "retryable", "executing"},
	},
	Tags: []string{"media_item", "local_data"},
}

// MediaRetentionWorkerPerform/1
func (a *App) MediaRetentionWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.Downloading.MediaRetentionWorker.perform/1")
}
