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

var sourceMetadataAllowedFields = []string{
	"metadata_filepath",
	"fanart_filepath",
	"poster_filepath",
	"banner_filepath",
}

var sourceMetadataRequiredFields = []string{
	"metadata_filepath",
}

// SourceMetadataChangeset/2
func SourceMetadataChangeset(sourceMetadata *SourceMetadata, attrs Attrs) *Changeset {
	return Cast(sourceMetadata, attrs, sourceMetadataAllowedFields).
		ValidateRequired(sourceMetadataRequiredFields...).
		UniqueConstraint([]string{"source_id"}, "source_id")
}

// SourceMetadataFilepathAttributes/0
func SourceMetadataFilepathAttributes() []string {
	return []string{"metadata_filepath", "fanart_filepath", "poster_filepath", "banner_filepath"}
}
