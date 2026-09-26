package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

const MediaProfileDeletionWorkerName = "Pinchflat.Profiles.MediaProfileDeletionWorker"

var mediaProfileDeletionWorkerOpts = obanlite.WorkerOpts{
	Queue: "local_data",
	Tags:  []string{"media_profiles", "local_data"},
}

// MediaProfileDeletionWorker.kickoff/1, kickoff/2, kickoff/3
func (a *App) MediaProfileDeletionWorkerKickoff(ctx context.Context, profile *MediaProfile, jobArgs ...Attrs) (*obanlite.Job, error) {
	panic("unported: Pinchflat.Profiles.MediaProfileDeletionWorker.kickoff/3")
}

// MediaProfileDeletionWorker.perform/1
func (a *App) MediaProfileDeletionWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	panic("unported: Pinchflat.Profiles.MediaProfileDeletionWorker.perform/1")
}
