package core

import (
	"context"
	"database/sql"
	"encoding/json"
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
		PreventCulling:    Ptr(false),
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

	uploadedAtVal := cs.GetChange("uploaded_at")
	if uploadedAtVal == nil {
		return cs
	}

	uploadedAt, ok := uploadedAtVal.(db.UTCDateTime)
	if !ok {
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

// MarshalJSON is `defimpl Jason.Encoder, for: MediaItem` (media_item.ex):
//
//	value
//	|> Repo.preload(:source)
//	|> Map.drop(MediaItem.json_exluded_fields())
//	|> Jason.Encode.map(opts)
//
// json_exluded_fields/0 is `~w(__meta__ __struct__ metadata tasks
// media_items_search_index)a`. details_updated_at is a real column but isn't
// declared in the Ecto schema (see CONVENTIONS.md), so Map.from_struct would
// never have included it either; it's left out here for the same reason.
//
// The nested source (and its nested media_profile) mirror Source's and
// MediaProfile's own Jason.Encoder impls (source.ex, media_profile.ex) so the
// whole tree matches testdata/elixir/golden/user_script_media_item.json.
func (m *MediaItem) MarshalJSON() ([]byte, error) {
	var sourceJSON any
	if m.Source != nil {
		sourceJSON = mediaItemSourceJSON(m.Source)
	}

	return json.Marshal(map[string]any{
		"id":                       m.ID,
		"uuid":                     m.UUID,
		"title":                    m.Title,
		"media_id":                 m.MediaID,
		"description":              m.Description,
		"original_url":             m.OriginalURL,
		"livestream":               m.Livestream,
		"short_form_content":       m.ShortFormContent,
		"media_downloaded_at":      m.MediaDownloadedAt,
		"media_redownloaded_at":    m.MediaRedownloadedAt,
		"uploaded_at":              m.UploadedAt,
		"upload_date_index":        m.UploadDateIndex,
		"duration_seconds":         m.DurationSeconds,
		"playlist_index":           m.PlaylistIndex,
		"predicted_media_filepath": m.PredictedMediaFilepath,
		"media_filepath":           m.MediaFilepath,
		"media_size_bytes":         m.MediaSizeBytes,
		"thumbnail_filepath":       m.ThumbnailFilepath,
		"metadata_filepath":        m.MetadataFilepath,
		"nfo_filepath":             m.NfoFilepath,
		"subtitle_filepaths":       m.SubtitleFilepaths,
		"last_error":               m.LastError,
		"prevent_download":         m.PreventDownload,
		"prevent_culling":          m.PreventCulling,
		"culled_at":                m.CulledAt,
		"matching_search_term":     m.MatchingSearchTerm,
		"source_id":                m.SourceID,
		"inserted_at":              m.InsertedAt,
		"updated_at":               m.UpdatedAt,
		"source":                   sourceJSON,
	})
}

// mediaItemSourceJSON is `defimpl Jason.Encoder, for: Source` (source.ex):
// every schema field except metadata/tasks/media_items, plus the preloaded
// media_profile.
func mediaItemSourceJSON(s *Source) map[string]any {
	var profileJSON any
	if s.MediaProfile != nil {
		profileJSON = mediaItemMediaProfileJSON(s.MediaProfile)
	}

	return map[string]any{
		"id":                            s.ID,
		"collection_name":               s.CollectionName,
		"collection_id":                 s.CollectionID,
		"collection_type":               s.CollectionType,
		"original_url":                  s.OriginalURL,
		"media_profile_id":              s.MediaProfileID,
		"index_frequency_minutes":       s.IndexFrequencyMinutes,
		"download_media":                s.DownloadMedia,
		"last_indexed_at":               s.LastIndexedAt,
		"custom_name":                   s.CustomName,
		"fast_index":                    s.FastIndex,
		"download_cutoff_date":          mediaItemDateJSON(s.DownloadCutoffDate),
		"nfo_filepath":                  s.NfoFilepath,
		"series_directory":              s.SeriesDirectory,
		"fanart_filepath":               s.FanartFilepath,
		"poster_filepath":               s.PosterFilepath,
		"banner_filepath":               s.BannerFilepath,
		"title_filter_regex":            s.TitleFilterRegex,
		"uuid":                          s.UUID,
		"description":                   s.Description,
		"retention_period_days":         s.RetentionPeriodDays,
		"output_path_template_override": s.OutputPathTemplateOverride,
		"marked_for_deletion_at":        s.MarkedForDeletionAt,
		"min_duration_seconds":          s.MinDurationSeconds,
		"max_duration_seconds":          s.MaxDurationSeconds,
		"enabled":                       s.Enabled,
		"cookie_behaviour":              s.CookieBehaviour,
		"inserted_at":                   s.InsertedAt,
		"updated_at":                    s.UpdatedAt,
		"media_profile":                 profileJSON,
	}
}

// mediaItemMediaProfileJSON is `defimpl Jason.Encoder, for: MediaProfile`
// (media_profile.ex): every schema field except sources.
func mediaItemMediaProfileJSON(mp *MediaProfile) map[string]any {
	return map[string]any{
		"id":                      mp.ID,
		"name":                    mp.Name,
		"output_path_template":    mp.OutputPathTemplate,
		"download_subs":           mp.DownloadSubs,
		"download_auto_subs":      mp.DownloadAutoSubs,
		"embed_subs":              mp.EmbedSubs,
		"sub_langs":               mp.SubLangs,
		"download_thumbnail":      mp.DownloadThumbnail,
		"embed_thumbnail":         mp.EmbedThumbnail,
		"shorts_behaviour":        mp.ShortsBehaviour,
		"livestream_behaviour":    mp.LivestreamBehaviour,
		"download_metadata":       mp.DownloadMetadata,
		"embed_metadata":          mp.EmbedMetadata,
		"preferred_resolution":    mp.PreferredResolution,
		"download_nfo":            mp.DownloadNfo,
		"download_source_images":  mp.DownloadSourceImages,
		"sponsorblock_behaviour":  mp.SponsorblockBehaviour,
		"sponsorblock_categories": mp.SponsorblockCategories,
		"redownload_delay_days":   mp.RedownloadDelayDays,
		"marked_for_deletion_at":  mp.MarkedForDeletionAt,
		"media_container":         mp.MediaContainer,
		"audio_track":             mp.AudioTrack,
		"inserted_at":             mp.InsertedAt,
		"updated_at":              mp.UpdatedAt,
	}
}

// mediaItemDateJSON formats a db.Date the way Jason encodes an Ecto :date
// field: "YYYY-MM-DD", not the full timestamp db.Date's embedded time.Time
// would otherwise produce.
func mediaItemDateJSON(d *db.Date) any {
	if d == nil {
		return nil
	}
	return d.UTC().Format("2006-01-02")
}
