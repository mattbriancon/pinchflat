package core

import (
	"context"
)

// SlowIndexingHelpers provides methods for performing slow indexing tasks.

// SlowIndexingHelpersKickoffIndexingTask/3
func (a *App) SlowIndexingHelpersKickoffIndexingTask(ctx context.Context, source *Source, jobArgs Attrs, jobOpts KW) (*Task, error) {
	panic("unported: Pinchflat.SlowIndexing.SlowIndexingHelpers.kickoff_indexing_task/3")
}

// SlowIndexingHelpersDeleteIndexingTasks/2
func (a *App) SlowIndexingHelpersDeleteIndexingTasks(ctx context.Context, source *Source, opts KW) error {
	panic("unported: Pinchflat.SlowIndexing.SlowIndexingHelpers.delete_indexing_tasks/2")
}

// SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems/2
func (a *App) SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ctx context.Context, source *Source, opts KW) ([]any, error) {
	panic("unported: Pinchflat.SlowIndexing.SlowIndexingHelpers.index_and_enqueue_download_for_media_items/2")
}
