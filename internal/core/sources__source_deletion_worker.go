package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

const SourceDeletionWorkerName = "Pinchflat.Sources.SourceDeletionWorker"

var sourceDeletionWorkerOpts = obanlite.WorkerOpts{
	Queue: "local_data",
	Tags:  []string{"sources", "local_data"},
}

// SourceDeletionWorker.kickoff/1, kickoff/2, kickoff/3
func (a *App) SourceDeletionWorkerKickoff(ctx context.Context, source *Source, jobArgs Attrs, jobOpts KW) (*obanlite.Job, error) {
	panic("unported: Pinchflat.Sources.SourceDeletionWorker.kickoff/3")
}

// SourceDeletionWorker.perform/1
func (a *App) SourceDeletionWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.Sources.SourceDeletionWorker.perform/1")
}
