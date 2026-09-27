package web

import (
	"context"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// Port of lib/pinchflat_web/controllers/pages/page_html/job_table_live.ex.

// getJobTableTasks fetches all executing tasks with show_in_dashboard tag.
func getJobTableTasks(ctx context.Context, app *core.App) ([]*core.Task, error) {
	q := core.TasksQueryJoinJob(core.TasksQueryNew()).
		Where(core.TasksQueryInState([]string{"executing"})).
		Where(core.TasksQueryHasTag("show_in_dashboard")).
		OrderBy("j.attempted_at DESC")

	tasks, err := core.All[core.Task](ctx, app.Q(ctx), q)
	if err != nil {
		return nil, err
	}

	// Preload associations (preload(tasks: [:job, :media_item, :source])).
	for _, task := range tasks {
		if _, err := app.PreloadTaskJob(ctx, task); err != nil {
			return nil, err
		}
		if _, err := app.PreloadTaskMediaItem(ctx, task); err != nil {
			return nil, err
		}
		if _, err := app.PreloadTaskSource(ctx, task); err != nil {
			return nil, err
		}
	}

	return tasks, nil
}

// mapWorkerToTaskName converts worker class name to task name.
func mapWorkerToTaskName(worker string) string {
	// Extract the last part of the module name
	parts := strings.Split(worker, ".")
	finalModulePart := parts[len(parts)-1]

	switch finalModulePart {
	case "FastIndexingWorker":
		return "Fast Indexing Source"
	case "MediaDownloadWorker":
		return "Downloading Media"
	case "MediaCollectionIndexingWorker":
		return "Indexing Source"
	case "MediaQualityUpgradeWorker":
		return "Upgrading Media Quality"
	case "SourceMetadataStorageWorker":
		return "Fetching Source Metadata"
	default:
		return finalModulePart + " (Report to Devs)"
	}
}

// taskToRecordName gets the name of the record associated with a task.
func taskToRecordName(task *core.Task) string {
	if task.Source != nil {
		return task.Source.CustomName
	}
	if task.MediaItem != nil && task.MediaItem.Title != nil {
		return *task.MediaItem.Title
	}
	return "Unknown Record"
}

// taskToLink gets the link for a task's record.
func taskToLink(ctx context.Context, task *core.Task) string {
	if task.Source != nil {
		return P(ctx, "/sources/%v", task.Source.ID)
	}
	if task.MediaItem != nil {
		return P(ctx, "/sources/%v/media/%v", task.MediaItem.SourceID, task.MediaItem.ID)
	}
	return "#"
}
