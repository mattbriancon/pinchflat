package core

import (
	"context"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// output_path_template/1
func (a *App) SourcesOutputPathTemplate(ctx context.Context, source *Source) string {
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

// use_cookies?/2
func SourcesUseCookies(source *Source, operation string) bool {
	switch source.CookieBehaviour {
	case SourceCookieBehaviourDisabled:
		return false
	case SourceCookieBehaviourAllOperations:
		return true
	case SourceCookieBehaviourWhenNeeded:
		return operation == "indexing" || operation == "error_recovery"
	}
	return false
}

// ListSources/0
func (a *App) SourcesListSources(ctx context.Context) ([]*Source, error) {
	return All[Source](ctx, a.Q(ctx), From[Source]())
}

// ListSourcesFor/1
func (a *App) SourcesListSourcesFor(ctx context.Context, profile *MediaProfile) ([]*Source, error) {
	return All[Source](ctx, a.Q(ctx), From[Source]().Where(sq.Eq{"media_profile_id": profile.ID}))
}

// GetSourceBang/1 - returns ErrNotFound if not found
func (a *App) SourcesGetSource(ctx context.Context, id int64) (*Source, error) {
	return Get[Source](ctx, a.Q(ctx), id)
}

// CreateSource/1 and CreateSource/2
func (a *App) SourcesCreateSource(ctx context.Context, attrs Attrs, opts KW) (*Source, error) {
	runPostCommitTasks := opts.GetOr("run_post_commit_tasks", true).(bool)

	// Initial validation
	cs := a.SourcesChangeSource(ctx, NewSource(), attrs, "initial")
	if !cs.Valid() {
		return Insert[Source](ctx, a.Q(ctx), cs)
	}

	// Build full changeset with API call
	cs = sourcesChangeSourceFromURL(ctx, a, NewSource(), attrs)
	cs = sourcesChangeIndexingFrequency(cs)

	return sourcesCommitAndHandleTasks(ctx, a, cs, runPostCommitTasks)
}

// UpdateSource/2 and UpdateSource/3
func (a *App) SourcesUpdateSource(ctx context.Context, source *Source, attrs Attrs, opts KW) (*Source, error) {
	runPostCommitTasks := opts.GetOr("run_post_commit_tasks", true).(bool)

	// Initial validation
	cs := a.SourcesChangeSource(ctx, source, attrs, "initial")
	if !cs.Valid() {
		return Update[Source](ctx, a.Q(ctx), cs)
	}

	// Build full changeset with API call
	cs = sourcesChangeSourceFromURL(ctx, a, source, attrs)
	cs = sourcesChangeIndexingFrequency(cs)

	return sourcesCommitAndHandleTasks(ctx, a, cs, runPostCommitTasks)
}

// DeleteSource/1 and DeleteSource/2
func (a *App) SourcesDeleteSource(ctx context.Context, source *Source, opts KW) (*Source, error) {
	deleteFiles := opts.Bool("delete_files")

	// Delete tasks (of any state, matching Elixir's Tasks.delete_tasks_for
	// default of Oban.Job.states()).
	_ = a.TasksDeleteTasksFor(ctx, source, nil, obanlite.AllStates)

	// Delete media items
	mediaItems, _ := All[MediaItem](ctx, a.Q(ctx), MediaQueryNew().Where(MediaQueryForSource(source.ID)))
	for _, item := range mediaItems {
		_, _ = a.MediaDeleteMediaItem(ctx, item, opts)
	}

	// Delete source files if requested
	if deleteFiles {
		sourcesDeleteSourceFiles(ctx, source)
	}

	// Always delete metadata files
	sourcesDeleteInternalMetadataFiles(ctx, a, source)

	// Delete the source
	err := Delete[Source](ctx, a.Q(ctx), source)
	if err != nil {
		return nil, err
	}

	return source, nil
}

// ChangeSource/2 and ChangeSource/3
func (a *App) SourcesChangeSource(ctx context.Context, source *Source, attrs Attrs, validationStage string) *Changeset {
	if validationStage == "" {
		validationStage = "pre_insert"
	}
	return SourceChangeset(source, attrs, validationStage)
}

// --- Private helpers ---

// sourcesChangeSourceFromURL builds a fresh, fully (pre_insert) validated
// changeset from attrs and, if original_url changed, fetches source details
// from the URL to fill in collection_type/collection_id/collection_name.
func sourcesChangeSourceFromURL(ctx context.Context, a *App, source *Source, attrs Attrs) *Changeset {
	changeset := a.SourcesChangeSource(ctx, source, attrs, "pre_insert")

	if !changeset.HasChange("original_url") {
		return changeset
	}

	cookieBehaviour := changeset.GetField("cookie_behaviour").(SourceCookieBehaviour)
	shouldUseCookies := cookieBehaviour == SourceCookieBehaviourAllOperations
	addlOpts := KW{Opt("use_cookies", shouldUseCookies), Opt("skip_sleep_interval", true)}

	sourceDetails, err := a.MediaCollectionGetSourceDetails(ctx, changeset.GetChange("original_url").(string), KW{}, addlOpts)
	if err != nil {
		var errMsg string
		switch e := err.(type) {
		case *CommandError:
			errMsg = e.Output
		default:
			errMsg = err.Error()
		}
		changeset.AddError("original_url", "could not fetch source details from URL", map[string]any{"error": errMsg})
		return changeset
	}

	// Parse the response
	collectionChanges := sourcesExtractCollectionDetails(sourceDetails)
	if collectionChanges == nil {
		changeset.AddError("original_url", "could not fetch source details from URL")
		return changeset
	}

	// Elixir rebuilds a brand new (pre_insert validated) changeset from attrs
	// merged with the fetched collection_type/collection_id/collection_name,
	// rather than patching the changeset that was built (and validated)
	// before those fields were known. Merging like this also lets the fetched
	// details win over any conflicting user-supplied values, same as
	// Map.merge(changes, collection_changes).
	mergedAttrs := Attrs{}
	for k, v := range attrs {
		mergedAttrs[k] = v
	}
	for k, v := range collectionChanges {
		mergedAttrs[k] = v
	}

	return a.SourcesChangeSource(ctx, source, mergedAttrs, "pre_insert")
}

// sourcesExtractCollectionDetails determines if the source is a channel or
// playlist. channel_id/playlist_id may be nil (e.g. a playlist has no
// channel_id), so this compares the raw (possibly-nil) interface values the
// same way Elixir's `==` does, rather than requiring both to be strings.
func sourcesExtractCollectionDetails(details map[string]any) map[string]any {
	playlistID := details["playlist_id"]
	channelID := details["channel_id"]

	if playlistID == nil && channelID == nil {
		return nil
	}

	if playlistID == channelID {
		return map[string]any{
			"collection_type": SourceCollectionTypeChannel,
			"collection_id":   channelID,
			"collection_name": details["channel_name"],
		}
	}

	return map[string]any{
		"collection_type": SourceCollectionTypePlaylist,
		"collection_id":   playlistID,
		"collection_name": details["playlist_name"],
	}
}

// sourcesChangeIndexingFrequency adjusts frequency if fast_index is enabled
func sourcesChangeIndexingFrequency(changeset *Changeset) *Changeset {
	fastIndex := changeset.GetField("fast_index").(bool)
	if fastIndex {
		changeset.PutChange("index_frequency_minutes", SourceIndexFrequencyWhenFastIndexing())
	}
	return changeset
}

// sourcesCommitAndHandleTasks inserts/updates and runs post-commit tasks
func sourcesCommitAndHandleTasks(ctx context.Context, a *App, changeset *Changeset, runTasks bool) (*Source, error) {
	var source *Source
	var err error

	if idOf(changeset.Data) == 0 {
		source, err = Insert[Source](ctx, a.Q(ctx), changeset)
	} else {
		source, err = Update[Source](ctx, a.Q(ctx), changeset)
	}

	if err != nil || !runTasks {
		return source, err
	}

	// Run post-commit tasks
	sourcesHandleMediaTasks(ctx, a, changeset, source)
	sourcesHandleIndexingTasks(ctx, a, changeset, source)
	sourcesHandleMetadataStorageTasks(ctx, a, changeset, source)

	return source, nil
}

// sourcesHandleMediaTasks enqueues/dequeues download tasks based on changes
func sourcesHandleMediaTasks(ctx context.Context, a *App, changeset *Changeset, source *Source) {
	// If the changeset is new (not persisted), do nothing
	if changeset.Data.(*Source).ID == 0 {
		return
	}

	currentChanges := changeset.Changes
	applied := changeset.Apply().(*Source)

	case1 := currentChanges["download_media"] != nil && currentChanges["download_media"].(bool) == true &&
		applied.Enabled == true
	case2 := currentChanges["enabled"] != nil && currentChanges["enabled"].(bool) == true &&
		applied.DownloadMedia == true
	case3 := currentChanges["download_media"] != nil && currentChanges["download_media"].(bool) == false
	case4 := currentChanges["enabled"] != nil && currentChanges["enabled"].(bool) == false

	if case1 || case2 {
		_ = a.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, KW{})
	} else if case3 || case4 {
		_ = a.DownloadingHelpersDequeuePendingDownloadTasks(ctx, source)
	}
}

// sourcesHandleIndexingTasks kicks off indexing tasks when needed
func sourcesHandleIndexingTasks(ctx context.Context, a *App, changeset *Changeset, source *Source) {
	// If new, kick off indexing tasks
	if changeset.Data.(*Source).ID == 0 {
		_, _ = a.SlowIndexingHelpersKickoffIndexingTask(ctx, source, Attrs{}, KW{})
		if changeset.GetField("fast_index").(bool) {
			_, _ = a.FastIndexingHelpersKickoffIndexingTask(ctx, source)
		}
		return
	}

	// If persisted, only update if conditions changed
	sourcesUpdateSlowIndexingTask(ctx, a, changeset, source)
	sourcesUpdateFastIndexingTask(ctx, a, changeset, source)
}

// sourcesUpdateSlowIndexingTask manages slow indexing based on changes
func sourcesUpdateSlowIndexingTask(ctx context.Context, a *App, changeset *Changeset, source *Source) {
	currentChanges := changeset.Changes
	applied := changeset.Apply().(*Source)

	case1 := currentChanges["index_frequency_minutes"] != nil &&
		currentChanges["index_frequency_minutes"].(int) > 0 && applied.Enabled == true
	case2 := currentChanges["enabled"] != nil && currentChanges["enabled"].(bool) == true &&
		applied.IndexFrequencyMinutes > 0
	case3 := currentChanges["index_frequency_minutes"] != nil
	case4 := currentChanges["enabled"] != nil && currentChanges["enabled"].(bool) == false

	if case1 || case2 {
		_, _ = a.SlowIndexingHelpersKickoffIndexingTask(ctx, source, Attrs{}, KW{})
	} else if case3 || case4 {
		// Elixir's SlowIndexingHelpers.delete_indexing_tasks/2 deletes both
		// the fast- and slow-indexing pending tasks, not just the slow one.
		_ = a.SlowIndexingHelpersDeleteIndexingTasks(ctx, source, KW{Opt("include_executing", true)})
	}
}

// sourcesUpdateFastIndexingTask manages fast indexing based on changes
func sourcesUpdateFastIndexingTask(ctx context.Context, a *App, changeset *Changeset, source *Source) {
	currentChanges := changeset.Changes
	applied := changeset.Apply().(*Source)

	case1 := currentChanges["fast_index"] != nil && currentChanges["fast_index"].(bool) == true &&
		applied.Enabled == true
	case2 := currentChanges["enabled"] != nil && currentChanges["enabled"].(bool) == true &&
		applied.FastIndex == true
	case3 := currentChanges["fast_index"] != nil && currentChanges["fast_index"].(bool) == false
	case4 := currentChanges["enabled"] != nil && currentChanges["enabled"].(bool) == false

	if case1 || case2 {
		_, _ = a.FastIndexingHelpersKickoffIndexingTask(ctx, source)
	} else if case3 || case4 {
		_ = a.TasksDeletePendingTasksFor(ctx, source, Ptr("FastIndexingWorker"), KW{Opt("include_executing", true)})
	}
}

// sourcesHandleMetadataStorageTasks kicks off metadata storage if needed
func sourcesHandleMetadataStorageTasks(ctx context.Context, a *App, changeset *Changeset, source *Source) {
	// If new, always fetch metadata
	if changeset.Data.(*Source).ID == 0 {
		_, _ = a.SourceMetadataStorageWorkerKickoffWithTask(ctx, source, KW{})
		return
	}

	// If persisted, only fetch if original_url changed
	if changeset.HasChange("original_url") {
		_, _ = a.SourceMetadataStorageWorkerKickoffWithTask(ctx, source, KW{})
	}
}

// sourcesDeleteSourceFiles deletes all source files
func sourcesDeleteSourceFiles(ctx context.Context, source *Source) {
	filepath_attrs := SourceFilepathAttributes()
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
			_ = FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ctx, *filePath)
		}
	}
}

// sourcesDeleteInternalMetadataFiles deletes source metadata files
func sourcesDeleteInternalMetadataFiles(ctx context.Context, a *App, source *Source) {
	source, _ = a.PreloadSourceMetadata(ctx, source)
	if source.Metadata == nil {
		return
	}

	metadata := source.Metadata
	filepath_attrs := SourceMetadataFilepathAttributes()
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
			_ = FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ctx, filePath)
		}
	}
}
