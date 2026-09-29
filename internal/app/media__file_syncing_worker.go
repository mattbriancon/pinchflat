package app

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

const FileSyncingWorkerName = "Pinchflat.Media.FileSyncingWorker"

var fileSyncingWorkerOpts = obanlite.WorkerOpts{
	Queue: "local_data",
	Tags:  []string{"sources", "local_data"},
}

// FileSyncingWorker.kickoff_with_task/2
func (a *App) FileSyncingWorkerKickoffWithTask(ctx context.Context, source *store.Source) (*store.Task, error) {
	jobSpec := obanlite.JobSpec{
		Worker: FileSyncingWorkerName,
		Args: map[string]any{
			"id": source.ID,
		},
	}

	return a.CreateJobWithTask(ctx, jobSpec, source)
}

// FileSyncingWorker.perform/1
func (a *App) FileSyncingWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	var args struct {
		ID int64 `json:"id"`
	}

	if err := job.DecodeArgs(&args); err != nil {
		return err
	}

	source, err := a.GetSource(ctx, args.ID)
	if err != nil {
		return err
	}

	// Preload media_items
	mediaItems, err := store.All[store.MediaItem](ctx, a.Q(ctx), store.From[store.MediaItem]().Where(sq.Eq{"source_id": source.ID}))
	if err != nil {
		return err
	}

	_, err = a.FileSyncingSyncFilePresenceOnDisk(ctx, mediaItems)
	return err
}
