package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

const FileSyncingWorkerName = "Pinchflat.Media.FileSyncingWorker"

var fileSyncingWorkerOpts = obanlite.WorkerOpts{
	Queue: "local_data",
	Tags:  []string{"sources", "local_data"},
}

// FileSyncingWorker.kickoff_with_task/2
func (a *App) FileSyncingWorkerKickoffWithTask(ctx context.Context, source *Source, opts ...KW) (*Task, error) {
	panic("unported: Pinchflat.Media.FileSyncingWorker.kickoff_with_task/2")
}

// FileSyncingWorker.perform/1
func (a *App) FileSyncingWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.Media.FileSyncingWorker.perform/1")
}
