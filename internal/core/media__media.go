package core

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// Some fields should only be set on insert and not on update.
var fieldsToDropOnUpdate = []string{"playlist_index"}

// ListMediaItems/0
func (a *App) MediaListMediaItems(ctx context.Context) ([]*MediaItem, error) {
	return All[MediaItem](ctx, a.Q(ctx), From[MediaItem]())
}

// ListUpgradeableMediaItems/0
func (a *App) MediaListUpgradeableMediaItems(ctx context.Context) ([]*MediaItem, error) {
	q := MediaQueryNew().
		RequireAssoc("media_profile").
		Where(MediaQueryUpgradeable())
	return All[MediaItem](ctx, a.Q(ctx), q)
}

// ListPendingMediaItemsFor/1
func (a *App) MediaListPendingMediaItemsFor(ctx context.Context, source *Source) ([]*MediaItem, error) {
	q := MediaQueryNew().
		RequireAssoc("media_profile").
		Where(sq.And{MediaQueryForSource(source.ID), MediaQueryPending()})
	return All[MediaItem](ctx, a.Q(ctx), q)
}

// PendingDownload/1
//
// Intentionally does not take the `download_media` setting of the source
// into account.
func (a *App) MediaPendingDownload(ctx context.Context, mediaItem *MediaItem) (bool, error) {
	if _, err := a.PreloadMediaItemSourceAndProfile(ctx, mediaItem); err != nil {
		return false, err
	}

	q := MediaQueryNew().
		RequireAssoc("media_profile").
		Where(sq.And{sq.Eq{"mi.id": mediaItem.ID}, MediaQueryPending()})

	row, err := One[MediaItem](ctx, a.Q(ctx), q)
	if err != nil {
		return false, err
	}
	return row != nil, nil
}

// Search/1 and Search/2
//
// Has explicit handling for blank search terms because SQLite doesn't like
// empty MATCH clauses.
func (a *App) MediaSearch(ctx context.Context, searchTerm string, opts KW) ([]*MediaItem, error) {
	if searchTerm == "" {
		return []*MediaItem{}, nil
	}

	limit := 50
	if v, ok := opts.Get("limit"); ok {
		if n, ok := v.(int); ok {
			limit = n
		}
	}

	term := searchTerm
	q := MediaQueryNew().
		MatchingSearchTerm(&term).
		Map(func(b sq.SelectBuilder) sq.SelectBuilder { return MaybeLimit(b, &limit) })

	return All[MediaItem](ctx, a.Q(ctx), q)
}

// GetMediaItem/1
func (a *App) MediaGetMediaItem(ctx context.Context, id int64) (*MediaItem, error) {
	return Get[MediaItem](ctx, a.Q(ctx), id)
}

// CreateMediaItem/1
func (a *App) MediaCreateMediaItem(ctx context.Context, attrs Attrs) (*MediaItem, error) {
	cs := MediaItemChangeset(ctx, a, NewMediaItem(), attrs)
	return Insert[MediaItem](ctx, a.Q(ctx), cs)
}

// CreateMediaItemFromBackendAttrs/2
//
// Unlike CreateMediaItem, this will attempt an update if the media_item
// already exists. This is so that future indexing can pick up attributes
// that we may not have asked for in the past (eg: uploaded_at).
func (a *App) MediaCreateMediaItemFromBackendAttrs(ctx context.Context, source *Source, mediaAttrsStruct any) (*MediaItem, error) {
	attrs := Attrs{"source_id": source.ID}
	for k, v := range mediaItemAttrsFromBackendStruct(mediaAttrsStruct) {
		attrs[k] = v
	}

	cs := MediaItemChangeset(ctx, a, NewMediaItem(), attrs)
	return mediaItemInsertOnConflict(ctx, a, cs, attrs)
}

// UpdateMediaItem/2
func (a *App) MediaUpdateMediaItem(ctx context.Context, mediaItem *MediaItem, attrs Attrs) (*MediaItem, error) {
	updateAttrs := Attrs{}
	for k, v := range attrs {
		updateAttrs[k] = v
	}
	for _, f := range fieldsToDropOnUpdate {
		delete(updateAttrs, f)
	}

	cs := MediaItemChangeset(ctx, a, mediaItem, updateAttrs)
	return Update[MediaItem](ctx, a.Q(ctx), cs)
}

