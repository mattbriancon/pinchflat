package core

import (
	"context"
	"log/slog"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
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
func (a *App) MediaCollectionIndexingWorkerKickoffWithTask(ctx context.Context, source *Source, jobArgs Attrs, jobOpts KW) (*Task, error) {
	// Build arguments
	args := Attrs{"id": source.ID}
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

	// Apply job options
	if scheduleIn, ok := jobOpts.Get("schedule_in"); ok {
		spec.ScheduleIn = scheduleIn.(int)
	}
	if maxAttempts, ok := jobOpts.Get("max_attempts"); ok {
		spec.MaxAttempts = maxAttempts.(int)
	}

	return a.TasksCreateJobWithTask(ctx, spec, source)
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
	source, err := a.SourcesGetSource(ctx, args.ID)
	if err != nil {
		// Log and return nil to discard the job
		slog.Info("MediaCollectionIndexingWorker discarded: source not found", "id", args.ID)
		return nil
	}

	indexFreq := source.IndexFrequencyMinutes
	lastIndexedAt := source.LastIndexedAt

	// Case 1: If indexing is on a schedule
	if indexFreq > 0 {
		if err := mediaCollectionIndexingWorkerPerformIndexingAndNotification(ctx, a, source, args.Force); err != nil {
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
		if err := mediaCollectionIndexingWorkerPerformIndexingAndNotification(ctx, a, source, args.Force); err != nil {
			return err
		}
		return nil
	}

	// Case 3: If source has been indexed and is not meant to reschedule
	// Only perform indexing if forced
	if args.Force {
		if err := mediaCollectionIndexingWorkerPerformIndexingAndNotification(ctx, a, source, true); err != nil {
			return err
		}
	}

	return nil
}

// Private helpers

func mediaCollectionIndexingWorkerPerformIndexingAndNotification(ctx context.Context, a *App, source *Source, wasForced bool) error {
	apprise, err := a.SettingsGetBang(ctx, "apprise_server")
	if err != nil {
		return err
	}

	// apprise_server is stored as a single string (List.wrap in Elixir); wrap
	// it into a one-element slice for the notification runner.
	appriseServers := []string{}
	if s, ok := apprise.(string); ok && s != "" {
		appriseServers = []string{s}
	}

	_, err = a.SourceNotificationsWrapNewMediaNotification(ctx, appriseServers, source, func() (any, error) {
		return a.SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ctx, source, KW{
			Opt("was_forced", wasForced),
		})
	})

	return err
}

func mediaCollectionIndexingWorkerRescheduleIndexing(ctx context.Context, a *App, source *Source) error {
	nextRunIn := source.IndexFrequencyMinutes * 60

	spec := obanlite.JobSpec{
		Worker:     MediaCollectionIndexingWorkerName,
		Args:       Attrs{"id": source.ID},
		ScheduleIn: nextRunIn,
	}

	_, err := a.TasksCreateJobWithTask(ctx, spec, source)
	if err != nil {
		if err.Error() == "duplicate_job" {
			return nil
		}
		return err
	}

	return nil
}

func mediaCollectionIndexingWorkerMaybeEnqueueFastIndexingTask(ctx context.Context, a *App, source *Source) error {
	if !source.FastIndex {
		return nil
	}

	// Delete existing fast indexing tasks
	if err := a.TasksDeletePendingTasksFor(ctx, source, stringPtr("FastIndexingWorker"), KW{}); err != nil {
		return err
	}

	nextRunIn := SourceFastIndexFrequency() * 60

	spec := obanlite.JobSpec{
		Worker:     FastIndexingWorkerName,
		Args:       Attrs{"id": source.ID},
		ScheduleIn: nextRunIn,
	}

	_, err := a.TasksCreateJobWithTask(ctx, spec, source)
	if err != nil {
		// Ignore duplicate job errors
		if err.Error() == "duplicate_job" {
			return nil
		}
		return err
	}

	return nil
}
