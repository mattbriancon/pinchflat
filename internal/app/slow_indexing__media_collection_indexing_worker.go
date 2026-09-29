package app

import (
	"context"
	"log/slog"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// MediaCollectionIndexingWorker indexes media collections for sources.

const MediaCollectionIndexingWorkerName = "Pinchflat.SlowIndexing.MediaCollectionIndexingWorker"

var mediaCollectionIndexingWorkerOpts = obanlite.WorkerOpts{
	Queue:    "media_collection_indexing",
	Priority: 0,
	Unique: &obanlite.UniqueOpts{
		Period: obanlite.Infinity,
		States: []string{"available", "scheduled", "retryable"},
	},
	Tags: []string{"media_source", "media_collection_indexing", "show_in_dashboard"},
}

// MediaCollectionIndexingWorkerKickoffWithTask/3
func (a *App) MediaCollectionIndexingWorkerKickoffWithTask(ctx context.Context, source *store.Source, jobArgs store.Attrs) (*store.Task, error) {
	// Build arguments
	args := store.Attrs{"id": source.ID}
	if jobArgs != nil {
		for k, v := range jobArgs {
			args[k] = v
		}
	}

	// Create job spec
	spec := obanlite.JobSpec{
		Worker: MediaCollectionIndexingWorkerName,
		Args:   args,
	}

	return a.CreateJobWithTask(ctx, spec, source)
}

// MediaCollectionIndexingWorkerPerform/1
func (a *App) MediaCollectionIndexingWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	// Decode job args
	var args struct {
		ID    int64 `json:"id"`
		Force bool  `json:"force"`
	}
	if err := job.DecodeArgs(&args); err != nil {
		return err
	}

	// Get the source
	source, err := a.GetSource(ctx, args.ID)
	if err != nil {
		// Log and return nil to discard the job
		slog.Info("MediaCollectionIndexingWorker discarded: source not found", "id", args.ID)
		return nil
	}

	indexFreq := source.IndexFrequencyMinutes
	lastIndexedAt := source.LastIndexedAt

	// Case 1: If indexing is on a schedule
	if indexFreq > 0 {
		if err := mediaCollectionIndexingWorkerPerformIndexing(ctx, a, source, args.Force); err != nil {
			return err
		}
		if err := mediaCollectionIndexingWorkerMaybeEnqueueFastIndexingTask(ctx, a, source); err != nil {
			return err
		}
		if err := mediaCollectionIndexingWorkerRescheduleIndexing(ctx, a, source); err != nil {
			return err
		}
		return nil
	}

	// Case 2: If source has never been indexed
	if lastIndexedAt == nil {
		if err := mediaCollectionIndexingWorkerPerformIndexing(ctx, a, source, args.Force); err != nil {
			return err
		}
		return nil
	}

	// Case 3: If source has been indexed and is not meant to reschedule
	// Only perform indexing if forced
	if args.Force {
		if err := mediaCollectionIndexingWorkerPerformIndexing(ctx, a, source, true); err != nil {
			return err
		}
	}

	return nil
}

// Private helpers

func mediaCollectionIndexingWorkerPerformIndexing(ctx context.Context, a *App, source *store.Source, wasForced bool) error {
	_, err := a.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ctx, source, wasForced)
	return err
}

func mediaCollectionIndexingWorkerRescheduleIndexing(ctx context.Context, a *App, source *store.Source) error {
	nextRunIn := source.IndexFrequencyMinutes * 60

	spec := obanlite.JobSpec{
		Worker:     MediaCollectionIndexingWorkerName,
		Args:       store.Attrs{"id": source.ID},
		ScheduleIn: nextRunIn,
	}

	_, err := a.CreateJobWithTask(ctx, spec, source)
	if err != nil {
		if err.Error() == "duplicate_job" {
			return nil
		}
		return err
	}

	return nil
}

func mediaCollectionIndexingWorkerMaybeEnqueueFastIndexingTask(ctx context.Context, a *App, source *store.Source) error {
	if !source.FastIndex {
		return nil
	}

	// Delete existing fast indexing tasks
	if err := a.DeletePendingTasksFor(ctx, source, stringPtr("FastIndexingWorker"), false); err != nil {
		return err
	}

	nextRunIn := store.SourceFastIndexFrequency() * 60

	spec := obanlite.JobSpec{
		Worker:     FastIndexingWorkerName,
		Args:       store.Attrs{"id": source.ID},
		ScheduleIn: nextRunIn,
	}

	_, err := a.CreateJobWithTask(ctx, spec, source)
	if err != nil {
		// Ignore duplicate job errors
		if err.Error() == "duplicate_job" {
			return nil
		}
		return err
	}

	return nil
}
