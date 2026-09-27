package core

import (
	"context"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
)

// FastIndexingHelpersKickoffIndexingTask/1
func (a *App) FastIndexingHelpersKickoffIndexingTask(ctx context.Context, source *Source) (*Task, error) {
	_ = a.TasksDeletePendingTasksFor(ctx, source, Ptr(FastIndexingWorkerName), KW{Flag("include_executing")})
	return a.FastIndexingWorkerKickoffWithTask(ctx, source, KW{})
}

// FastIndexingHelpersIndexAndKickoffDownloads/1
func (a *App) FastIndexingHelpersIndexAndKickoffDownloads(ctx context.Context, source *Source) ([]*MediaItem, error) {
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

	var maybeNewMediaItems []*MediaItem
	for _, mediaID := range newMediaIDs {
		mediaItem, err := fastIndexingHelpersCreateMediaItemFromMediaID(ctx, a, source, mediaID)
		if err != nil {
			slog.Error("Error creating media item from URL", "media_id", mediaID, "error", err)
			continue
		}

		_, err = a.DownloadingHelpersKickoffDownloadIfPending(ctx, mediaItem, KW{Opt("priority", 0)})
		if err != nil {
			slog.Error("Error kicking off download", "media_item_id", mediaItem.ID, "error", err)
		}

		maybeNewMediaItems = append(maybeNewMediaItems, mediaItem)
	}

	// Pick up any stragglers. Intentionally has a lower priority than the per-media item
	// kickoff above
	_ = a.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, KW{Opt("priority", 1)})

	return maybeNewMediaItems, nil
}

func fastIndexingHelpersGetRecentMediaIDs(ctx context.Context, a *App, source *Source) ([]string, error) {
	if a.YoutubeApiEnabled(ctx) {
		mediaIDs, err := a.YoutubeApiGetRecentMediaIDs(ctx, source)
		if err == nil {
			return mediaIDs, nil
		}
	}

	return a.YoutubeRssGetRecentMediaIDs(ctx, source)
}

func fastIndexingHelpersListMediaItemsByMediaIDFor(ctx context.Context, a *App, source *Source, mediaIDs []string) ([]*MediaItem, error) {
	q := MediaQueryNew().
		Where(MediaQueryForSource(source.ID)).
		Where(sq.Eq{"mi.media_id": mediaIDs})

	return All[MediaItem](ctx, a.Q(ctx), q)
}

func fastIndexingHelpersCreateMediaItemFromMediaID(ctx context.Context, a *App, source *Source, mediaID string) (*MediaItem, error) {
	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", mediaID)
	// This is set to :metadata instead of :indexing since this happens _after_ the
	// actual indexing process. In reality, slow indexing is the only thing that
	// should be using :indexing.
	shouldUseCookies := SourcesUseCookies(source, "metadata")

	commandOpts := KW{Opt("output", a.DownloadOptionBuilderBuildOutputPathForSource(ctx, source))}
	commandOpts = append(commandOpts, a.DownloadOptionBuilderBuildQualityOptionsForSource(ctx, source)...)

	ytDlpMedia, err := a.YtDlpMediaGetMediaAttributes(ctx, url, commandOpts, KW{Opt("use_cookies", shouldUseCookies)})
	if err != nil {
		return nil, err
	}

	return a.MediaCreateMediaItemFromBackendAttrs(ctx, source, ytDlpMedia)
}
