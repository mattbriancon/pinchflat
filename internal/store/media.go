package store

import (
	"context"
	"strings"

	sq "github.com/Masterminds/squirrel"
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

// CreateMediaItem creates a media item from p. Invalid params return
// ValidationErrors.
func (s *Store) CreateMediaItem(ctx context.Context, p MediaItemParams) (*MediaItem, error) {
	rec, _, err := s.prepareMediaItem(ctx, NewMediaItem(), p)
	if err != nil {
		return nil, err
	}
	if err := Insert(ctx, s.Q(ctx), rec); err != nil {
		return nil, err
	}
	return rec, s.saveMediaItemMetadata(ctx, rec, p)
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
	// Members-only media would only ever fail to download, so keep it out of
	// pending. A source with cookies may belong to a member, so leave those.
	if m.RequiresMembership && source.CookieBehaviour == SourceCookieBehaviourDisabled {
		p.PreventDownload = Ptr(true)
	}

	rec, _, err := s.prepareMediaItem(ctx, NewMediaItem(), p)
	if err != nil {
		return nil, err
	}

	// On a (source_id, media_id) conflict, refresh everything indexing sets
	// except playlist_index.
	var set []string
	var setVals []any
	for _, c := range mediaUpsertColumns {
		set = append(set, `"`+c+`" = ?`)
		setVals = append(setVals, columnValue(rec, c))
	}
	// prevent_download can be set by indexing but never cleared, so a user's
	// choice survives re-indexing.
	set = append(set, `"prevent_download" = MAX("prevent_download", ?)`)
	setVals = append(setVals, rec.PreventDownload)
	conflict := " ON CONFLICT (source_id, media_id) DO UPDATE SET " + strings.Join(set, ", ")
	if err := insertRow(ctx, s.Q(ctx), rec, conflict, setVals...); err != nil {
		return nil, err
	}
	return rec, s.saveMediaItemMetadata(ctx, rec, p)
}

// UpdateMediaItem updates mediaItem with p. Invalid params return
// ValidationErrors.
func (s *Store) UpdateMediaItem(ctx context.Context, mediaItem *MediaItem, p MediaItemParams) (*MediaItem, error) {
	// Some fields should only be set on insert and not on update.
	p.PlaylistIndex = nil

	rec, changed, err := s.prepareMediaItem(ctx, mediaItem, p)
	if err != nil {
		return nil, err
	}
	if err := Update(ctx, s.Q(ctx), rec, changed.list()...); err != nil {
		return nil, err
	}
	if p.Metadata != nil {
		if err := s.saveMediaMetadata(ctx, rec, p.Metadata, mediaItem.Metadata); err != nil {
			return nil, err
		}
	}
	return rec, nil
}

// saveMediaItemMetadata inserts the metadata row of a freshly inserted media
// item, if p has one.
func (s *Store) saveMediaItemMetadata(ctx context.Context, rec *MediaItem, p MediaItemParams) error {
	if p.Metadata == nil {
		return nil
	}
	return s.saveMediaMetadata(ctx, rec, p.Metadata, nil)
}

// mediaUpsertColumns are the columns an upsert refreshes when the media item
// already exists (everything indexing sets except playlist_index).
var mediaUpsertColumns = []string{
	"description", "duration_seconds", "livestream", "media_id", "original_url",
	"predicted_media_filepath", "short_form_content", "source_id", "title", "uploaded_at",
}
