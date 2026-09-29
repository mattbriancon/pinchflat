package store

import (
	"net/url"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/db"
)

// SourceParams is the user-settable input for creating or updating a Source:
// one pointer field per column. A nil field means "not submitted": the column
// keeps its current value (or its default on create). A string field pointing
// at a blank string was submitted blank: required columns reject it and
// nullable ones become NULL. Other columns are submitted as NULL with Clear.
type SourceParams struct {
	CollectionName             *string
	CollectionID               *string
	CollectionType             *SourceCollectionType
	OriginalURL                *string
	MediaProfileID             *int64
	IndexFrequencyMinutes      *int
	DownloadMedia              *bool
	LastIndexedAt              *db.UTCDateTime
	CustomName                 *string
	FastIndex                  *bool
	DownloadCutoffDate         *db.Date
	NfoFilepath                *string
	SeriesDirectory            *string
	FanartFilepath             *string
	PosterFilepath             *string
	BannerFilepath             *string
	TitleFilterRegex           *string
	Description                *string
	RetentionPeriodDays        *int
	OutputPathTemplateOverride *string
	MarkedForDeletionAt        *db.UTCDateTime
	MinDurationSeconds         *int
	MaxDurationSeconds         *int
	Enabled                    *bool
	CookieBehaviour            *SourceCookieBehaviour

	// Clear submits these columns as NULL. For a required column this means
	// the params are invalid ("can't be blank").
	Clear SourceClear

	// Metadata is the nested source_metadata row to insert/update alongside.
	// Blank fields keep the existing row's value.
	Metadata *SourceMetadata

	// parseErrs are the fields ParseSourceParams could not convert.
	parseErrs map[string][]string
}

// SourceClear is a set of columns to submit as NULL.
type SourceClear uint32

const (
	ClearIndexFrequencyMinutes SourceClear = 1 << iota
	ClearDownloadMedia
	ClearFastIndex
	ClearMediaProfileID
	ClearDownloadCutoffDate
	ClearLastIndexedAt
	ClearRetentionPeriodDays
	ClearMinDurationSeconds
	ClearMaxDurationSeconds
	ClearMarkedForDeletionAt
)

const (
	sourceFormName = "source"

	outputPathTemplateOverrideFormat = `\.({{ ?ext ?}}|%\( ?ext ?\)[sS])$`
)

var (
	sourceRequiredInitial = []string{
		"index_frequency_minutes",
		"fast_index",
		"download_media",
		"original_url",
		"media_profile_id",
	}
	sourceRequiredPreInsert = []string{
		"index_frequency_minutes",
		"fast_index",
		"download_media",
		"original_url",
		"media_profile_id",
		"uuid",
		"custom_name",
		"collection_name",
		"collection_id",
		"collection_type",
	}
)

// SourceFormFields are the source columns the source form shows.
func SourceFormFields() []string {
	return []string{
		"enabled", "custom_name", "original_url", "media_profile_id", "index_frequency_minutes",
		"fast_index", "download_media", "cookie_behaviour", "download_cutoff_date",
		"retention_period_days", "min_duration_seconds", "max_duration_seconds",
		"title_filter_regex", "output_path_template_override",
	}
}

// ParseSourceParams reads the source[...] fields of a submitted form. Values
// that can't be converted are reported by Validate as "is invalid".
func ParseSourceParams(form url.Values) SourceParams {
	var p SourceParams
	f := newFormReader(form, sourceFormName)

	f.str("original_url", &p.OriginalURL)
	f.str("custom_name", &p.CustomName)
	f.str("collection_name", &p.CollectionName)
	f.str("collection_id", &p.CollectionID)
	f.str("description", &p.Description)
	f.str("title_filter_regex", &p.TitleFilterRegex)
	f.str("output_path_template_override", &p.OutputPathTemplateOverride)
	f.str("series_directory", &p.SeriesDirectory)
	f.str("nfo_filepath", &p.NfoFilepath)
	f.str("poster_filepath", &p.PosterFilepath)
	f.str("fanart_filepath", &p.FanartFilepath)
	f.str("banner_filepath", &p.BannerFilepath)

	f.boolOrFalse("enabled", &p.Enabled)
	b, st := f.boolean("fast_index")
	setOrClear(&p.FastIndex, b, st, &p.Clear, ClearFastIndex)
	b, st = f.boolean("download_media")
	setOrClear(&p.DownloadMedia, b, st, &p.Clear, ClearDownloadMedia)

	n, st := f.integer("index_frequency_minutes")
	setOrClear(&p.IndexFrequencyMinutes, n, st, &p.Clear, ClearIndexFrequencyMinutes)
	n, st = f.integer("retention_period_days")
	setOrClear(&p.RetentionPeriodDays, n, st, &p.Clear, ClearRetentionPeriodDays)
	n, st = f.integer("min_duration_seconds")
	setOrClear(&p.MinDurationSeconds, n, st, &p.Clear, ClearMinDurationSeconds)
	n, st = f.integer("max_duration_seconds")
	setOrClear(&p.MaxDurationSeconds, n, st, &p.Clear, ClearMaxDurationSeconds)
	id, st := f.integer64("media_profile_id")
	setOrClear(&p.MediaProfileID, id, st, &p.Clear, ClearMediaProfileID)
	d, st := f.date("download_cutoff_date")
	setOrClear(&p.DownloadCutoffDate, d, st, &p.Clear, ClearDownloadCutoffDate)

	enumeration(f, "collection_type", &p.CollectionType, SourceCollectionTypeChannel, SourceCollectionTypePlaylist)
	enumeration(f, "cookie_behaviour", &p.CookieBehaviour,
		SourceCookieBehaviourDisabled, SourceCookieBehaviourWhenNeeded, SourceCookieBehaviourAllOperations)

	p.parseErrs = f.errs
	return p
}

