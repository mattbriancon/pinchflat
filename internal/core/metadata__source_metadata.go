package core

import (
	"github.com/mattbriancon/pinchflat/internal/db"
)

type SourceMetadata struct {
	ID               int64          `db:"id"`
	MetadataFilepath string         `db:"metadata_filepath"`
	SourceID         int64          `db:"source_id"`
	FanartFilepath   *string        `db:"fanart_filepath"`
	PosterFilepath   *string        `db:"poster_filepath"`
	BannerFilepath   *string        `db:"banner_filepath"`
	InsertedAt       db.UTCDateTime `db:"inserted_at"`
	UpdatedAt        db.UTCDateTime `db:"updated_at"`

	// Associations
	Source *Source `db:"-"`
}

func (SourceMetadata) TableName() string { return "source_metadata" }

func NewSourceMetadata() *SourceMetadata {
	return &SourceMetadata{}
}

var sourceMetadataAllowedFields = []string{
	"metadata_filepath",
	"fanart_filepath",
	"poster_filepath",
	"banner_filepath",
}
