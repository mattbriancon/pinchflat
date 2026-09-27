package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// Port of lib/pinchflat_web/controllers/pages/page_html/job_table_live.ex.

// JobTableLiveRender renders the job table as an htmx fragment.
func (s *Server) JobTableLiveRender(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tasks, err := getJobTableTasks(ctx, s.App)
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	s.RenderFragment(w, r, http.StatusOK, PagesPageHTMLJobTableLiveContent(ctx, tasks))
}

// getJobTableTasks fetches all executing tasks with show_in_dashboard tag.
func getJobTableTasks(ctx context.Context, app *core.App) ([]*core.Task, error) {
	q := core.TasksQueryNew().
		Where(core.TasksQueryInState([]string{"executing"})).
		Where(core.TasksQueryHasTag("show_in_dashboard")).
		OrderBy("j.attempted_at DESC")

	tasks, err := core.All[core.Task](ctx, app.Q(ctx), q)
	if err != nil {
		return nil, err
	}

	// Preload associations
	for _, task := range tasks {
		if task.MediaItemID != nil {
			mi, _ := app.MediaGetMediaItem(ctx, *task.MediaItemID)
			task.MediaItem = mi
			if mi != nil {
				s, _ := app.SourcesGetSource(ctx, mi.SourceID)
				task.Source = s
			}
		}
		if task.SourceID != nil {
			s, _ := app.SourcesGetSource(ctx, *task.SourceID)
			task.Source = s
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
