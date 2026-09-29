package store

import (
	"context"
	"database/sql"
	"net/url"
	"reflect"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
)

// MediaItemParams is the input for creating or updating a media item: one
// pointer field per settable column. A nil field means "not submitted": the
// column keeps its current value (or its default on create). Setting a
// column to NULL is done with Clear.
type MediaItemParams struct {
	MediaID                *string
	Title                  *string
	Description            *string
	OriginalURL            *string
	Livestream             *bool
	SourceID               *int64
	ShortFormContent       *bool
	UploadedAt             *time.Time
	UploadDateIndex        *int
	DurationSeconds        *int
	PredictedMediaFilepath *string
	MediaDownloadedAt      *time.Time
	MediaFilepath          *string
	MediaSizeBytes         *int64
	SubtitleFilepaths      *db.NestedStringArray
	ThumbnailFilepath      *string
	MetadataFilepath       *string
	NfoFilepath            *string
	LastError              *string
	PreventDownload        *bool
	PreventCulling         *bool
	CulledAt               *time.Time
	MediaRedownloadedAt    *time.Time
	PlaylistIndex          *int

	// Clear submits these columns as NULL. For a required column this means
	// the params are invalid ("can't be blank").
	Clear MediaItemClear

	// Metadata creates or updates the media item's metadata row.
	Metadata *MediaMetadataParams
}

// MediaItemClear is a set of columns to submit as NULL.
type MediaItemClear uint32

const (
	ClearTitle MediaItemClear = 1 << iota
	ClearDescription
	ClearMediaFilepath
	ClearThumbnailFilepath
	ClearMetadataFilepath
	ClearNfoFilepath
	ClearLastError
	ClearCulledAt
	ClearUploadedAt
	ClearShortFormContent
	ClearLivestream
	ClearPreventCulling

	// ClearFilepaths is every filepath column MediaItemFilepathAttributes
	// lists, except subtitle_filepaths (reset that with an empty array).
	ClearFilepaths = ClearMediaFilepath | ClearThumbnailFilepath | ClearMetadataFilepath | ClearNfoFilepath
)

// MediaMetadataParams is the input for a media item's metadata row. Both
// filepaths are required once the row exists.
type MediaMetadataParams struct {
	MetadataFilepath  *string
	ThumbnailFilepath *string
}

// ParseMediaItemParams reads the media_item[...] fields a media item form can
// submit. Blank strings are stored as NULL and blank booleans as false, except
// where the column is nullable or required, which Validate then reports.
// Unparseable booleans are reported in the returned error map.
func ParseMediaItemParams(form url.Values) (MediaItemParams, map[string][]string) {
	var p MediaItemParams
	f := newFormReader(form, "media_item")

	f.str("title", &p.Title)
	f.str("description", &p.Description)
	f.str("media_id", &p.MediaID)
	f.str("original_url", &p.OriginalURL)
	f.str("media_filepath", &p.MediaFilepath)
	f.str("thumbnail_filepath", &p.ThumbnailFilepath)
	f.str("metadata_filepath", &p.MetadataFilepath)
	f.str("nfo_filepath", &p.NfoFilepath)
	f.str("predicted_media_filepath", &p.PredictedMediaFilepath)
	f.str("last_error", &p.LastError)

	b, st := f.boolean("livestream")
	setOrClear(&p.Livestream, b, st, &p.Clear, ClearLivestream)
	b, st = f.boolean("short_form_content")
	setOrClear(&p.ShortFormContent, b, st, &p.Clear, ClearShortFormContent)
	b, st = f.boolean("prevent_culling")
	setOrClear(&p.PreventCulling, b, st, &p.Clear, ClearPreventCulling)
	f.boolOrFalse("prevent_download", &p.PreventDownload)
	return p, f.errs
}

const blank = "can't be blank"

// Validate returns the errors, keyed by field ("metadata.field" for the
// metadata row), that applying p to existing (nil for a new media item) would
// produce. It doesn't cover the unique media_id/source_id constraint, which
// only the database can check.
func (p MediaItemParams) Validate(existing *MediaItem) map[string][]string {
	if existing == nil {
		existing = NewMediaItem()
	}
	errs := map[string][]string{}
	for field, msgs := range p.validate(existing) {
		errs[field] = msgs
	}
	if p.Metadata != nil {
		for field, msgs := range p.Metadata.validate(existing.Metadata) {
			errs["metadata."+field] = msgs
		}
	}
	return errs
}

