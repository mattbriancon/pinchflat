package core

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
)

type MediaItem struct {
	ID                     int64                `db:"id"`
	MediaID                string               `db:"media_id"`
	Title                  *string              `db:"title"`
	MediaFilepath          *string              `db:"media_filepath"`
	SourceID               int64                `db:"source_id"`
	SubtitleFilepaths      db.NestedStringArray `db:"subtitle_filepaths"`
	ThumbnailFilepath      *string              `db:"thumbnail_filepath"`
	MetadataFilepath       *string              `db:"metadata_filepath"`
	Livestream             bool                 `db:"livestream"`
	OriginalURL            string               `db:"original_url"`
	MediaDownloadedAt      *db.UTCDateTime      `db:"media_downloaded_at"`
	DetailsUpdatedAt       *db.UTCDateTime      `db:"details_updated_at"`
	Description            *string              `db:"description"`
	MediaSizeBytes         *int64               `db:"media_size_bytes"`
	ShortFormContent       bool                 `db:"short_form_content"`
	UploadedAt             db.UTCDateTime       `db:"uploaded_at"`
	NfoFilepath            *string              `db:"nfo_filepath"`
	UUID                   *string              `db:"uuid"`
	DurationSeconds        *int                 `db:"duration_seconds"`
	PreventDownload        bool                 `db:"prevent_download"`
	CulledAt               *db.UTCDateTime      `db:"culled_at"`
	PreventCulling         *bool                `db:"prevent_culling"`
	MediaRedownloadedAt    *db.UTCDateTime      `db:"media_redownloaded_at"`
	UploadDateIndex        int                  `db:"upload_date_index"`
	PlaylistIndex          int                  `db:"playlist_index"`
	PredictedMediaFilepath *string              `db:"predicted_media_filepath"`
	LastError              *string              `db:"last_error"`
	InsertedAt             db.UTCDateTime       `db:"inserted_at"`
	UpdatedAt              db.UTCDateTime       `db:"updated_at"`

	// Virtual field (populated by queries with FULL OUTER JOIN or similar)
	MatchingSearchTerm *string `db:"matching_search_term,virtual"`

	// Associations
	Source                *Source                `db:"-"`
	Metadata              *MediaMetadata         `db:"-"`
	MediaItemsSearchIndex *MediaItemsSearchIndex `db:"-"`
	Tasks                 []*Task                `db:"-"`
}

func (MediaItem) TableName() string { return "media_items" }

func NewMediaItem() *MediaItem {
	uploadedAt, _ := time.Parse("2006-01-02", "1970-01-01")
	return &MediaItem{
		Livestream:        false,
		ShortFormContent:  false,
		PreventDownload:   false,
		UploadDateIndex:   0,
		PlaylistIndex:     0,
		UploadedAt:        db.UTCDateTime{Time: uploadedAt},
		SubtitleFilepaths: db.NestedStringArray{},
	}
}

var mediaItemAllowedFields = []string{
	"playlist_index",
	"title",
	"media_id",
	"description",
	"original_url",
	"livestream",
	"source_id",
	"short_form_content",
	"uploaded_at",
	"upload_date_index",
	"duration_seconds",
	"predicted_media_filepath",
	"media_downloaded_at",
	"media_filepath",
	"media_size_bytes",
	"subtitle_filepaths",
	"thumbnail_filepath",
	"metadata_filepath",
	"nfo_filepath",
	"last_error",
	"prevent_download",
	"prevent_culling",
	"culled_at",
	"media_redownloaded_at",
}

// Pretty much all the fields captured at index are required.
var mediaItemRequiredFields = []string{
	"uuid",
	"title",
	"original_url",
	"livestream",
	"media_id",
	"source_id",
	"uploaded_at",
	"short_form_content",
}

