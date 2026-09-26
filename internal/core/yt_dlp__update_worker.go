package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

const UpdateWorkerName = "Pinchflat.YtDlp.UpdateWorker"

var updateWorkerOpts = obanlite.WorkerOpts{
	Queue:    "local_data",
	Priority: 0,
	Tags:     []string{"local_data"},
}

// UpdateWorkerKickoff/0
func (a *App) UpdateWorkerKickoff(ctx context.Context) (*obanlite.Job, error) {
	panic("unported: Pinchflat.YtDlp.UpdateWorker.kickoff/0")
}

// UpdateWorkerPerform/1
func (a *App) UpdateWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.YtDlp.UpdateWorker.perform/1")
}
