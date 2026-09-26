package core

import (
	"github.com/mattbriancon/pinchflat/internal/db"
)

type MediaMetadata struct {
	ID                int64          `db:"id"`
	MediaItemID       int64          `db:"media_item_id"`
	MetadataFilepath  string         `db:"metadata_filepath"`
	ThumbnailFilepath string         `db:"thumbnail_filepath"`
	InsertedAt        db.UTCDateTime `db:"inserted_at"`
	UpdatedAt         db.UTCDateTime `db:"updated_at"`

	// Associations
	MediaItem *MediaItem `db:"-"`
}

func (MediaMetadata) TableName() string { return "media_metadata" }

func NewMediaMetadata() *MediaMetadata {
	return &MediaMetadata{}
}

var mediaMetadataAllowedFields = []string{
	"metadata_filepath",
	"thumbnail_filepath",
}
