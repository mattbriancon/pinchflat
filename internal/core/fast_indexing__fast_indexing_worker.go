package core

import (
	"context"
	"log/slog"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

const FastIndexingWorkerName = "Pinchflat.FastIndexing.FastIndexingWorker"

var fastIndexingWorkerOpts = obanlite.WorkerOpts{
	Queue:    "fast_indexing",
	Priority: 0,
	Unique: &obanlite.UniqueOpts{
		Period: obanlite.Infinity,
		States: []string{"available", "scheduled", "retryable"},
	},
	Tags: []string{"media_source", "fast_indexing", "show_in_dashboard"},
}

// FastIndexingWorkerKickoffWithTask/2
func (a *App) FastIndexingWorkerKickoffWithTask(ctx context.Context, source *store.Source, opts store.KW) (*store.Task, error) {
	jobSpec := obanlite.JobSpec{
		Worker: FastIndexingWorkerName,
		Args:   map[string]any{"id": source.ID},
	}

	if scheduleIn, ok := opts.Get("schedule_in"); ok {
		jobSpec.ScheduleIn = scheduleIn.(int)
	}

	return a.CreateJobWithTask(ctx, jobSpec, source)
}

// FastIndexingWorkerPerform/1
func (a *App) FastIndexingWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	var args struct {
		ID int64 `json:"id"`
	}

	if err := job.DecodeArgs(&args); err != nil {
		slog.Error("Failed to decode FastIndexingWorker args", "error", err)
		return err
	}

	source, err := a.GetSource(ctx, args.ID)
	if err != nil {
		if err == store.ErrNotFound {
			slog.Info("FastIndexingWorker discarded: source not found", "source_id", args.ID)
			return nil
		}
		return err
	}

	if !source.FastIndex {
		return nil
	}

	fastIndexingWorkerPerformIndexing(ctx, a, source)
	return fastIndexingWorkerRescheduleIndexing(ctx, a, source)
}

func fastIndexingWorkerPerformIndexing(ctx context.Context, a *App, source *store.Source) {
	if _, err := a.FastIndexingHelpersIndexAndKickoffDownloads(ctx, source); err != nil {
		slog.Error("Error indexing media", "source_id", source.ID, "error", err)
	}
}

func fastIndexingWorkerRescheduleIndexing(ctx context.Context, a *App, source *store.Source) error {
	nextRunInSeconds := store.SourceFastIndexFrequency() * 60

	_, err := a.FastIndexingWorkerKickoffWithTask(ctx, source, store.KW{store.Opt("schedule_in", nextRunInSeconds)})
	if err != nil {
		if err.Error() == "duplicate_job" {
			return nil
		}
		return err
	}
	return nil
}
