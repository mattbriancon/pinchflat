package core

import "context"

// DownloadingHelpers provides helper methods for downloading media.

// DownloadingHelpersEnqueuePendingDownloadTasks/2
func (a *App) DownloadingHelpersEnqueuePendingDownloadTasks(ctx context.Context, source *Source, jobOpts KW) error {
	panic("unported: Pinchflat.Downloading.DownloadingHelpers.enqueue_pending_download_tasks/2")
}

// DownloadingHelpersDequeuePendingDownloadTasks/1
func (a *App) DownloadingHelpersDequeuePendingDownloadTasks(ctx context.Context, source *Source) error {
	panic("unported: Pinchflat.Downloading.DownloadingHelpers.dequeue_pending_download_tasks/1")
}

// DownloadingHelpersKickoffDownloadIfPending/2
func (a *App) DownloadingHelpersKickoffDownloadIfPending(ctx context.Context, mediaItem *MediaItem, jobOpts KW) (*Task, error) {
	panic("unported: Pinchflat.Downloading.DownloadingHelpers.kickoff_download_if_pending/2")
}

// DownloadingHelpersKickoffRedownloadForExistingMedia/1
func (a *App) DownloadingHelpersKickoffRedownloadForExistingMedia(ctx context.Context, source *Source) ([]interface{}, error) {
	panic("unported: Pinchflat.Downloading.DownloadingHelpers.kickoff_redownload_for_existing_media/1")
}
