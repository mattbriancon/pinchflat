package app

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

const MediaProfileDeletionWorkerName = "Pinchflat.Profiles.MediaProfileDeletionWorker"

var mediaProfileDeletionWorkerOpts = obanlite.WorkerOpts{
	Queue: "local_data",
	Tags:  []string{"media_profiles", "local_data"},
}

// MediaProfileDeletionWorker.kickoff/1, kickoff/2, kickoff/3
func (a *App) MediaProfileDeletionWorkerKickoff(ctx context.Context, profile *store.MediaProfile, jobArgs map[string]any) (*obanlite.Job, error) {
	// Build args: {id: profile.id} merged with jobArgs
	args := make(map[string]any)
	args["id"] = profile.ID
	for k, v := range jobArgs {
		args[k] = v
	}

	// Create job spec with base worker options
	spec := obanlite.JobSpec{
		Worker: MediaProfileDeletionWorkerName,
		Args:   args,
	}

	return a.Oban.Insert(ctx, a.Q(ctx), spec)
}

// MediaProfileDeletionWorker.perform/1
func (a *App) MediaProfileDeletionWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	var args struct {
		ID          int64 `json:"id"`
		DeleteFiles *bool `json:"delete_files"`
	}
	if err := job.DecodeArgs(&args); err != nil {
		return err
	}

	deleteFiles := false
	if args.DeleteFiles != nil {
		deleteFiles = *args.DeleteFiles
	}

	profile, err := a.GetMediaProfile(ctx, args.ID)
	if err != nil {
		return err
	}

	_, err = a.ProfilesDeleteMediaProfile(ctx, profile, deleteFiles)
	return err
}
