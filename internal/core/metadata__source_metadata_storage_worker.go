package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

const SourceMetadataStorageWorkerName = "Pinchflat.Metadata.SourceMetadataStorageWorker"

var sourceMetadataStorageWorkerOpts = obanlite.WorkerOpts{
	Queue:       "remote_metadata",
	Tags:        []string{"media_source", "source_metadata", "remote_metadata", "show_in_dashboard"},
	MaxAttempts: 3,
}

// SourceMetadataStorageWorkerKickoffWithTask/2
func (a *App) SourceMetadataStorageWorkerKickoffWithTask(ctx context.Context, source *Source, opts KW) (*Task, error) {
	panic("unported: Pinchflat.Metadata.SourceMetadataStorageWorker.kickoff_with_task/2")
}

// perform/1
func (a *App) SourceMetadataStorageWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.Metadata.SourceMetadataStorageWorker.perform/1")
}
