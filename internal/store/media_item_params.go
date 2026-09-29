package store

import (
	"context"
	"database/sql"
	"net/url"
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

// mediaItemColumn is one submitted column: its name and its value (nil for
// NULL).
type mediaItemColumn struct {
	name string
	val  any
}

func mediaItemCol[T any](cols []mediaItemColumn, name string, v *T, clear bool) []mediaItemColumn {
	switch {
	case clear:
		return append(cols, mediaItemColumn{name, nil})
	case v != nil:
		return append(cols, mediaItemColumn{name, *v})
	}
	return cols
}

// columns lists the submitted columns in the order the changeset applies them.
func (p MediaItemParams) columns() []mediaItemColumn {
	var c []mediaItemColumn
	c = mediaItemCol(c, "playlist_index", p.PlaylistIndex, false)
	c = mediaItemCol(c, "title", p.Title, p.Clear&ClearTitle != 0)
	c = mediaItemCol(c, "media_id", p.MediaID, false)
	c = mediaItemCol(c, "description", p.Description, p.Clear&ClearDescription != 0)
	c = mediaItemCol(c, "original_url", p.OriginalURL, false)
	c = mediaItemCol(c, "livestream", p.Livestream, p.Clear&ClearLivestream != 0)
	c = mediaItemCol(c, "source_id", p.SourceID, false)
	c = mediaItemCol(c, "short_form_content", p.ShortFormContent, p.Clear&ClearShortFormContent != 0)
	c = mediaItemCol(c, "uploaded_at", p.UploadedAt, p.Clear&ClearUploadedAt != 0)
	c = mediaItemCol(c, "upload_date_index", p.UploadDateIndex, false)
	c = mediaItemCol(c, "duration_seconds", p.DurationSeconds, false)
	c = mediaItemCol(c, "predicted_media_filepath", p.PredictedMediaFilepath, false)
	c = mediaItemCol(c, "media_downloaded_at", p.MediaDownloadedAt, false)
	c = mediaItemCol(c, "media_filepath", p.MediaFilepath, p.Clear&ClearMediaFilepath != 0)
	c = mediaItemCol(c, "media_size_bytes", p.MediaSizeBytes, false)
	c = mediaItemCol(c, "subtitle_filepaths", p.SubtitleFilepaths, false)
	c = mediaItemCol(c, "thumbnail_filepath", p.ThumbnailFilepath, p.Clear&ClearThumbnailFilepath != 0)
	c = mediaItemCol(c, "metadata_filepath", p.MetadataFilepath, p.Clear&ClearMetadataFilepath != 0)
	c = mediaItemCol(c, "nfo_filepath", p.NfoFilepath, p.Clear&ClearNfoFilepath != 0)
	c = mediaItemCol(c, "last_error", p.LastError, p.Clear&ClearLastError != 0)
	c = mediaItemCol(c, "prevent_download", p.PreventDownload, false)
	c = mediaItemCol(c, "prevent_culling", p.PreventCulling, p.Clear&ClearPreventCulling != 0)
	c = mediaItemCol(c, "culled_at", p.CulledAt, p.Clear&ClearCulledAt != 0)
	c = mediaItemCol(c, "media_redownloaded_at", p.MediaRedownloadedAt, false)
	return c
}

// MediaMetadataParams is the input for a media item's metadata row. Both
// filepaths are required once the row exists.
type MediaMetadataParams struct {
	MetadataFilepath  *string
	ThumbnailFilepath *string
}

// ParseMediaItemParams reads the fields a media item form can submit from
// values (keyed by bare field name; the last value wins, like Plug). Blank
// strings are stored as NULL and blank booleans as false, except where the
// column is nullable or required, which Validate then reports. Unparseable
// booleans are reported in the returned error map.
func ParseMediaItemParams(values url.Values) (MediaItemParams, map[string][]string) {
	var p MediaItemParams
	errs := map[string][]string{}

	last := func(field string) (string, bool) {
		vals := values[field]
		if len(vals) == 0 {
			return "", false
		}
		return vals[len(vals)-1], true
	}
	for field, dst := range map[string]**string{
		"title": &p.Title, "description": &p.Description, "media_id": &p.MediaID,
		"original_url": &p.OriginalURL, "media_filepath": &p.MediaFilepath,
		"thumbnail_filepath": &p.ThumbnailFilepath, "metadata_filepath": &p.MetadataFilepath,
		"nfo_filepath": &p.NfoFilepath, "predicted_media_filepath": &p.PredictedMediaFilepath,
		"last_error": &p.LastError,
	} {
		if raw, ok := last(field); ok {
			*dst = &raw
		}
	}
	for field, b := range map[string]struct {
		dst   **bool
		clear MediaItemClear
	}{
		"livestream":         {&p.Livestream, ClearLivestream},
		"short_form_content": {&p.ShortFormContent, ClearShortFormContent},
		"prevent_download":   {&p.PreventDownload, 0},
		"prevent_culling":    {&p.PreventCulling, ClearPreventCulling},
	} {
		raw, ok := last(field)
		if !ok {
			continue
		}
		switch strings.ToLower(raw) {
		case "true", "1", "on":
			*b.dst = Ptr(true)
		case "false", "0", "off":
			*b.dst = Ptr(false)
		default:
			if strings.TrimSpace(raw) != "" {
				errs[field] = append(errs[field], "is invalid")
			} else if b.clear != 0 {
				p.Clear |= b.clear
			} else {
				*b.dst = Ptr(false)
			}
		}
	}
	return p, errs
}

// castString mirrors how a submitted string is stored: blank becomes NULL.
func castString(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return s
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
		title = castString(p.Title)
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
	if p.Title != nil && p.Clear&ClearTitle == 0 {
		if t := castString(p.Title); t != nil && !equalValues(existing.Title, *t) && strings.HasPrefix(*t, "youtube video #") {
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

// putIfChanged records v as field's new value unless it equals the current one.
func putIfChanged(cs *Changeset, field string, v any) {
	cv, err := castValue(cs.fields[field], v)
	if err != nil {
		cs.AddError(field, "is invalid", map[string]any{"validation": "cast"})
		return
	}
	if equalValues(cs.dataValue(field), cv) {
		return
	}
	cs.Changes[field] = cv
}

// mediaItemChangeset applies p to mediaItem the way Insert and Update expect:
// changed columns, the metadata row, the generated uuid and upload_date_index,
// and p's validation errors.
func (s *Store) mediaItemChangeset(ctx context.Context, mediaItem *MediaItem, p MediaItemParams) (*Changeset, error) {
	cs := Change(mediaItem, nil)
	for _, c := range p.columns() {
		putIfChanged(cs, c.name, c.val)
	}

	if p.Metadata != nil {
		md := mediaItem.Metadata
		if md == nil {
			md = NewMediaMetadata()
		}
		child := Change(md, nil)
		if p.Metadata.MetadataFilepath != nil {
			putIfChanged(child, "metadata_filepath", *p.Metadata.MetadataFilepath)
		}
		if p.Metadata.ThumbnailFilepath != nil {
			putIfChanged(child, "thumbnail_filepath", *p.Metadata.ThumbnailFilepath)
		}
		for field, msgs := range p.Metadata.validate(mediaItem.Metadata) {
			for _, m := range msgs {
				child.AddError(field, m, map[string]any{"validation": "required"})
			}
		}
		child.UniqueConstraint([]string{"media_item_id"}, "media_item_id")
		cs.assocs = map[string]*Changeset{"metadata": child}
	}

	cs.DynamicDefault("uuid", func(*Changeset) any { return GenerateUUID() })
	if err := mediaItemUpdateUploadDateIndex(ctx, s, cs); err != nil {
		return nil, err
	}
	for field, msgs := range p.validate(mediaItem) {
		for _, m := range msgs {
			cs.AddError(field, m)
		}
	}
	cs.UniqueConstraint([]string{"media_id", "source_id"}, "")
	return cs, nil
}

// mediaItemUpdateUploadDateIndex computes upload_date_index. Run it on new
// records no matter what. The method we delegate to will handle the case where
// `uploaded_at` is unchanged.
func mediaItemUpdateUploadDateIndex(ctx context.Context, s *Store, cs *Changeset) error {
	data, _ := cs.Data.(*MediaItem)
	if data != nil && data.ID == 0 {
		return mediaItemDoUpdateUploadDateIndex(ctx, s, cs)
	}

	// For the update case, we only want to recalculate if the day itself has
	// changed. For instance, this is useful in the migration from
	// `upload_date` to `uploaded_at`.
	if cs.HasChange("uploaded_at") {
		if newUploadedAt, ok := cs.GetChange("uploaded_at").(db.UTCDateTime); ok && mediaItemSameDate(data.UploadedAt.Time, newUploadedAt.Time) {
			return nil
		}
		return mediaItemDoUpdateUploadDateIndex(ctx, s, cs)
	}

	// If the record is persisted and the `uploaded_at` field is not being
	// changed, we don't need to recalculate the index.
	return nil
}

func mediaItemSameDate(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

func mediaItemDoUpdateUploadDateIndex(ctx context.Context, s *Store, cs *Changeset) error {
	uploadedAt, ok := cs.GetChange("uploaded_at").(db.UTCDateTime)
	if !ok {
		return nil
	}

	sourceID, _ := cs.GetField("source_id").(int64)
	source, err := MustOne[Source](ctx, s.Q(ctx), From[Source]().Where(sq.Eq{"sources.id": sourceID}))
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
		Where(sq.And{MediaQueryUploadDateMatches(uploadedAt.Time), MediaQueryForSource(source.ID)}).
		Map(func(b sq.SelectBuilder) sq.SelectBuilder {
			return b.RemoveColumns().Column(aggregator + "(mi.upload_date_index) AS agg")
		})

	currentMax, err := Scalar[sql.NullInt64](ctx, s.Q(ctx), q)
	if err != nil {
		return err
	}

	if !currentMax.Valid {
		cs.PutChange("upload_date_index", defaultIndex)
	} else {
		cs.PutChange("upload_date_index", int(currentMax.Int64)+changeDirection)
	}
	return nil
}
