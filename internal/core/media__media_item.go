package core

import (
	"time"

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
	MatchingSearchTerm *string `db:"-"`

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
