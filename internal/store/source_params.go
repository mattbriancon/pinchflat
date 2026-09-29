package store

import (
	"reflect"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/db"
)

// SourceParams is the user-settable input for creating or updating a Source.
// A field is only written when it was submitted (see ParseSourceParams), which
// the params track separately from the value: a submitted nil clears a
// nullable column, while an unsubmitted field leaves it alone.
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

	// Metadata is the nested source_metadata row to insert/update alongside.
	// Blank fields keep the existing row's value.
	Metadata *SourceMetadata

	set       map[string]bool
	uuid      *string // generated, never user-settable
	parseErrs map[string][]string
}

var sourceSpecs = []paramSpec[SourceParams]{
	{name: "enabled", typ: typeOf[bool](), field: func(p *SourceParams) any { return &p.Enabled }},
	{name: "collection_name", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.CollectionName }},
	{name: "collection_id", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.CollectionID }},
	{name: "collection_type", typ: typeOf[SourceCollectionType](), enum: []string{"channel", "playlist"}, field: func(p *SourceParams) any { return &p.CollectionType }},
	{name: "custom_name", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.CustomName }},
	{name: "description", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.Description }},
	{name: "nfo_filepath", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.NfoFilepath }},
	{name: "poster_filepath", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.PosterFilepath }},
	{name: "fanart_filepath", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.FanartFilepath }},
	{name: "banner_filepath", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.BannerFilepath }},
	{name: "series_directory", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.SeriesDirectory }},
	{name: "index_frequency_minutes", typ: typeOf[int](), field: func(p *SourceParams) any { return &p.IndexFrequencyMinutes }},
	{name: "fast_index", typ: typeOf[bool](), field: func(p *SourceParams) any { return &p.FastIndex }},
	{name: "cookie_behaviour", typ: typeOf[SourceCookieBehaviour](), enum: []string{"disabled", "when_needed", "all_operations"}, field: func(p *SourceParams) any { return &p.CookieBehaviour }},
	{name: "download_media", typ: typeOf[bool](), field: func(p *SourceParams) any { return &p.DownloadMedia }},
	{name: "last_indexed_at", typ: typeOf[db.UTCDateTime](), field: func(p *SourceParams) any { return &p.LastIndexedAt }},
	{name: "original_url", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.OriginalURL }},
	{name: "download_cutoff_date", typ: typeOf[db.Date](), field: func(p *SourceParams) any { return &p.DownloadCutoffDate }},
	{name: "retention_period_days", typ: typeOf[int](), field: func(p *SourceParams) any { return &p.RetentionPeriodDays }},
	{name: "title_filter_regex", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.TitleFilterRegex }},
	{name: "media_profile_id", typ: typeOf[int64](), field: func(p *SourceParams) any { return &p.MediaProfileID }},
	{name: "output_path_template_override", typ: typeOf[string](), field: func(p *SourceParams) any { return &p.OutputPathTemplateOverride }},
	{name: "marked_for_deletion_at", typ: typeOf[db.UTCDateTime](), field: func(p *SourceParams) any { return &p.MarkedForDeletionAt }},
	{name: "min_duration_seconds", typ: typeOf[int](), field: func(p *SourceParams) any { return &p.MinDurationSeconds }},
	{name: "max_duration_seconds", typ: typeOf[int](), field: func(p *SourceParams) any { return &p.MaxDurationSeconds }},
}

// SourceFormFields are the source columns a params map may set.
func SourceFormFields() []string {
	names := make([]string, len(sourceSpecs))
	for i, sp := range sourceSpecs {
		names[i] = sp.name
	}
	return names
}

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

const outputPathTemplateOverrideFormat = `\.({{ ?ext ?}}|%\( ?ext ?\)[sS])$`

// ParseSourceParams casts submitted attrs (strings from a form, or Go values)
// into typed params. Keys that aren't source columns are ignored; values that
// can't be cast are reported by Validate as "is invalid". The nested
// "metadata" key becomes Metadata.
func ParseSourceParams(attrs Attrs) SourceParams {
	p := SourceParams{set: map[string]bool{}, parseErrs: map[string][]string{}}
	parseParams(sourceSpecs, &p, attrs, p.set, p.parseErrs)

	if raw, ok := attrs["metadata"]; ok && raw != nil {
		child, ok := toAttrs(raw)
		if !ok {
			addErr(p.parseErrs, "metadata", "is invalid")
		} else {
			p.Metadata = sourceMetadataFromAttrs(child, p.parseErrs)
		}
	}
	return p
}

// WithIndexFrequencyMinutes returns p with index_frequency_minutes submitted as n.
func (p SourceParams) WithIndexFrequencyMinutes(n int) SourceParams {
	p.set = copySet(p.set)
	p.set["index_frequency_minutes"] = true
	p.IndexFrequencyMinutes = &n
	return p
}

// WithCollection returns p with the collection fields submitted.
func (p SourceParams) WithCollection(typ SourceCollectionType, id, name *string) SourceParams {
	p.set = copySet(p.set)
	p.set["collection_type"] = true
	p.set["collection_id"] = true
	p.set["collection_name"] = true
	p.CollectionType = &typ
	p.CollectionID = id
	p.CollectionName = name
	return p
}

