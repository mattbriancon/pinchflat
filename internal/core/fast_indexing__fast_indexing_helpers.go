package core

import (
	"context"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// FastIndexingHelpersKickoffIndexingTask/1
func (a *App) FastIndexingHelpersKickoffIndexingTask(ctx context.Context, source *store.Source) (*store.Task, error) {
	_ = a.DeletePendingTasksFor(ctx, source, store.Ptr(FastIndexingWorkerName), store.KW{store.Flag("include_executing")})
	return a.FastIndexingWorkerKickoffWithTask(ctx, source, store.KW{})
}

// FastIndexingHelpersIndexAndKickoffDownloads/1
func (a *App) FastIndexingHelpersIndexAndKickoffDownloads(ctx context.Context, source *store.Source) ([]*store.MediaItem, error) {
	// The media_profile is needed to determine the quality options to _then_ determine a more
	// accurate predicted filepath
	var err error
	source, err = a.PreloadSourceMediaProfile(ctx, source)
	if err != nil {
		return nil, err
	}

	mediaIDs, err := fastIndexingHelpersGetRecentMediaIDs(ctx, a, source)
	if err != nil {
		return nil, err
	}

	existingMediaItems, err := fastIndexingHelpersListMediaItemsByMediaIDFor(ctx, a, source, mediaIDs)
	if err != nil {
		return nil, err
	}

	existingIDs := make(map[string]bool)
	for _, item := range existingMediaItems {
		existingIDs[item.MediaID] = true
	}

	var newMediaIDs []string
	for _, id := range mediaIDs {
		if !existingIDs[id] {
			newMediaIDs = append(newMediaIDs, id)
		}
	}

	var maybeNewMediaItems []*store.MediaItem
	for _, mediaID := range newMediaIDs {
		mediaItem, err := fastIndexingHelpersCreateMediaItemFromMediaID(ctx, a, source, mediaID)
		if err != nil {
			slog.Error("Error creating media item from URL", "media_id", mediaID, "error", err)
			continue
		}

		_, err = a.DownloadingHelpersKickoffDownloadIfPending(ctx, mediaItem, store.KW{store.Opt("priority", 0)})
		if err != nil {
			slog.Error("Error kicking off download", "media_item_id", mediaItem.ID, "error", err)
		}

		maybeNewMediaItems = append(maybeNewMediaItems, mediaItem)
	}

	// Pick up any stragglers. Intentionally has a lower priority than the per-media item
	// kickoff above
	_ = a.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, store.KW{store.Opt("priority", 1)})

	return maybeNewMediaItems, nil
}

func fastIndexingHelpersGetRecentMediaIDs(ctx context.Context, a *App, source *store.Source) ([]string, error) {
	if a.YoutubeApiEnabled(ctx) {
		mediaIDs, err := a.YoutubeApiGetRecentMediaIDs(ctx, source)
		if err == nil {
			return mediaIDs, nil
		}
	}

	return a.YoutubeRssGetRecentMediaIDs(ctx, source)
}

func fastIndexingHelpersListMediaItemsByMediaIDFor(ctx context.Context, a *App, source *store.Source, mediaIDs []string) ([]*store.MediaItem, error) {
	q := store.MediaQueryNew().
		Where(store.MediaQueryForSource(source.ID)).
		Where(sq.Eq{"mi.media_id": mediaIDs})

	return store.All[store.MediaItem](ctx, a.Q(ctx), q)
}

func fastIndexingHelpersCreateMediaItemFromMediaID(ctx context.Context, a *App, source *store.Source, mediaID string) (*store.MediaItem, error) {
	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", mediaID)
	// This is set to :metadata instead of :indexing since this happens _after_ the
	// actual indexing process. In reality, slow indexing is the only thing that
	// should be using :indexing.
	shouldUseCookies := store.UseCookies(source, "metadata")

	commandOpts := store.KW{store.Opt("output", a.DownloadOptionBuilderBuildOutputPathForSource(ctx, source))}
	commandOpts = append(commandOpts, a.DownloadOptionBuilderBuildQualityOptionsForSource(ctx, source)...)

	ytDlpMedia, err := a.YtDlpMediaGetMediaAttributes(ctx, url, commandOpts, store.KW{store.Opt("use_cookies", shouldUseCookies)})
	if err != nil {
		return nil, err
	}

	return a.CreateMediaItemFromBackendAttrs(ctx, source, ytDlpMedia)
}
