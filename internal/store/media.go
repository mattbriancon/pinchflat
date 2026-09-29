package store

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// Some fields should only be set on insert and not on update.
var fieldsToDropOnUpdate = []string{"playlist_index"}

// ListMediaItems returns every media item.
func (s *Store) ListMediaItems(ctx context.Context) ([]*MediaItem, error) {
	return All[MediaItem](ctx, s.Q(ctx), From[MediaItem]())
}

// ListUpgradeableMediaItems returns media items whose quality can be upgraded.
func (s *Store) ListUpgradeableMediaItems(ctx context.Context) ([]*MediaItem, error) {
	q := MediaQueryNew().
		RequireAssoc("media_profile").
		Where(MediaQueryUpgradeable())
	return All[MediaItem](ctx, s.Q(ctx), q)
}

// ListPendingMediaItemsFor returns source's pending media items.
func (s *Store) ListPendingMediaItemsFor(ctx context.Context, source *Source) ([]*MediaItem, error) {
	q := MediaQueryNew().
		RequireAssoc("media_profile").
		Where(sq.And{MediaQueryForSource(source.ID), MediaQueryPending()})
	return All[MediaItem](ctx, s.Q(ctx), q)
}

// PendingDownload reports whether mediaItem is pending download.
//
// Intentionally does not take the `download_media` setting of the source
// into account.
func (s *Store) PendingDownload(ctx context.Context, mediaItem *MediaItem) (bool, error) {
	if _, err := s.PreloadMediaItemSourceAndProfile(ctx, mediaItem); err != nil {
		return false, err
	}

	q := MediaQueryNew().
		RequireAssoc("media_profile").
		Where(sq.And{sq.Eq{"mi.id": mediaItem.ID}, MediaQueryPending()})

	row, err := One[MediaItem](ctx, s.Q(ctx), q)
	if err != nil {
		return false, err
	}
	return row != nil, nil
}

// Search finds media items matching searchTerm.
//
// A limit <= 0 means the default of 50 results.
//
// Has explicit handling for blank search terms because SQLite doesn't like
// empty MATCH clauses.
func (s *Store) Search(ctx context.Context, searchTerm string, limit int) ([]*MediaItem, error) {
	if searchTerm == "" {
		return []*MediaItem{}, nil
	}

	if limit <= 0 {
		limit = 50
	}

	term := searchTerm
	q := MediaQueryNew().
		MatchingSearchTerm(&term).
		Map(func(b sq.SelectBuilder) sq.SelectBuilder { return MaybeLimit(b, &limit) })

	return All[MediaItem](ctx, s.Q(ctx), q)
}

// GetMediaItem returns ErrNotFound if the media item doesn't exist.
func (s *Store) GetMediaItem(ctx context.Context, id int64) (*MediaItem, error) {
	return Get[MediaItem](ctx, s.Q(ctx), id)
}

// CreateMediaItem creates a media item from attrs.
func (s *Store) CreateMediaItem(ctx context.Context, attrs Attrs) (*MediaItem, error) {
	cs := MediaItemChangeset(ctx, s, NewMediaItem(), attrs)
	return Insert[MediaItem](ctx, s.Q(ctx), cs)
}

// CreateMediaItemFromBackendAttrs creates or updates a media item from a
// *ytdlp.Media (or ytdlp.Media) response.
//
// Unlike CreateMediaItem, this will attempt an update if the media_item
// already exists. This is so that future indexing can pick up attributes
// that we may not have asked for in the past (eg: uploaded_at).
func (s *Store) CreateMediaItemFromBackendAttrs(ctx context.Context, source *Source, mediaAttrsStruct any) (*MediaItem, error) {
	attrs := Attrs{"source_id": source.ID}
	for k, v := range mediaItemAttrsFromBackendStruct(mediaAttrsStruct) {
		attrs[k] = v
	}

	cs := MediaItemChangeset(ctx, s, NewMediaItem(), attrs)
	return mediaItemInsertOnConflict(ctx, s, cs, attrs)
}

// UpdateMediaItem updates mediaItem with attrs.
func (s *Store) UpdateMediaItem(ctx context.Context, mediaItem *MediaItem, attrs Attrs) (*MediaItem, error) {
	updateAttrs := Attrs{}
	for k, v := range attrs {
		updateAttrs[k] = v
	}
	for _, f := range fieldsToDropOnUpdate {
		delete(updateAttrs, f)
	}

	cs := MediaItemChangeset(ctx, s, mediaItem, updateAttrs)
	return Update[MediaItem](ctx, s.Q(ctx), cs)
}

// ChangeMediaItem builds a changeset for mediaItem from attrs.
func (s *Store) ChangeMediaItem(ctx context.Context, mediaItem *MediaItem, attrs Attrs) *Changeset {
	return MediaItemChangeset(ctx, s, mediaItem, attrs)
}

// ComputeAndSaveMediaFilesize fetches the on-disk size of a media item's
// file and saves it to the database.
func (s *Store) ComputeAndSaveMediaFilesize(ctx context.Context, mediaItem *MediaItem) (*MediaItem, error) {
	if mediaItem.MediaFilepath == nil {
		return nil, fmt.Errorf("media_filepath is nil")
	}

	stat, err := os.Stat(*mediaItem.MediaFilepath)
	if err != nil {
		return nil, err
	}

	return s.UpdateMediaItem(ctx, mediaItem, Attrs{
		"media_size_bytes": stat.Size(),
	})
}

// mediaItemAttrsFromBackendStruct is Map.from_struct(media_attrs_struct):
// converts a *ytdlp.Media (Pinchflat.YtDlp.Media) into an Attrs map.
func mediaItemAttrsFromBackendStruct(mediaAttrsStruct any) Attrs {
	var v *ytdlp.Media
	switch m := mediaAttrsStruct.(type) {
	case *ytdlp.Media:
		v = m
	case ytdlp.Media:
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
func mediaItemInsertOnConflict(ctx context.Context, s *Store, cs *Changeset, attrs Attrs) (*MediaItem, error) {
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
	if err := s.Q(ctx).GetContext(ctx, &id, query, allVals...); err != nil {
		return nil, cs.mapConstraintError((*rec).TableName(), err)
	}
	setID(rec, id)

	if err := saveAssocs(ctx, s.Q(ctx), cs, rec); err != nil {
		return nil, err
	}
	return rec, nil
}
