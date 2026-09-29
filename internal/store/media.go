package store

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

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

// CreateMediaItem creates a media item from p. Invalid params return a
// *ChangesetError.
func (s *Store) CreateMediaItem(ctx context.Context, p MediaItemParams) (*MediaItem, error) {
	cs, err := s.mediaItemChangeset(ctx, NewMediaItem(), p)
	if err != nil {
		return nil, err
	}
	return Insert[MediaItem](ctx, s.Q(ctx), cs)
}

// UpsertMediaItemFromYtDlp creates or updates a media item from a yt-dlp
// response.
//
// Unlike CreateMediaItem, this will attempt an update if the media_item
// already exists. This is so that future indexing can pick up attributes
// that we may not have asked for in the past (eg: uploaded_at).
func (s *Store) UpsertMediaItemFromYtDlp(ctx context.Context, source *Source, m *ytdlp.Media) (*MediaItem, error) {
	p := MediaItemParams{
		SourceID:               &source.ID,
		MediaID:                &m.MediaID,
		Title:                  &m.Title,
		Description:            &m.Description,
		OriginalURL:            &m.OriginalURL,
		Livestream:             &m.Livestream,
		ShortFormContent:       m.ShortFormContent,
		UploadedAt:             m.UploadedAt,
		DurationSeconds:        m.DurationSeconds,
		PredictedMediaFilepath: &m.PredictedMediaFilepath,
		PlaylistIndex:          m.PlaylistIndex,
	}
	// A response without these is invalid rather than defaulted.
	if m.ShortFormContent == nil {
		p.Clear |= ClearShortFormContent
	}
	if m.UploadedAt == nil {
		p.Clear |= ClearUploadedAt
	}

	cs, err := s.mediaItemChangeset(ctx, NewMediaItem(), p)
	if err != nil {
		return nil, err
	}
	return mediaItemInsertOnConflict(ctx, s, cs)
}

// UpdateMediaItem updates mediaItem with p. Invalid params return a
// *ChangesetError.
func (s *Store) UpdateMediaItem(ctx context.Context, mediaItem *MediaItem, p MediaItemParams) (*MediaItem, error) {
	// Some fields should only be set on insert and not on update.
	p.PlaylistIndex = nil

	cs, err := s.mediaItemChangeset(ctx, mediaItem, p)
	if err != nil {
		return nil, err
	}
	return Update[MediaItem](ctx, s.Q(ctx), cs)
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

	return s.UpdateMediaItem(ctx, mediaItem, MediaItemParams{MediaSizeBytes: Ptr(stat.Size())})
}

// mediaUpsertColumns are the columns an upsert refreshes when the media item
// already exists (everything indexing sets except playlist_index).
var mediaUpsertColumns = []string{
	"description", "duration_seconds", "livestream", "media_id", "original_url",
	"predicted_media_filepath", "short_form_content", "source_id", "title", "uploaded_at",
}

// mediaItemInsertOnConflict inserts cs's record, updating mediaUpsertColumns
// on a (source_id, media_id) conflict.
//
// Uses the package's private repo helpers (columnValues, quoteCols,
// fieldValue, setID, setTimestamp) to build the same INSERT the ordinary
// Insert[T] would, adding an ON CONFLICT clause raw SQL can't avoid.
func mediaItemInsertOnConflict(ctx context.Context, s *Store, cs *Changeset) (*MediaItem, error) {
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

	updateCols := mediaUpsertColumns

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