func (p MediaItemParams) validate(existing *MediaItem) map[string][]string {
	errs := map[string][]string{}
	add := func(field, msg string) { errs[field] = append(errs[field], msg) }

	// The columns whose defaults are never blank (booleans, source_id and
	// uploaded_at) only fail when explicitly cleared.
	title := existing.Title
	if p.Title != nil {
		title = p.Title
	}
	if p.Clear&ClearTitle != 0 {
		title = nil
	}
	if title == nil || strings.TrimSpace(*title) == "" {
		add("title", blank)
	}
	str := func(field string, set *string, cur string) {
		v := cur
		if set != nil {
			v = *set
		}
		if strings.TrimSpace(v) == "" {
			add(field, blank)
		}
	}
	str("original_url", p.OriginalURL, existing.OriginalURL)
	str("media_id", p.MediaID, existing.MediaID)
	if p.Clear&ClearUploadedAt != 0 {
		add("uploaded_at", blank)
	}
	if p.Clear&ClearShortFormContent != 0 {
		add("short_form_content", blank)
	}
	if p.Clear&ClearLivestream != 0 {
		add("livestream", blank)
	}

	// Titles starting with "youtube video #" indicate a restriction by
	// YouTube (see issue #549). Only a changed title is checked.
	if p.Title != nil && p.Clear&ClearTitle == 0 && !isBlank(*p.Title) {
		if t := *p.Title; (existing.Title == nil || *existing.Title != t) && strings.HasPrefix(t, "youtube video #") {
			add("title", "has invalid format")
		}
	}
	return errs
}

func (p *MediaMetadataParams) validate(existing *MediaMetadata) map[string][]string {
	if existing == nil {
		existing = NewMediaMetadata()
	}
	errs := map[string][]string{}
	for _, f := range []struct {
		name string
		set  *string
		cur  string
	}{
		{"metadata_filepath", p.MetadataFilepath, existing.MetadataFilepath},
		{"thumbnail_filepath", p.ThumbnailFilepath, existing.ThumbnailFilepath},
	} {
		v := f.cur
		if f.set != nil {
			v = *f.set
		}
		if strings.TrimSpace(v) == "" {
			errs[f.name] = append(errs[f.name], blank)
		}
	}
	return errs
}

// apply returns a copy of existing with p applied and the columns that
// changed. A missing uuid is generated.
func (p MediaItemParams) apply(existing *MediaItem) (*MediaItem, changes) {
	next := *existing
	c := changes{}
	clear := func(flag MediaItemClear) bool { return p.Clear&flag != 0 }

	setValue(c, "playlist_index", &next.PlaylistIndex, p.PlaylistIndex)
	setString(c, "media_id", &next.MediaID, p.MediaID)
	setString(c, "original_url", &next.OriginalURL, p.OriginalURL)
	setValue(c, "livestream", &next.Livestream, p.Livestream)
	setValue(c, "source_id", &next.SourceID, p.SourceID)
	setValue(c, "short_form_content", &next.ShortFormContent, p.ShortFormContent)
	setTime(c, "uploaded_at", &next.UploadedAt, p.UploadedAt)
	setValue(c, "upload_date_index", &next.UploadDateIndex, p.UploadDateIndex)
	setValue(c, "prevent_download", &next.PreventDownload, p.PreventDownload)
	if p.SubtitleFilepaths != nil && !reflect.DeepEqual(next.SubtitleFilepaths, *p.SubtitleFilepaths) {
		c["subtitle_filepaths"] = true
		next.SubtitleFilepaths = *p.SubtitleFilepaths
	}

	setNullableString(c, "title", &next.Title, p.Title, clear(ClearTitle))
	setNullableString(c, "description", &next.Description, p.Description, clear(ClearDescription))
	setNullableString(c, "predicted_media_filepath", &next.PredictedMediaFilepath, p.PredictedMediaFilepath, false)
	setNullableString(c, "media_filepath", &next.MediaFilepath, p.MediaFilepath, clear(ClearMediaFilepath))
	setNullableString(c, "thumbnail_filepath", &next.ThumbnailFilepath, p.ThumbnailFilepath, clear(ClearThumbnailFilepath))
	setNullableString(c, "metadata_filepath", &next.MetadataFilepath, p.MetadataFilepath, clear(ClearMetadataFilepath))
	setNullableString(c, "nfo_filepath", &next.NfoFilepath, p.NfoFilepath, clear(ClearNfoFilepath))
	setNullableString(c, "last_error", &next.LastError, p.LastError, clear(ClearLastError))
	setNullable(c, "duration_seconds", &next.DurationSeconds, p.DurationSeconds, false)
	setNullable(c, "media_size_bytes", &next.MediaSizeBytes, p.MediaSizeBytes, false)
	setNullable(c, "prevent_culling", &next.PreventCulling, p.PreventCulling, clear(ClearPreventCulling))
	setNullableTime(c, "media_downloaded_at", &next.MediaDownloadedAt, p.MediaDownloadedAt, false)
	setNullableTime(c, "culled_at", &next.CulledAt, p.CulledAt, clear(ClearCulledAt))
	setNullableTime(c, "media_redownloaded_at", &next.MediaRedownloadedAt, p.MediaRedownloadedAt, false)

	if next.UUID == nil {
		next.UUID = Ptr(GenerateUUID())
		c["uuid"] = true
	}
	return &next, c
}

