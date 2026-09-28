package app

import (
	"context"
	"log/slog"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

const UpdateWorkerName = "Pinchflat.YtDlp.UpdateWorker"

var updateWorkerOpts = obanlite.WorkerOpts{
	Queue:    "local_data",
	Priority: 0,
	Tags:     []string{"local_data"},
}

// UpdateWorkerKickoff/0
func (a *App) UpdateWorkerKickoff(ctx context.Context) (*obanlite.Job, error) {
	job, _, err := a.InsertUniqueJob(ctx, obanlite.JobSpec{
		Worker: UpdateWorkerName,
		Args:   map[string]any{},
	})
	return job, err
}

// UpdateWorkerPerform/1
func (a *App) UpdateWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	slog.Info("Updating yt-dlp")

	a.YtDlp.Update(ctx)

	version, err := a.YtDlp.Version(ctx)
	if err != nil {
		return err
	}

	_, err = a.SetSetting(ctx, store.KW{store.Opt("yt_dlp_version", version)})
	return err
}