// DeleteMediaItem/1 and DeleteMediaItem/2
//
// Deletes a media_item, its associated tasks, and our internal metadata
// files. Can optionally delete the media_item's media files (media,
// thumbnail, subtitles, etc).
func (a *App) MediaDeleteMediaItem(ctx context.Context, mediaItem *MediaItem, opts KW) (*MediaItem, error) {
	deleteFiles := opts.Bool("delete_files")

	if err := a.TasksDeleteTasksFor(ctx, mediaItem, nil, obanlite.AllStates); err != nil {
		return nil, err
	}

	if deleteFiles {
		if err := mediaDoDeleteMediaFiles(ctx, mediaItem); err != nil {
			return nil, err
		}
		// The Elixir code doesn't check the result of this call.
		_ = a.UserScripts.Run(ctx, "media_deleted", mediaItem)
	}

	// Should delete these no matter what
	// The Elixir code doesn't check the result of this call.
	_ = a.mediaDeleteInternalMetadataFiles(ctx, mediaItem)

	if err := Delete[MediaItem](ctx, a.Q(ctx), mediaItem); err != nil {
		return nil, err
	}
	return mediaItem, nil
}

// DeleteMediaFiles/1 and DeleteMediaFiles/2
//
// Deletes the tasks and media files associated with a media_item but leaves
// the media_item in the database. Does not delete anything to do with
// associated metadata.
//
// addlAttrs will be merged into the media_item before it is updated. Useful
// for setting things like `prevent_download` and `culled_at`, if wanted.
func (a *App) MediaDeleteMediaFiles(ctx context.Context, mediaItem *MediaItem, addlAttrs Attrs) (*MediaItem, error) {
	filepathAttrs := MediaItemFilepathAttributeDefaults()

	if err := a.TasksDeleteTasksFor(ctx, mediaItem, nil, obanlite.AllStates); err != nil {
		return nil, err
	}
	if err := mediaDoDeleteMediaFiles(ctx, mediaItem); err != nil {
		return nil, err
	}
	// The Elixir code doesn't check the result of this call.
	_ = a.UserScripts.Run(ctx, "media_deleted", mediaItem)

	mergedAttrs := Attrs{}
	for k, v := range filepathAttrs {
		mergedAttrs[k] = v
	}
	for k, v := range addlAttrs {
		mergedAttrs[k] = v
	}

	return a.MediaUpdateMediaItem(ctx, mediaItem, mergedAttrs)
}

// ChangeMediaItem/1 and ChangeMediaItem/2
func (a *App) MediaChangeMediaItem(ctx context.Context, mediaItem *MediaItem, attrs Attrs) *Changeset {
	return MediaItemChangeset(ctx, a, mediaItem, attrs)
}

// ComputeAndSaveMediaFilesize fetches the on-disk size of a media item's
// file and saves it to the database.
func (a *App) ComputeAndSaveMediaFilesize(ctx context.Context, mediaItem *MediaItem) (*MediaItem, error) {
	if mediaItem.MediaFilepath == nil {
		return nil, fmt.Errorf("media_filepath is nil")
	}

	stat, err := os.Stat(*mediaItem.MediaFilepath)
	if err != nil {
		return nil, err
	}

	return a.MediaUpdateMediaItem(ctx, mediaItem, Attrs{
		"media_size_bytes": stat.Size(),
	})
}

// do_delete_media_files/1
func mediaDoDeleteMediaFiles(ctx context.Context, mediaItem *MediaItem) error {
	var paths []string
	for _, field := range MediaItemFilepathAttributes() {
		switch field {
		case "subtitle_filepaths":
			for _, pair := range mediaItem.SubtitleFilepaths {
				if len(pair) >= 2 {
					paths = append(paths, pair[1])
				}
			}
		case "media_filepath":
			if mediaItem.MediaFilepath != nil {
				paths = append(paths, *mediaItem.MediaFilepath)
			}
		case "thumbnail_filepath":
			if mediaItem.ThumbnailFilepath != nil {
				paths = append(paths, *mediaItem.ThumbnailFilepath)
			}
		case "metadata_filepath":
			if mediaItem.MetadataFilepath != nil {
				paths = append(paths, *mediaItem.MetadataFilepath)
			}
		case "nfo_filepath":
			if mediaItem.NfoFilepath != nil {
				paths = append(paths, *mediaItem.NfoFilepath)
			}
		}
	}

	// Mirrors Elixir's Enum.each/2, which discards each call's return value
	// (including "file not found" errors) rather than aborting the deletion.
	for _, p := range paths {
		_ = fsutil.DeleteFileAndRemoveEmptyDirs(p)
	}
	return nil
}

