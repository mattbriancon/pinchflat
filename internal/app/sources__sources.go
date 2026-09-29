package app

import (
	"context"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// output_path_template/1
func (a *App) SourcesOutputPathTemplate(ctx context.Context, source *store.Source) string {
	if source.OutputPathTemplateOverride != nil && strings.TrimSpace(*source.OutputPathTemplateOverride) != "" {
		return *source.OutputPathTemplateOverride
	}
	if source.MediaProfile != nil {
		return source.MediaProfile.OutputPathTemplate
	}
	// Fallback: preload and try again
	source, _ = a.PreloadSourceMediaProfile(ctx, source)
	return source.MediaProfile.OutputPathTemplate
}

// CreateSource/1 and CreateSource/2. Invalid params come back as a
// store.ValidationErrors error.
func (a *App) SourcesCreateSource(ctx context.Context, p store.SourceParams, runPostCommitTasks bool) (*store.Source, error) {
	blank := store.NewSource()

	// Fail fast before asking yt-dlp anything.
	if errs := p.Validate(blank, "initial"); len(errs) > 0 {
		return nil, store.ValidationErrors(errs)
	}

	p, err := sourcesParamsFromURL(ctx, a, blank, p)
	if err != nil {
		return nil, err
	}
	source, errs, err := a.CreateSource(ctx, p)
	if err != nil {
		return nil, err
	}
	if len(errs) > 0 {
		return nil, store.ValidationErrors(errs)
	}

	if runPostCommitTasks {
		// A new source always gets indexed and has its metadata fetched.
		_, _ = a.SlowIndexingHelpersKickoffIndexingTask(ctx, source, map[string]any{})
		if source.FastIndex {
			_, _ = a.FastIndexingHelpersKickoffIndexingTask(ctx, source)
		}
		_, _ = a.SourceMetadataStorageWorkerKickoffWithTask(ctx, source)
	}
	return source, nil
}

// UpdateSource/2 and UpdateSource/3. Invalid params come back as a
// store.ValidationErrors error.
func (a *App) SourcesUpdateSource(ctx context.Context, source *store.Source, p store.SourceParams, runPostCommitTasks bool) (*store.Source, error) {

	// Fail fast before asking yt-dlp anything.
	if errs := p.Validate(source, "initial"); len(errs) > 0 {
		return nil, store.ValidationErrors(errs)
	}

	p, err := sourcesParamsFromURL(ctx, a, source, p)
	if err != nil {
		return nil, err
	}
	updated, changes, errs, err := a.UpdateSource(ctx, source, p)
	if err != nil {
		return nil, err
	}
	if len(errs) > 0 {
		return nil, store.ValidationErrors(errs)
	}

	if runPostCommitTasks {
		sourcesHandleUpdateTasks(ctx, a, changes)
	}
	return updated, nil
}

// DeleteSource/1 and DeleteSource/2
func (a *App) SourcesDeleteSource(ctx context.Context, source *store.Source, deleteFiles bool) (*store.Source, error) {
	// Delete tasks (of any state, matching Elixir's Tasks.delete_tasks_for
	// default of Oban.Job.states()).
	_ = a.DeleteTasksFor(ctx, source, nil, obanlite.AllStates)

	// Delete media items
	mediaItems, _ := store.All[store.MediaItem](ctx, a.Q(ctx), store.MediaQueryNew().Where(store.MediaQueryForSource(source.ID)))
	for _, item := range mediaItems {
		_, _ = a.MediaDeleteMediaItem(ctx, item, deleteFiles)
	}

	// Delete source files if requested
	if deleteFiles {
		sourcesDeleteSourceFiles(ctx, source)
	}

	// Always delete metadata files
	sourcesDeleteInternalMetadataFiles(ctx, a, source)

	// Delete the source
	err := store.Delete[store.Source](ctx, a.Q(ctx), source)
	if err != nil {
		return nil, err
	}

	return source, nil
}

// --- Private helpers ---

// sourcesParamsFromURL fills in collection_type/collection_id/collection_name
// from the URL if original_url is changing, and bumps the index frequency for
// fast indexing. Problems talking to yt-dlp come back as a
// store.ValidationErrors error.
func sourcesParamsFromURL(ctx context.Context, a *App, existing *store.Source, p store.SourceParams) (store.SourceParams, error) {
	if p.Changed(existing)["original_url"] {
		applied := p.Apply(existing)
		shouldUseCookies := applied.CookieBehaviour == store.SourceCookieBehaviourAllOperations
		callOpts := ytdlp.CallOptions{UseCookies: shouldUseCookies, SkipSleepInterval: true}

		fail := func(msg string) (store.SourceParams, error) {
			errs := p.Validate(existing, "pre_insert")
			errs["original_url"] = append(errs["original_url"], msg)
			return p, store.ValidationErrors(errs)
		}

		sourceDetails, err := a.MediaCollectionGetSourceDetails(ctx, applied.OriginalURL, nil, callOpts)
		if err != nil {
			return fail("could not fetch source details from URL")
		}
		collection, ok := sourcesExtractCollectionDetails(sourceDetails)
		if !ok {
			return fail("could not fetch source details from URL")
		}
		// Fetched details win over any user-supplied values.
		p.CollectionType = &collection.Type
		p.CollectionID = store.Ptr(store.Deref(collection.ID))
		p.CollectionName = store.Ptr(store.Deref(collection.Name))
	}

	if p.Apply(existing).FastIndex {
		p.IndexFrequencyMinutes = store.Ptr(store.SourceIndexFrequencyWhenFastIndexing())
		p.Clear &^= store.ClearIndexFrequencyMinutes
	}
	return p, nil
}

// sourceCollection is the collection details yt-dlp reports for a source URL.
type sourceCollection struct {
	Type     store.SourceCollectionType
	ID, Name *string
}

// sourcesExtractCollectionDetails determines if the source is a channel or
// playlist. channel_id/playlist_id may be nil (e.g. a playlist has no
// channel_id), so this compares the raw (possibly-nil) values the same way
// Elixir's `==` does.
func sourcesExtractCollectionDetails(details map[string]any) (sourceCollection, bool) {
	playlistID := details["playlist_id"]
	channelID := details["channel_id"]

	if playlistID == nil && channelID == nil {
		return sourceCollection{}, false
	}
	str := func(v any) *string {
		s, ok := v.(string)
		if !ok {
			return nil
		}
		return &s
	}

	if playlistID == channelID {
		return sourceCollection{store.SourceCollectionTypeChannel, str(channelID), str(details["channel_name"])}, true
	}
	return sourceCollection{store.SourceCollectionTypePlaylist, str(playlistID), str(details["playlist_name"])}, true
}

// taskAction is what an update means for a kind of pending task.
type taskAction int

const (
	taskNone taskAction = iota
	taskEnqueue
	taskDequeue
)

// sourcesMediaAction decides what to do with pending download tasks.
func sourcesMediaAction(c store.SourceChanges) taskAction {
	switch {
	case c.Changed["download_media"] && c.After.DownloadMedia && c.After.Enabled,
		c.Changed["enabled"] && c.After.Enabled && c.After.DownloadMedia:
		return taskEnqueue
	case c.Changed["download_media"] && !c.After.DownloadMedia,
		c.Changed["enabled"] && !c.After.Enabled:
		return taskDequeue
	}
	return taskNone
}

// sourcesSlowIndexingAction decides what to do with the slow indexing task.
func sourcesSlowIndexingAction(c store.SourceChanges) taskAction {
	switch {
	case c.Changed["index_frequency_minutes"] && c.After.IndexFrequencyMinutes > 0 && c.After.Enabled,
		c.Changed["enabled"] && c.After.Enabled && c.After.IndexFrequencyMinutes > 0:
		return taskEnqueue
	case c.Changed["index_frequency_minutes"],
		c.Changed["enabled"] && !c.After.Enabled:
		return taskDequeue
	}
	return taskNone
}

// sourcesFastIndexingAction decides what to do with the fast indexing task.
func sourcesFastIndexingAction(c store.SourceChanges) taskAction {
	switch {
	case c.Changed["fast_index"] && c.After.FastIndex && c.After.Enabled,
		c.Changed["enabled"] && c.After.Enabled && c.After.FastIndex:
		return taskEnqueue
	case c.Changed["fast_index"] && !c.After.FastIndex,
		c.Changed["enabled"] && !c.After.Enabled:
		return taskDequeue
	}
	return taskNone
}

// sourcesHandleUpdateTasks enqueues/dequeues tasks based on what changed.
func sourcesHandleUpdateTasks(ctx context.Context, a *App, c store.SourceChanges) {
	source := c.After

	switch sourcesMediaAction(c) {
	case taskEnqueue:
		_ = a.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, nil)
	case taskDequeue:
		_ = a.DownloadingHelpersDequeuePendingDownloadTasks(ctx, source)
	}

	switch sourcesSlowIndexingAction(c) {
	case taskEnqueue:
		_, _ = a.SlowIndexingHelpersKickoffIndexingTask(ctx, source, map[string]any{})
	case taskDequeue:
		// Elixir's SlowIndexingHelpers.delete_indexing_tasks/2 deletes both
		// the fast- and slow-indexing pending tasks, not just the slow one.
		_ = a.SlowIndexingHelpersDeleteIndexingTasks(ctx, source, true)
	}

	switch sourcesFastIndexingAction(c) {
	case taskEnqueue:
		_, _ = a.FastIndexingHelpersKickoffIndexingTask(ctx, source)
	case taskDequeue:
		_ = a.DeletePendingTasksFor(ctx, source, store.Ptr("FastIndexingWorker"), true)
	}

	// Only refetch metadata if the URL changed.
	if c.Changed["original_url"] {
		_, _ = a.SourceMetadataStorageWorkerKickoffWithTask(ctx, source)
	}
}

