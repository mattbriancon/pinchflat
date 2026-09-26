package core

import (
	"github.com/mattbriancon/pinchflat/internal/db"
)

// SourceCollectionType is an Ecto.Enum for collection_type
type SourceCollectionType string

const (
	SourceCollectionTypeChannel  SourceCollectionType = "channel"
	SourceCollectionTypePlaylist SourceCollectionType = "playlist"
)

// SourceCookieBehaviour is an Ecto.Enum for cookie_behaviour
type SourceCookieBehaviour string

const (
	SourceCookieBehaviourDisabled      SourceCookieBehaviour = "disabled"
	SourceCookieBehaviourWhenNeeded    SourceCookieBehaviour = "when_needed"
	SourceCookieBehaviourAllOperations SourceCookieBehaviour = "all_operations"
)

type Source struct {
	ID                         int64                 `db:"id"`
	CollectionName             string                `db:"collection_name"`
	CollectionID               string                `db:"collection_id"`
	CollectionType             SourceCollectionType  `db:"collection_type" enum:"channel,playlist"`
	OriginalURL                string                `db:"original_url"`
	MediaProfileID             int64                 `db:"media_profile_id"`
	IndexFrequencyMinutes      int                   `db:"index_frequency_minutes"`
	DownloadMedia              bool                  `db:"download_media"`
	LastIndexedAt              *db.UTCDateTime       `db:"last_indexed_at"`
	CustomName                 string                `db:"custom_name"`
	FastIndex                  bool                  `db:"fast_index"`
	DownloadCutoffDate         *db.Date              `db:"download_cutoff_date"`
	NfoFilepath                *string               `db:"nfo_filepath"`
	SeriesDirectory            *string               `db:"series_directory"`
	FanartFilepath             *string               `db:"fanart_filepath"`
	PosterFilepath             *string               `db:"poster_filepath"`
	BannerFilepath             *string               `db:"banner_filepath"`
	TitleFilterRegex           *string               `db:"title_filter_regex"`
	UUID                       *string               `db:"uuid"`
	Description                *string               `db:"description"`
	RetentionPeriodDays        *int                  `db:"retention_period_days"`
	OutputPathTemplateOverride *string               `db:"output_path_template_override"`
	MarkedForDeletionAt        *db.UTCDateTime       `db:"marked_for_deletion_at"`
	MinDurationSeconds         *int                  `db:"min_duration_seconds"`
	MaxDurationSeconds         *int                  `db:"max_duration_seconds"`
	Enabled                    bool                  `db:"enabled"`
	CookieBehaviour            SourceCookieBehaviour `db:"cookie_behaviour" enum:"disabled,when_needed,all_operations"`
	InsertedAt                 db.UTCDateTime        `db:"inserted_at"`
	UpdatedAt                  db.UTCDateTime        `db:"updated_at"`

	// Associations
	MediaProfile *MediaProfile   `db:"-"`
	Metadata     *SourceMetadata `db:"-"`
	Tasks        []*Task         `db:"-"`
	MediaItems   []*MediaItem    `db:"-"`
}

func (Source) TableName() string { return "sources" }

func NewSource() *Source {
	return &Source{
		IndexFrequencyMinutes: 60 * 24,
		DownloadMedia:         true,
		FastIndex:             false,
		Enabled:               true,
		CookieBehaviour:       SourceCookieBehaviourDisabled,
	}
}

var sourceAllowedFields = []string{
	"enabled",
	"collection_name",
	"collection_id",
	"collection_type",
	"custom_name",
	"description",
	"nfo_filepath",
	"poster_filepath",
	"fanart_filepath",
	"banner_filepath",
	"series_directory",
	"index_frequency_minutes",
	"fast_index",
	"cookie_behaviour",
	"download_media",
	"last_indexed_at",
	"original_url",
	"download_cutoff_date",
	"retention_period_days",
	"title_filter_regex",
	"media_profile_id",
	"output_path_template_override",
	"marked_for_deletion_at",
	"min_duration_seconds",
	"max_duration_seconds",
}
