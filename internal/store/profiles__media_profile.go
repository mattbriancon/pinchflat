package store

import (
	"github.com/mattbriancon/pinchflat/internal/db"
)

// MediaProfileSponsorblockBehaviour is an Ecto.Enum for sponsorblock_behaviour
type MediaProfileSponsorblockBehaviour string

const (
	MediaProfileSponsorblockBehaviourDisabled MediaProfileSponsorblockBehaviour = "disabled"
	MediaProfileSponsorblockBehaviourMark     MediaProfileSponsorblockBehaviour = "mark"
	MediaProfileSponsorblockBehaviourRemove   MediaProfileSponsorblockBehaviour = "remove"
)

// MediaProfileShortsBehaviour is an Ecto.Enum for shorts_behaviour
type MediaProfileShortsBehaviour string

const (
	MediaProfileShortsBehaviourInclude MediaProfileShortsBehaviour = "include"
	MediaProfileShortsBehaviourExclude MediaProfileShortsBehaviour = "exclude"
	MediaProfileShortsBehaviourOnly    MediaProfileShortsBehaviour = "only"
)

// MediaProfileLivestreamBehaviour is an Ecto.Enum for livestream_behaviour
type MediaProfileLivestreamBehaviour string

const (
	MediaProfileLivestreamBehaviourInclude MediaProfileLivestreamBehaviour = "include"
	MediaProfileLivestreamBehaviourExclude MediaProfileLivestreamBehaviour = "exclude"
	MediaProfileLivestreamBehaviourOnly    MediaProfileLivestreamBehaviour = "only"
)

// MediaProfilePreferredResolution is an Ecto.Enum for preferred_resolution
type MediaProfilePreferredResolution string

const (
	MediaProfilePreferredResolution4320p MediaProfilePreferredResolution = "4320p"
	MediaProfilePreferredResolution2160p MediaProfilePreferredResolution = "2160p"
	MediaProfilePreferredResolution1440p MediaProfilePreferredResolution = "1440p"
	MediaProfilePreferredResolution1080p MediaProfilePreferredResolution = "1080p"
	MediaProfilePreferredResolution720p  MediaProfilePreferredResolution = "720p"
	MediaProfilePreferredResolution480p  MediaProfilePreferredResolution = "480p"
	MediaProfilePreferredResolution360p  MediaProfilePreferredResolution = "360p"
	MediaProfilePreferredResolutionAudio MediaProfilePreferredResolution = "audio"
)

type MediaProfile struct {
	ID                     int64                              `db:"id"`
	Name                   string                             `db:"name"`
	OutputPathTemplate     string                             `db:"output_path_template"`
	DownloadSubs           bool                               `db:"download_subs"`
	DownloadAutoSubs       bool                               `db:"download_auto_subs"`
	EmbedSubs              bool                               `db:"embed_subs"`
	SubLangs               string                             `db:"sub_langs"`
	DownloadThumbnail      bool                               `db:"download_thumbnail"`
	EmbedThumbnail         bool                               `db:"embed_thumbnail"`
	ShortsBehaviour        MediaProfileShortsBehaviour        `db:"shorts_behaviour" enum:"include,exclude,only"`
	LivestreamBehaviour    MediaProfileLivestreamBehaviour    `db:"livestream_behaviour" enum:"include,exclude,only"`
	DownloadMetadata       bool                               `db:"download_metadata"`
	EmbedMetadata          bool                               `db:"embed_metadata"`
	PreferredResolution    MediaProfilePreferredResolution    `db:"preferred_resolution" enum:"4320p,2160p,1440p,1080p,720p,480p,360p,audio"`
	DownloadNfo            bool                               `db:"download_nfo"`
	DownloadSourceImages   bool                               `db:"download_source_images"`
	SponsorblockBehaviour  *MediaProfileSponsorblockBehaviour `db:"sponsorblock_behaviour" enum:"disabled,mark,remove"`
	SponsorblockCategories db.JSON[[]string]                  `db:"sponsorblock_categories"`
	RedownloadDelayDays    *int                               `db:"redownload_delay_days"`
	MarkedForDeletionAt    *db.UTCDateTime                    `db:"marked_for_deletion_at"`
	MediaContainer         *string                            `db:"media_container"`
	AudioTrack             *string                            `db:"audio_track"`
	InsertedAt             db.UTCDateTime                     `db:"inserted_at"`
	UpdatedAt              db.UTCDateTime                     `db:"updated_at"`

	// Associations
	Sources []*Source `db:"-"`
}

func (MediaProfile) TableName() string { return "media_profiles" }

func NewMediaProfile() *MediaProfile {
	return &MediaProfile{
		OutputPathTemplate:     "/{{ source_custom_name }}/{{ upload_yyyy_mm_dd }} {{ title }}/{{ title }} [{{ id }}].{{ ext }}",
		DownloadSubs:           false,
		DownloadAutoSubs:       false,
		EmbedSubs:              false,
		SubLangs:               "en",
		DownloadThumbnail:      false,
		EmbedThumbnail:         false,
		DownloadSourceImages:   false,
		DownloadMetadata:       false,
		EmbedMetadata:          false,
		DownloadNfo:            false,
		SponsorblockBehaviour:  Ptr(MediaProfileSponsorblockBehaviourDisabled),
		SponsorblockCategories: db.NewJSON([]string{}),
		ShortsBehaviour:        MediaProfileShortsBehaviourInclude,
		LivestreamBehaviour:    MediaProfileLivestreamBehaviourInclude,
		PreferredResolution:    MediaProfilePreferredResolution1080p,
		MediaContainer:         nil,
	}
}

var mediaProfileAllowedFields = []string{
	"name",
	"output_path_template",
	"download_subs",
	"download_auto_subs",
	"embed_subs",
	"sub_langs",
	"download_thumbnail",
	"embed_thumbnail",
	"download_source_images",
	"download_metadata",
	"embed_metadata",
	"download_nfo",
	"sponsorblock_behaviour",
	"sponsorblock_categories",
	"shorts_behaviour",
	"livestream_behaviour",
	"audio_track",
	"preferred_resolution",
	"media_container",
	"redownload_delay_days",
	"marked_for_deletion_at",
}

// MediaProfile.changeset/2
func MediaProfileChangeset(profile *MediaProfile, attrs Attrs) *Changeset {
	return Cast(profile, attrs, mediaProfileAllowedFields).
		ValidateRequired("name", "output_path_template").
		ValidateFormat("output_path_template", mediaProfileExtRegex(), "must end with .{{ ext }}").
		ValidateNumber("redownload_delay_days", NumberOpts{
			GreaterThanOrEqualTo: Num(0),
		}).
		UniqueConstraint([]string{"name"}, "name")
}

// mediaProfileExtRegex returns the regex pattern for output_path_template validation
func mediaProfileExtRegex() string {
	return `\.({{ ?ext ?}}|%\( ?ext ?\)[sS])$`
}