// delete_internal_metadata_files/1
func (a *App) mediaDeleteInternalMetadataFiles(ctx context.Context, mediaItem *MediaItem) error {
	if _, err := a.PreloadMediaItemMetadata(ctx, mediaItem); err != nil {
		return err
	}

	metadata := mediaItem.Metadata
	if metadata == nil {
		metadata = NewMediaMetadata()
	}

	for _, field := range MediaMetadataFilepathAttributes() {
		var path string
		switch field {
		case "metadata_filepath":
			path = metadata.MetadataFilepath
		case "thumbnail_filepath":
			path = metadata.ThumbnailFilepath
		}
		if path == "" {
			continue
		}
		// Mirrors Elixir's Enum.each/2, which discards each call's return
		// value rather than aborting on a "file not found" error.
		_ = fsutil.DeleteFileAndRemoveEmptyDirs(path)
	}
	return nil
}

// mediaItemAttrsFromBackendStruct is Map.from_struct(media_attrs_struct):
// converts a *YtDlpMedia (Pinchflat.YtDlp.Media) into an Attrs map.
func mediaItemAttrsFromBackendStruct(mediaAttrsStruct any) Attrs {
	var v *YtDlpMedia
	switch m := mediaAttrsStruct.(type) {
	case *YtDlpMedia:
		v = m
	case YtDlpMedia:
		v = &m
	default:
		panic(fmt.Sprintf("create_media_item_from_backend_attrs/2: unsupported media_attrs_struct %T", mediaAttrsStruct))
	}

	return Attrs{
		"media_id":                 v.MediaID,
		"title":                    v.Title,
		"description":              v.Description,
		"original_url":             v.OriginalURL,
		"livestream":               v.Livestream,
		"short_form_content":       v.ShortFormContent,
		"uploaded_at":              v.UploadedAt,
		"duration_seconds":         v.DurationSeconds,
		"predicted_media_filepath": v.PredictedMediaFilepath,
		"playlist_index":           v.PlaylistIndex,
	}
}

// mediaItemInsertOnConflict is Repo.insert(changeset, on_conflict: [set:
// attrs |> Map.drop(@fields_to_drop_on_update) |> Map.to_list()],
// conflict_target: [:source_id, :media_id]).
//
// Uses the package's private repo helpers (columnValues, quoteCols,
// fieldValue, setID, setTimestamp) to build the same INSERT the ordinary
// Insert[T] would, adding an ON CONFLICT clause raw SQL can't avoid.
func mediaItemInsertOnConflict(ctx context.Context, a *App, cs *Changeset, attrs Attrs) (*MediaItem, error) {
	cs.Action = "insert"
	if !cs.Valid() {
		return nil, &ChangesetError{cs}
	}

	rec := cs.Apply().(*MediaItem)
	now := db.Now()
	setTimestamp(rec, "inserted_at", now, true)
	setTimestamp(rec, "updated_at", now, true)

	cols, vals := columnValues(rec, true)
	fields := fieldsOf(reflect.TypeOf(rec).Elem())

	var updateCols []string
	for k := range attrs {
		if k == "playlist_index" {
			continue
		}
		if _, ok := fields[k]; ok {
			updateCols = append(updateCols, k)
		}
	}
	sort.Strings(updateCols) // deterministic SQL text; doesn't affect behaviour

	var setSQL []string
	var setVals []any
	for _, c := range updateCols {
		setSQL = append(setSQL, `"`+c+`" = ?`)
		setVals = append(setVals, fieldValue(rec, fields[c]))
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (source_id, media_id) DO UPDATE SET %s RETURNING id`,
		(*rec).TableName(),
		quoteCols(cols),
		strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", "),
		strings.Join(setSQL, ", "),
	)

	allVals := append(append([]any{}, vals...), setVals...)

	var id int64
	if err := a.Q(ctx).GetContext(ctx, &id, query, allVals...); err != nil {
		return nil, cs.mapConstraintError((*rec).TableName(), err)
	}
	setID(rec, id)

	if err := saveAssocs(ctx, a.Q(ctx), cs, rec); err != nil {
		return nil, err
	}
	return rec, nil
}
