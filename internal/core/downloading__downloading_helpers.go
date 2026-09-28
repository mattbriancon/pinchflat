package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mattbriancon/pinchflat/internal/store"
)

var ErrShouldNotDownload = errors.New("should_not_download")

// DownloadingHelpers provides helper methods for downloading media.

// DownloadingHelpersEnqueuePendingDownloadTasks/2
func (a *App) DownloadingHelpersEnqueuePendingDownloadTasks(ctx context.Context, source *store.Source, jobOpts store.KW) error {
	if !source.DownloadMedia {
		return nil
	}

	mediaItems, err := a.ListPendingMediaItemsFor(ctx, source)
	if err != nil {
		return err
	}

	for _, mi := range mediaItems {
		// Elixir uses Enum.each, which discards the return value of
		// kickoff_with_task (including any {:error, ...}); mirror that here.
		_, _ = a.MediaDownloadWorkerKickoffWithTask(ctx, mi, store.Attrs{}, jobOpts)
	}

	return nil
}

// DownloadingHelpersDequeuePendingDownloadTasks/1
func (a *App) DownloadingHelpersDequeuePendingDownloadTasks(ctx context.Context, source *store.Source) error {
	mediaItems, err := a.ListPendingMediaItemsFor(ctx, source)
	if err != nil {
		return err
	}

	for _, mi := range mediaItems {
		err := a.DeletePendingTasksFor(ctx, mi, nil, store.KW{})
		if err != nil {
			return err
		}
	}

	return nil
}

// DownloadingHelpersKickoffDownloadIfPending/2
func (a *App) DownloadingHelpersKickoffDownloadIfPending(ctx context.Context, mediaItem *store.MediaItem, jobOpts store.KW) (*store.Task, error) {
	// Preload the source
	mediaItem, err := a.mediaPreloadSource(ctx, mediaItem)
	if err != nil {
		return nil, err
	}

	isPending, err := a.PendingDownload(ctx, mediaItem)
	if err != nil {
		return nil, err
	}

	if mediaItem.Source.DownloadMedia && isPending {
		slog.Info(fmt.Sprintf("Kicking off download for media item #%d (%s)", mediaItem.ID, mediaItem.MediaID))

		return a.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, store.Attrs{}, jobOpts)
	}

	return nil, ErrShouldNotDownload
}

// DownloadingHelpersKickoffRedownloadForExistingMedia/1
func (a *App) DownloadingHelpersKickoffRedownloadForExistingMedia(ctx context.Context, source *store.Source) ([]interface{}, error) {
	q := store.MediaQueryNew().
		RequireAssoc("media_profile").
		Where(store.MediaQueryForSource(source.ID)).
		Where(store.MediaQueryDownloaded()).
		Where(store.Not(store.MediaQueryDownloadPrevented()))

	mediaItems, err := store.All[store.MediaItem](ctx, a.Q(ctx), q)
	if err != nil {
		return nil, err
	}

	results := make([]interface{}, 0)
	for _, mi := range mediaItems {
		task, err := a.MediaDownloadWorkerKickoffWithTask(ctx, mi, store.Attrs{}, store.KW{})
		if err != nil {
			results = append(results, store.Attrs{"error": err})
		} else {
			results = append(results, task)
		}
	}

	return results, nil
}

// mediaPreloadSource is a helper to preload the source for a media item
func (a *App) mediaPreloadSource(ctx context.Context, mi *store.MediaItem) (*store.MediaItem, error) {
	if mi.Source != nil {
		return mi, nil
	}

	source, err := a.GetSource(ctx, mi.SourceID)
	if err != nil {
		return nil, err
	}

	mi.Source = source
	return mi, nil
}
