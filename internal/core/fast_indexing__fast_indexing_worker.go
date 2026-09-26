package core

import (
	"context"
	"log/slog"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
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
func (a *App) FastIndexingWorkerKickoffWithTask(ctx context.Context, source *Source, opts KW) (*Task, error) {
	jobSpec := obanlite.JobSpec{
		Worker: FastIndexingWorkerName,
		Args:   map[string]any{"id": source.ID},
	}

	if scheduleIn, ok := opts.Get("schedule_in"); ok {
		jobSpec.ScheduleIn = scheduleIn.(int)
	}

	return a.TasksCreateJobWithTask(ctx, jobSpec, source)
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

	source, err := a.SourcesGetSource(ctx, args.ID)
	if err != nil {
		if err == ErrNotFound {
			slog.Info("FastIndexingWorker discarded: source not found", "source_id", args.ID)
			return nil
		}
		return err
	}

	if !source.FastIndex {
		return nil
	}

	fastIndexingWorkerPerformIndexingAndSendNotification(ctx, a, source)
	return fastIndexingWorkerRescheduleIndexing(ctx, a, source)
}

func fastIndexingWorkerPerformIndexingAndSendNotification(ctx context.Context, a *App, source *Source) {
	newMediaItems, err := a.FastIndexingHelpersIndexAndKickoffDownloads(ctx, source)
	if err != nil {
		slog.Error("Error indexing media", "source_id", source.ID, "error", err)
		return
	}

	var filteredItems []*MediaItem
	for _, item := range newMediaItems {
		if item != nil {
			pending, err := a.MediaPendingDownload(ctx, item)
			if err == nil && pending {
				filteredItems = append(filteredItems, item)
			}
		}
	}

	if source.DownloadMedia {
		appriseServer, err := a.SettingsGet(ctx, "apprise_server")
		if err == nil {
			// apprise_server is stored as a single string (List.wrap in Elixir);
			// wrap it into a one-element slice for the notification runner.
			servers := []string{}
			if s, ok := appriseServer.(string); ok && s != "" {
				servers = []string{s}
			}
			_ = a.SourceNotificationsSendNewMediaNotification(ctx, servers, source, len(filteredItems))
		}
	}
}

func fastIndexingWorkerRescheduleIndexing(ctx context.Context, a *App, source *Source) error {
	nextRunInSeconds := SourceFastIndexFrequency() * 60

	_, err := a.FastIndexingWorkerKickoffWithTask(ctx, source, KW{Opt("schedule_in", nextRunInSeconds)})
	if err != nil {
		if err.Error() == "duplicate_job" {
			return nil
		}
		return err
	}
	return nil
}