// apply returns a copy of existing with p applied, and the columns that
// changed. A blank custom_name defaults to the collection_name, and a missing
// uuid is generated.
func (p SourceParams) apply(existing *Source) (*Source, changes) {
	next := *existing
	c := changes{}
	clear := func(flag SourceClear) bool { return p.Clear&flag != 0 }

	setValue(c, "enabled", &next.Enabled, p.Enabled)
	setValue(c, "collection_type", &next.CollectionType, p.CollectionType)
	setValue(c, "cookie_behaviour", &next.CookieBehaviour, p.CookieBehaviour)
	setValue(c, "download_media", &next.DownloadMedia, p.DownloadMedia)
	setValue(c, "fast_index", &next.FastIndex, p.FastIndex)
	setValue(c, "index_frequency_minutes", &next.IndexFrequencyMinutes, p.IndexFrequencyMinutes)
	setValue(c, "media_profile_id", &next.MediaProfileID, p.MediaProfileID)
	setString(c, "collection_name", &next.CollectionName, p.CollectionName)
	setString(c, "collection_id", &next.CollectionID, p.CollectionID)
	setString(c, "custom_name", &next.CustomName, p.CustomName)
	setString(c, "original_url", &next.OriginalURL, p.OriginalURL)

	setNullableString(c, "description", &next.Description, p.Description, false)
	setNullableString(c, "nfo_filepath", &next.NfoFilepath, p.NfoFilepath, false)
	setNullableString(c, "poster_filepath", &next.PosterFilepath, p.PosterFilepath, false)
	setNullableString(c, "fanart_filepath", &next.FanartFilepath, p.FanartFilepath, false)
	setNullableString(c, "banner_filepath", &next.BannerFilepath, p.BannerFilepath, false)
	setNullableString(c, "series_directory", &next.SeriesDirectory, p.SeriesDirectory, false)
	setNullableString(c, "title_filter_regex", &next.TitleFilterRegex, p.TitleFilterRegex, false)
	setNullableString(c, "output_path_template_override", &next.OutputPathTemplateOverride, p.OutputPathTemplateOverride, false)
	setNullable(c, "retention_period_days", &next.RetentionPeriodDays, p.RetentionPeriodDays, clear(ClearRetentionPeriodDays))
	setNullable(c, "min_duration_seconds", &next.MinDurationSeconds, p.MinDurationSeconds, clear(ClearMinDurationSeconds))
	setNullable(c, "max_duration_seconds", &next.MaxDurationSeconds, p.MaxDurationSeconds, clear(ClearMaxDurationSeconds))
	setNullableTime(c, "last_indexed_at", &next.LastIndexedAt, timeOf(p.LastIndexedAt), clear(ClearLastIndexedAt))
	setNullableTime(c, "marked_for_deletion_at", &next.MarkedForDeletionAt, timeOf(p.MarkedForDeletionAt), clear(ClearMarkedForDeletionAt))
	switch {
	case clear(ClearDownloadCutoffDate):
		if next.DownloadCutoffDate != nil {
			c["download_cutoff_date"] = true
			next.DownloadCutoffDate = nil
		}
	case p.DownloadCutoffDate != nil:
		if d := truncateDate(p.DownloadCutoffDate.Time); next.DownloadCutoffDate == nil || !next.DownloadCutoffDate.Equal(d.Time) {
			c["download_cutoff_date"] = true
			next.DownloadCutoffDate = &d
		}
	}

	if next.CustomName == "" {
		setString(c, "custom_name", &next.CustomName, &next.CollectionName)
	}
	if next.UUID == nil {
		next.UUID = Ptr(GenerateUUID())
		c["uuid"] = true
	}
	return &next, c
}