// apply returns a copy of existing (nil for a new row) with p applied and the
// columns that changed.
func (p *MediaMetadataParams) apply(existing *MediaMetadata) (*MediaMetadata, changes) {
	if existing == nil {
		existing = NewMediaMetadata()
	}
	next := *existing
	c := changes{}
	setString(c, "metadata_filepath", &next.MetadataFilepath, p.MetadataFilepath)
	setString(c, "thumbnail_filepath", &next.ThumbnailFilepath, p.ThumbnailFilepath)
	return &next, c
}

// prepareMediaItem applies p to existing and validates the result. It returns
// the new record and its changed columns; invalid params come back as
// ValidationErrors.
func (s *Store) prepareMediaItem(ctx context.Context, existing *MediaItem, p MediaItemParams) (*MediaItem, changes, error) {
	next, c := p.apply(existing)
	if err := s.setUploadDateIndex(ctx, existing, next, c); err != nil {
		return nil, nil, err
	}
	if errs := p.Validate(existing); len(errs) > 0 {
		return nil, nil, ValidationErrors(errs)
	}
	return next, c, nil
}

// saveMediaMetadata inserts or updates the media item's metadata row and
// points parent.Metadata at it.
func (s *Store) saveMediaMetadata(ctx context.Context, parent *MediaItem, p *MediaMetadataParams, current *MediaMetadata) error {
	rec, c := p.apply(current)
	rec.MediaItemID = parent.ID
	var err error
	if rec.ID == 0 {
		err = Insert(ctx, s.Q(ctx), rec)
	} else {
		err = Update(ctx, s.Q(ctx), rec, c.list()...)
	}
	if err != nil {
		return err
	}
	parent.Metadata = rec
	return nil
}

// setUploadDateIndex computes upload_date_index for next when uploaded_at
// changed. New records are always computed; for an update we only want to
// recalculate if the day itself has changed. For instance, this is useful in
// the migration from `upload_date` to `uploaded_at`.
func (s *Store) setUploadDateIndex(ctx context.Context, existing, next *MediaItem, c changes) error {
	if !c["uploaded_at"] {
		return nil
	}
	if existing.ID != 0 && mediaItemSameDate(existing.UploadedAt.Time, next.UploadedAt.Time) {
		return nil
	}

	source, err := MustOne[Source](ctx, s.Q(ctx), From[Source]().Where(sq.Eq{"sources.id": next.SourceID}))
	if err != nil {
		return err
	}

	// Channels should count down from 99, playlists should count up from 0.
	// This reflects the fact that channels prepend new videos to the top of
	// the list and playlists append new videos to the bottom of the list.
	defaultIndex := 0
	aggregator := "MAX"
	changeDirection := 1
	if source.CollectionType == SourceCollectionTypeChannel {
		defaultIndex = 99
		aggregator = "MIN"
		changeDirection = -1
	}
	q := MediaQueryNew().
		Where(sq.And{MediaQueryUploadDateMatches(next.UploadedAt.Time), MediaQueryForSource(source.ID)}).
		Map(func(b sq.SelectBuilder) sq.SelectBuilder {
			return b.RemoveColumns().Column(aggregator + "(mi.upload_date_index) AS agg")
		})

	currentMax, err := Scalar[sql.NullInt64](ctx, s.Q(ctx), q)
	if err != nil {
		return err
	}

	index := defaultIndex
	if currentMax.Valid {
		index = int(currentMax.Int64) + changeDirection
	}
	setValue(c, "upload_date_index", &next.UploadDateIndex, &index)
	return nil
}

func mediaItemSameDate(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}