// sourcesDeleteSourceFiles deletes all source files
func sourcesDeleteSourceFiles(ctx context.Context, source *store.Source) {
	filepath_attrs := store.SourceFilepathAttributes()
	for _, attr := range filepath_attrs {
		var filePath *string
		switch attr {
		case "nfo_filepath":
			filePath = source.NfoFilepath
		case "fanart_filepath":
			filePath = source.FanartFilepath
		case "poster_filepath":
			filePath = source.PosterFilepath
		case "banner_filepath":
			filePath = source.BannerFilepath
		}

		if filePath != nil && *filePath != "" {
			_ = fsutil.DeleteFileAndRemoveEmptyDirs(*filePath)
		}
	}
}

// sourcesDeleteInternalMetadataFiles deletes source metadata files
func sourcesDeleteInternalMetadataFiles(ctx context.Context, a *App, source *store.Source) {
	source, _ = a.PreloadSourceMetadata(ctx, source)
	if source.Metadata == nil {
		return
	}

	metadata := source.Metadata
	filepath_attrs := store.SourceMetadataFilepathAttributes()
	for _, attr := range filepath_attrs {
		var filePath string
		switch attr {
		case "metadata_filepath":
			filePath = metadata.MetadataFilepath
		case "fanart_filepath":
			if metadata.FanartFilepath != nil {
				filePath = *metadata.FanartFilepath
			}
		case "poster_filepath":
			if metadata.PosterFilepath != nil {
				filePath = *metadata.PosterFilepath
			}
		case "banner_filepath":
			if metadata.BannerFilepath != nil {
				filePath = *metadata.BannerFilepath
			}
		}

		if filePath != "" {
			_ = fsutil.DeleteFileAndRemoveEmptyDirs(filePath)
		}
	}
}