// Apply returns a copy of existing with p applied (defaults included).
func (p SourceParams) Apply(existing *Source) *Source {
	next, _ := p.apply(existing)
	return next
}

// Changed reports which submitted fields differ from existing's values.
func (p SourceParams) Changed(existing *Source) map[string]bool {
	_, c := p.apply(existing)
	return c
}

// Validate checks p against existing and returns field -> messages (empty when
// valid). stage is "initial" (only the fields needed to look the source up)
// or "pre_insert" (everything a stored source needs).
func (p SourceParams) Validate(existing *Source, stage string) map[string][]string {
	errs := map[string][]string{}
	for f, msgs := range p.parseErrs {
		errs[f] = append([]string(nil), msgs...)
	}
	next, changed := p.apply(existing)

	required := sourceRequiredInitial
	if stage == "pre_insert" {
		required = sourceRequiredPreInsert
	}
	blank := map[string]bool{
		"index_frequency_minutes": p.Clear&ClearIndexFrequencyMinutes != 0,
		"fast_index":              p.Clear&ClearFastIndex != 0,
		"download_media":          p.Clear&ClearDownloadMedia != 0,
		"media_profile_id":        p.Clear&ClearMediaProfileID != 0,
		"original_url":            strings.TrimSpace(next.OriginalURL) == "",
		"uuid":                    next.UUID == nil || *next.UUID == "",
		"custom_name":             strings.TrimSpace(next.CustomName) == "",
		"collection_name":         strings.TrimSpace(next.CollectionName) == "",
		"collection_id":           strings.TrimSpace(next.CollectionID) == "",
		"collection_type":         next.CollectionType == "",
	}
	for _, f := range required {
		if len(errs[f]) == 0 && blank[f] {
			addErr(errs, f, "can't be blank")
		}
	}

	if changed["title_filter_regex"] && next.TitleFilterRegex != nil {
		if _, err := db.CompileRegex(*next.TitleFilterRegex); err != nil {
			addErr(errs, "title_filter_regex", "is invalid")
		}
	}

	if changed["min_duration_seconds"] && changed["max_duration_seconds"] &&
		next.MinDurationSeconds != nil && next.MaxDurationSeconds != nil &&
		*next.MinDurationSeconds >= *next.MaxDurationSeconds {
		addErr(errs, "max_duration_seconds", "must be greater than minumum duration")
	}

	if changed["retention_period_days"] && next.RetentionPeriodDays != nil && *next.RetentionPeriodDays < 0 {
		addErr(errs, "retention_period_days", "must be greater than or equal to 0")
	}

	if changed["output_path_template_override"] && next.OutputPathTemplateOverride != nil &&
		!matchesFormat(outputPathTemplateOverrideFormat, *next.OutputPathTemplateOverride) {
		addErr(errs, "output_path_template_override", "must end with .{{ ext }}")
	}

	if changed["original_url"] && !matchesFormat(sourceYoutubeChannelOrPlaylistRegex(), next.OriginalURL) {
		addErr(errs, "original_url", "must be a channel or playlist URL")
	}

	if p.Metadata != nil {
		merged := applySourceMetadata(existing.Metadata, p.Metadata)
		if strings.TrimSpace(merged.MetadataFilepath) == "" {
			addErr(errs, "metadata.metadata_filepath", "can't be blank")
		}
	}

	for f, msgs := range errs {
		if len(msgs) == 0 {
			delete(errs, f)
		}
	}
	return errs
}

func matchesFormat(pattern, s string) bool {
	re, err := db.CompileRegex(pattern)
	if err != nil {
		panic("bad validation pattern: " + err.Error())
	}
	m, _ := re.MatchString(s)
	return m
}

// SourceChanges describes what an update did.
type SourceChanges struct {
	// Changed names the submitted fields whose value differs from Before.
	Changed map[string]bool
	Before  *Source
	After   *Source
}

// applySourceMetadata overlays the non-blank fields of m onto current (which
// may be nil) and returns the result.
func applySourceMetadata(current, m *SourceMetadata) *SourceMetadata {
	rec := SourceMetadata{}
	if current != nil {
		rec = *current
	}
	if m.MetadataFilepath != "" {
		rec.MetadataFilepath = m.MetadataFilepath
	}
	if m.FanartFilepath != nil {
		rec.FanartFilepath = m.FanartFilepath
	}
	if m.PosterFilepath != nil {
		rec.PosterFilepath = m.PosterFilepath
	}
	if m.BannerFilepath != nil {
		rec.BannerFilepath = m.BannerFilepath
	}
	return &rec
}