// MediaItem.changeset/2
//
// Ported as a method on *App (with ctx) because update_upload_date_index
// queries the database (Sources.get_source!/1 and a MediaQuery aggregate).
func MediaItemChangeset(ctx context.Context, a *App, mediaItem *MediaItem, attrs Attrs) *Changeset {
	cs := Cast(mediaItem, attrs, mediaItemAllowedFields)
	cs.CastAssoc(attrs, "metadata", mediaItem.Metadata, func(data any, attrs Attrs) *Changeset {
		md, _ := data.(*MediaMetadata)
		if md == nil {
			md = NewMediaMetadata()
		}
		return MediaMetadataChangeset(md, attrs)
	})
	cs.DynamicDefault("uuid", func(*Changeset) any { return GenerateUUID() })
	cs = mediaItemUpdateUploadDateIndex(ctx, a, cs)
	cs.ValidateRequired(mediaItemRequiredFields...)
	// Validate that the title does NOT start with "youtube video #" since that indicates a restriction by YouTube.
	// See issue #549 for more information.
	cs.ValidateFormat("title", `^(?!youtube video #)`)
	cs.UniqueConstraint([]string{"media_id", "source_id"}, "")
	return cs
}

// MediaItem.filepath_attributes/0
func MediaItemFilepathAttributes() []string {
	return []string{"media_filepath", "thumbnail_filepath", "metadata_filepath", "subtitle_filepaths", "nfo_filepath"}
}

// MediaItem.filepath_attribute_defaults/0
func MediaItemFilepathAttributeDefaults() Attrs {
	out := Attrs{}
	for _, field := range MediaItemFilepathAttributes() {
		if field == "subtitle_filepaths" {
			out[field] = db.NestedStringArray{}
		} else {
			out[field] = nil
		}
	}
	return out
}

// MediaItem.json_exluded_fields/0
func MediaItemJSONExcludedFields() []string {
	return []string{"__meta__", "__struct__", "metadata", "tasks", "media_items_search_index"}
}

// update_upload_date_index/1. Run it on new records no matter what. The
// method we delegate to will handle the case where `uploaded_at` is `nil`.
func mediaItemUpdateUploadDateIndex(ctx context.Context, a *App, cs *Changeset) *Changeset {
	data, _ := cs.Data.(*MediaItem)
	if data != nil && data.ID == 0 {
		return mediaItemDoUpdateUploadDateIndex(ctx, a, cs)
	}

	// For the update case, we only want to recalculate if the day itself has
	// changed. For instance, this is useful in the migration from
	// `upload_date` to `uploaded_at`.
	if cs.HasChange("uploaded_at") {
		oldUploadedAt := data.UploadedAt
		newUploadedAt := cs.GetChange("uploaded_at").(db.UTCDateTime)
		if mediaItemSameDate(oldUploadedAt.Time, newUploadedAt.Time) {
			return cs
		}
		return mediaItemDoUpdateUploadDateIndex(ctx, a, cs)
	}

	// If the record is persisted and the `uploaded_at` field is not being
	// changed, we don't need to recalculate the index.
	return cs
}

func mediaItemSameDate(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

func mediaItemDoUpdateUploadDateIndex(ctx context.Context, a *App, cs *Changeset) *Changeset {
	if !cs.HasChange("uploaded_at") {
		return cs
	}

	sourceID, _ := cs.GetField("source_id").(int64)
	// Repo.get!/2: raises Ecto.NoResultsError if the source doesn't exist.
	source, err := MustOne[Source](ctx, a.Q(ctx), From[Source]().Where(sq.Eq{"sources.id": sourceID}))
	if err != nil {
		panic(err)
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

	uploadedAt := cs.GetChange("uploaded_at").(db.UTCDateTime)
	q := MediaQueryNew().
		Where(sq.And{MediaQueryUploadDateMatches(uploadedAt.Time), MediaQueryForSource(source.ID)}).
		Map(func(b sq.SelectBuilder) sq.SelectBuilder {
			return b.RemoveColumns().Column(aggregator + "(mi.upload_date_index) AS agg")
		})

	currentMax, err := Scalar[sql.NullInt64](ctx, a.Q(ctx), q)
	if err != nil {
		panic(err)
	}

	if !currentMax.Valid {
		cs.PutChange("upload_date_index", defaultIndex)
	} else {
		cs.PutChange("upload_date_index", int(currentMax.Int64)+changeDirection)
	}
	return cs
}
