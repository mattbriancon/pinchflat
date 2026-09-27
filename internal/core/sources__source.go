package core

import (
	"reflect"

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

var sourceInitiallyRequiredFields = []string{
	"index_frequency_minutes",
	"fast_index",
	"download_media",
	"original_url",
	"media_profile_id",
}

var sourcePreInsertRequiredFields = append(
	sourceInitiallyRequiredFields,
	"uuid",
	"custom_name",
	"collection_name",
	"collection_id",
	"collection_type",
)

// Source.changeset/3
func SourceChangeset(source *Source, attrs Attrs, validationStage string) *Changeset {
	requiredFields := sourceInitiallyRequiredFields
	if validationStage == "pre_insert" {
		requiredFields = sourcePreInsertRequiredFields
	}

	return Cast(source, attrs, sourceAllowedFields).
		DynamicDefault("custom_name", func(cs *Changeset) any { return cs.GetField("collection_name") }).
		DynamicDefault("uuid", func(cs *Changeset) any { return GenerateUUID() }).
		ValidateRequired(requiredFields...).
		ValidateTitleRegex().
		ValidateMinAndMaxDurations().
		ValidateNumber("retention_period_days", NumberOpts{GreaterThanOrEqualTo: Num(0)}).
		ValidateFormat("output_path_template_override", `\.({{ ?ext ?}}|%\( ?ext ?\)[sS])$`, "must end with .{{ ext }}").
		ValidateFormat("original_url", sourceYoutubeChannelOrPlaylistRegex(), "must be a channel or playlist URL").
		CastAssoc(attrs, "metadata", source.Metadata, func(current any, childAttrs Attrs) *Changeset {
			m := current.(*SourceMetadata)
			if m == nil {
				m = &SourceMetadata{}
			}
			return SourceMetadataChangeset(m, childAttrs)
		}).
		UniqueConstraint([]string{"collection_id", "media_profile_id", "title_filter_regex"}, "original_url")
}

// validateTitleRegex checks if a title_filter_regex is valid by running
// regexp_like query against SQLite. This is called during changeset validation.
//
// Note: castValue stores pointer-typed fields' changes as their base
// (dereferenced) type (e.g. plain string, not *string), so this reads the
// change with derefString rather than asserting a pointer type.
func (cs *Changeset) ValidateTitleRegex() *Changeset {
	if cs.HasChange("title_filter_regex") {
		v := cs.GetChange("title_filter_regex")
		if v == nil {
			return cs
		}
		regex, ok := derefString(v)
		if !ok {
			return cs
		}
		// We need a DB connection to validate. This is a simplified check.
		// In production, we'd run the regexp_like query. For now, we'll
		// use the Go regex compiler to do a basic check.
		_, err := db.CompileRegex(regex)
		if err != nil {
			cs.AddError("title_filter_regex", "is invalid")
		}
	}
	return cs
}

// validateMinAndMaxDurations checks that min_duration <= max_duration.
// Ports Elixir's `validate_min_and_max_durations/1`, which only looks at
// get_change (this changeset's own changes), not the field's persisted
// value, for both min and max.
func (cs *Changeset) ValidateMinAndMaxDurations() *Changeset {
	minChange := cs.GetChange("min_duration_seconds")
	maxChange := cs.GetChange("max_duration_seconds")

	if minChange == nil || maxChange == nil {
		return cs
	}

	minInt, minOK := derefInt(minChange)
	maxInt, maxOK := derefInt(maxChange)
	if !minOK || !maxOK {
		return cs
	}

	if minInt >= maxInt {
		cs.AddError("max_duration_seconds", "must be greater than minumum duration")
	}

	return cs
}

// derefInt reads an int (or *int) value, dereferencing pointers.
func derefInt(v any) (int, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() < reflect.Int || rv.Kind() > reflect.Int64 {
		return 0, false
	}
	return int(rv.Int()), true
}

// index_frequency_when_fast_indexing/0
func SourceIndexFrequencyWhenFastIndexing() int {
	// 30 days in minutes
	return 60 * 24 * 30
}

// fast_index_frequency/0
func SourceFastIndexFrequency() int {
	// minutes
	return 10
}

// filepath_attributes/0
func SourceFilepathAttributes() []string {
	return []string{"nfo_filepath", "fanart_filepath", "poster_filepath", "banner_filepath"}
}

// youtube_channel_or_playlist_regex/0
func sourceYoutubeChannelOrPlaylistRegex() string {
	// Validate that the original URL is not a video URL
	// Also matches if the string does NOT contain youtube.com or youtu.be. This preserves
	// support for non-youtube sources.
	return `^(?:(?!youtube\.com/(watch|shorts|embed)|youtu\.be).)*$`
}
