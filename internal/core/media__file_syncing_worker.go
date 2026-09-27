package core

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

const FileSyncingWorkerName = "Pinchflat.Media.FileSyncingWorker"

var fileSyncingWorkerOpts = obanlite.WorkerOpts{
	Queue: "local_data",
	Tags:  []string{"sources", "local_data"},
}

// FileSyncingWorker.kickoff_with_task/2
func (a *App) FileSyncingWorkerKickoffWithTask(ctx context.Context, source *Source, opts KW) (*Task, error) {
	jobSpec := obanlite.JobSpec{
		Worker: FileSyncingWorkerName,
		Args: map[string]any{
			"id": source.ID,
		},
	}

	// Apply any options passed in (schedule_in, priority, etc)
	for _, kv := range opts {
		if kv.Flag {
			continue
		}
		switch kv.Key {
		case "schedule_in":
			if scheduleIn, ok := kv.Value.(int); ok {
				jobSpec.ScheduleIn = scheduleIn
			}
		case "priority":
			if priority, ok := kv.Value.(int); ok {
				jobSpec.Priority = &priority
			}
		case "max_attempts":
			if maxAttempts, ok := kv.Value.(int); ok {
				jobSpec.MaxAttempts = maxAttempts
			}
		}
	}

	return a.TasksCreateJobWithTask(ctx, jobSpec, source)
}

// FileSyncingWorker.perform/1
func (a *App) FileSyncingWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	var args struct {
		ID int64 `json:"id"`
	}

	if err := job.DecodeArgs(&args); err != nil {
		return err
	}

	source, err := a.SourcesGetSource(ctx, args.ID)
	if err != nil {
		return err
	}

	// Preload media_items
	mediaItems, err := All[MediaItem](ctx, a.Q(ctx), From[MediaItem]().Where(sq.Eq{"source_id": source.ID}))
	if err != nil {
		return err
	}

	_, err = a.FileSyncingSyncFilePresenceOnDisk(ctx, mediaItems)
	return err
}
