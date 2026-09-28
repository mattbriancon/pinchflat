package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// MediaRetentionWorker handles deletion of media items past retention date.

const MediaRetentionWorkerName = "Pinchflat.Downloading.MediaRetentionWorker"

var mediaRetentionWorkerOpts = obanlite.WorkerOpts{
	Queue: "local_data",
	Unique: &obanlite.UniqueOpts{
		Period: obanlite.Infinity,
		States: []string{"available", "scheduled", "retryable", "executing"},
	},
	Tags: []string{"media_item", "local_data"},
}

// MediaRetentionWorkerPerform/1
func (a *App) MediaRetentionWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	if err := mediaRetentionWorkerCullCullableMediaItems(ctx, a); err != nil {
		return err
	}
	if err := mediaRetentionWorkerDeleteMediaItemsFromBeforeCutoff(ctx, a); err != nil {
		return err
	}

	return nil
}

// cull_cullable_media_items/0
func mediaRetentionWorkerCullCullableMediaItems(ctx context.Context, a *App) error {
	cullableMedia, err := store.All[store.MediaItem](ctx, a.Q(ctx),
		store.MediaQueryNew().
			RequireAssoc("source").
			Where(store.MediaQueryCullable()))
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Culling %d media items past their retention date", len(cullableMedia)))

	for _, mediaItem := range cullableMedia {
		// store.Setting `prevent_download` does what it says on the tin, but `culled_at` is purely informational.
		// We don't actually do anything with that in terms of queries and it gets set to nil if the media item
		// gets re-downloaded.
		_, err := a.MediaDeleteMediaFiles(ctx, mediaItem, store.Attrs{
			"prevent_download": true,
			"culled_at":        time.Now().UTC(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// delete_media_items_from_before_cutoff/0
func mediaRetentionWorkerDeleteMediaItemsFromBeforeCutoff(ctx context.Context, a *App) error {
	deletableMedia, err := store.All[store.MediaItem](ctx, a.Q(ctx),
		store.MediaQueryNew().
			RequireAssoc("source").
			Where(store.MediaQueryDeletableBasedOnSourceCutoff()))
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Deleting %d media items that are from before the source cutoff", len(deletableMedia)))

	for _, mediaItem := range deletableMedia {
		// Note that I'm not setting `prevent_download` on the media_item here.
		// That's because cutoff_date can easily change and it's a valid behavior to re-download older
		// media items if the cutoff_date changes.
		// Download is ultimately prevented because `MediaQuery.pending()` only returns media items
		// from after the cutoff date (among other things), so it's not like the media will just immediately
		// be re-downloaded.
		_, err := a.MediaDeleteMediaFiles(ctx, mediaItem, store.Attrs{
			"culled_at": time.Now().UTC(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
