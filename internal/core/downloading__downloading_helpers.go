package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

var ErrShouldNotDownload = errors.New("should_not_download")

// DownloadingHelpers provides helper methods for downloading media.

// DownloadingHelpersEnqueuePendingDownloadTasks/2
func (a *App) DownloadingHelpersEnqueuePendingDownloadTasks(ctx context.Context, source *Source, jobOpts KW) error {
	if !source.DownloadMedia {
		return nil
	}

	mediaItems, err := a.MediaListPendingMediaItemsFor(ctx, source)
	if err != nil {
		return err
	}

	for _, mi := range mediaItems {
		// Elixir uses Enum.each, which discards the return value of
		// kickoff_with_task (including any {:error, ...}); mirror that here.
		_, _ = a.MediaDownloadWorkerKickoffWithTask(ctx, mi, Attrs{}, jobOpts)
	}

	return nil
}

// DownloadingHelpersDequeuePendingDownloadTasks/1
func (a *App) DownloadingHelpersDequeuePendingDownloadTasks(ctx context.Context, source *Source) error {
	mediaItems, err := a.MediaListPendingMediaItemsFor(ctx, source)
	if err != nil {
		return err
	}

	for _, mi := range mediaItems {
		err := a.TasksDeletePendingTasksFor(ctx, mi, nil, KW{})
		if err != nil {
			return err
		}
	}

	return nil
}

// DownloadingHelpersKickoffDownloadIfPending/2
func (a *App) DownloadingHelpersKickoffDownloadIfPending(ctx context.Context, mediaItem *MediaItem, jobOpts KW) (*Task, error) {
	// Preload the source
	mediaItem, err := a.mediaPreloadSource(ctx, mediaItem)
	if err != nil {
		return nil, err
	}

	isPending, err := a.MediaPendingDownload(ctx, mediaItem)
	if err != nil {
		return nil, err
	}

	if mediaItem.Source.DownloadMedia && isPending {
		slog.Info(fmt.Sprintf("Kicking off download for media item #%d (%s)", mediaItem.ID, mediaItem.MediaID))

		return a.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, Attrs{}, jobOpts)
	}

	return nil, ErrShouldNotDownload
}

// DownloadingHelpersKickoffRedownloadForExistingMedia/1
func (a *App) DownloadingHelpersKickoffRedownloadForExistingMedia(ctx context.Context, source *Source) ([]interface{}, error) {
	q := MediaQueryNew().
		RequireAssoc("media_profile").
		Where(MediaQueryForSource(source.ID)).
		Where(MediaQueryDownloaded()).
		Where(Not(MediaQueryDownloadPrevented()))

	mediaItems, err := All[MediaItem](ctx, a.Q(ctx), q)
	if err != nil {
		return nil, err
	}

	results := make([]interface{}, 0)
	for _, mi := range mediaItems {
		task, err := a.MediaDownloadWorkerKickoffWithTask(ctx, mi, Attrs{}, KW{})
		if err != nil {
			results = append(results, Attrs{"error": err})
		} else {
			results = append(results, task)
		}
	}

	return results, nil
}

// mediaPreloadSource is a helper to preload the source for a media item
func (a *App) mediaPreloadSource(ctx context.Context, mi *MediaItem) (*MediaItem, error) {
	if mi.Source != nil {
		return mi, nil
	}

	source, err := a.SourcesGetSource(ctx, mi.SourceID)
	if err != nil {
		return nil, err
	}

	mi.Source = source
	return mi, nil
}