// field is the value a field will have once p is applied to existing: the
// submitted value if there is one (nil when submitted blank), else the
// existing one.
func (p *SourceParams) field(existing *Source, name string) any {
	if p.set[name] {
		return paramValue(sourceSpecs, p, name)
	}
	if name == "uuid" && p.uuid != nil {
		return *p.uuid
	}
	return structValue(existing, name)
}

// withDefaults fills custom_name from collection_name and generates a uuid
// when they are blank.
func (p SourceParams) withDefaults(existing *Source) SourceParams {
	// A non-nil field counts as submitted, so params can be built as literals.
	set := copySet(p.set)
	for _, sp := range sourceSpecs {
		if getParam(sp.field(&p)) != nil {
			set[sp.name] = true
		}
	}
	p.set = set
	if isBlankValue(p.field(existing, "custom_name"), false) {
		v, _ := castValue(fieldInfo{name: "custom_name", typ: typeOf[string]()}, p.field(existing, "collection_name"))
		p.set["custom_name"] = true
		setParam(&p.CustomName, typeOf[string](), v)
	}
	if isBlankValue(p.field(existing, "uuid"), false) {
		u := GenerateUUID()
		p.uuid = &u
	}
	return p
}

// Apply returns a copy of existing with p applied (defaults included).
func (p SourceParams) Apply(existing *Source) *Source {
	p = p.withDefaults(existing)
	cp := *existing
	applyParams(sourceSpecs, &p, p.set, reflect.ValueOf(&cp).Elem())
	if p.uuid != nil {
		cp.UUID = p.uuid
	}
	return &cp
}

// Changed reports which submitted fields differ from existing's values.
func (p SourceParams) Changed(existing *Source) map[string]bool {
	p = p.withDefaults(existing)
	changed := map[string]bool{}
	for _, sp := range sourceSpecs {
		if p.set[sp.name] && !equalValues(structValue(existing, sp.name), getParam(sp.field(&p))) {
			changed[sp.name] = true
		}
	}
	return changed
}

// Validate checks p against existing and returns field -> messages (empty when
// valid). stage is "initial" (only the fields needed to look the source up)
// or "pre_insert" (everything a stored source needs).
func (p SourceParams) Validate(existing *Source, stage string) map[string][]string {
	p = p.withDefaults(existing)
	errs := copyErrs(p.parseErrs)
	changed := p.Changed(existing)

	required := sourceRequiredInitial
	if stage == "pre_insert" {
		required = sourceRequiredPreInsert
	}
	for _, f := range required {
		if len(errs[f]) > 0 {
			continue
		}
		if isBlankValue(p.field(existing, f), true) {
			addErr(errs, f, "can't be blank")
		}
	}

	if changed["title_filter_regex"] && p.TitleFilterRegex != nil {
		if _, err := db.CompileRegex(*p.TitleFilterRegex); err != nil {
			addErr(errs, "title_filter_regex", "is invalid")
		}
	}

	if changed["min_duration_seconds"] && changed["max_duration_seconds"] &&
		p.MinDurationSeconds != nil && p.MaxDurationSeconds != nil &&
		*p.MinDurationSeconds >= *p.MaxDurationSeconds {
		addErr(errs, "max_duration_seconds", "must be greater than minumum duration")
	}

	if changed["retention_period_days"] && p.RetentionPeriodDays != nil && *p.RetentionPeriodDays < 0 {
		addErr(errs, "retention_period_days", "must be greater than or equal to 0")
	}

	if changed["output_path_template_override"] && p.OutputPathTemplateOverride != nil &&
		!matchesFormat(outputPathTemplateOverrideFormat, *p.OutputPathTemplateOverride) {
		addErr(errs, "output_path_template_override", "must end with .{{ ext }}")
	}

	if changed["original_url"] && p.OriginalURL != nil &&
		!matchesFormat(sourceYoutubeChannelOrPlaylistRegex(), *p.OriginalURL) {
		addErr(errs, "original_url", "must be a channel or playlist URL")
	}

	if p.Metadata != nil {
		merged := applySourceMetadata(existing.Metadata, p.Metadata)
		if len(errs["metadata.metadata_filepath"]) == 0 && strings.TrimSpace(merged.MetadataFilepath) == "" {
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

// SourceFieldValue is source's current value for a column, for redisplaying a
// form (nil pointers come back as untyped nil).
func SourceFieldValue(source *Source, column string) any { return structValue(source, column) }

// SourceChanges describes what an update did.
type SourceChanges struct {
	// Changed names the submitted fields whose value differs from Before.
	Changed map[string]bool
	Before  *Source
	After   *Source
}

// sourceMetadataFromAttrs builds the nested metadata row from attrs. Blank
// values are left empty (meaning "keep the existing value").
func sourceMetadataFromAttrs(attrs Attrs, errs map[string][]string) *SourceMetadata {
	m := &SourceMetadata{}
	for _, name := range SourceMetadataFilepathAttributes() {
		raw, ok := attrs[name]
		if !ok {
			continue
		}
		v, err := castValue(fieldInfo{name: name, typ: typeOf[string]()}, raw)
		if err != nil {
			addErr(errs, "metadata."+name, "is invalid")
			continue
		}
		s, _ := v.(string)
		switch name {
		case "metadata_filepath":
			m.MetadataFilepath = s
		case "fanart_filepath":
			m.FanartFilepath = strPtr(s)
		case "poster_filepath":
			m.PosterFilepath = strPtr(s)
		case "banner_filepath":
			m.BannerFilepath = strPtr(s)
		}
	}
	return m
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
