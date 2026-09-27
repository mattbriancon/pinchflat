package web

// Port of lib/pinchflat_web/controllers/sources/source_html.ex: the helper
// functions and template-property definitions.

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db"
)

// FriendlyIndexFrequencies is friendly_index_frequencies/0: a list of
// (label, value) pairs for the index frequency select.
func FriendlyIndexFrequencies() [][2]interface{} {
	return [][2]interface{}{
		{"Only once when first created", int64(-1)},
		{"30 minutes", int64(30)},
		{"1 Hour", int64(60)},
		{"3 Hours", int64(180)},
		{"6 Hours", int64(360)},
		{"12 Hours", int64(720)},
		{"Daily (recommended)", int64(1440)},
		{"Weekly", int64(10080)},
		{"Monthly", int64(43200)},
	}
}

// FriendlyIndexFrequencyOptions is FriendlyIndexFrequencies as select options.
func FriendlyIndexFrequencyOptions() []CoreSelectOption {
	out := make([]CoreSelectOption, 0, len(FriendlyIndexFrequencies()))
	for _, pair := range FriendlyIndexFrequencies() {
		out = append(out, CoreSelectOption{Label: pair[0].(string), Value: fmt.Sprint(pair[1])})
	}
	return out
}

// FriendlyCookieBehaviours is friendly_cookie_behaviours/0: a list of
// (label, value) pairs for the cookie behaviour select.
func FriendlyCookieBehaviours() [][2]interface{} {
	return [][2]interface{}{
		{"Disabled", "disabled"},
		{"When Needed", "when_needed"},
		{"All Operations", "all_operations"},
	}
}

// FriendlyCookieBehaviourOptions is FriendlyCookieBehaviours as select options.
func FriendlyCookieBehaviourOptions() []CoreSelectOption {
	out := make([]CoreSelectOption, 0, len(FriendlyCookieBehaviours()))
	for _, pair := range FriendlyCookieBehaviours() {
		out = append(out, CoreSelectOption{Label: pair[0].(string), Value: pair[1].(string)})
	}
	return out
}

// MediaProfileOptions turns media profiles into select options
// (Enum.map(@media_profiles, &{&1.name, &1.id})).
func MediaProfileOptions(mediaProfiles []*core.MediaProfile) []CoreSelectOption {
	out := make([]CoreSelectOption, 0, len(mediaProfiles))
	for _, p := range mediaProfiles {
		out = append(out, CoreSelectOption{Label: p.Name, Value: fmt.Sprint(p.ID)})
	}
	return out
}

// CutoffDatePresets is cutoff_date_presets/0: a list of (label, value) pairs
// for download cutoff date presets, computed from "now" in the configured
// timezone (Application.get_env(:pinchflat, :timezone)).
func CutoffDatePresets(ctx context.Context) [][2]string {
	now := sourcesHTMLNow(ctx)
	return [][2]string{
		{"7 days", computeDateOffset(now, 7)},
		{"14 days", computeDateOffset(now, 14)},
		{"30 days", computeDateOffset(now, 30)},
		{"60 days", computeDateOffset(now, 60)},
		{"90 days", computeDateOffset(now, 90)},
		{"180 days", computeDateOffset(now, 180)},
		{"365 days", computeDateOffset(now, 365)},
	}
}

// CutoffDatePresetOptions is CutoffDatePresets as select options.
func CutoffDatePresetOptions(ctx context.Context) []CoreSelectOption {
	presets := CutoffDatePresets(ctx)
	out := make([]CoreSelectOption, 0, len(presets))
	for _, pair := range presets {
		out = append(out, CoreSelectOption{Label: pair[0], Value: pair[1]})
	}
	return out
}

func sourcesHTMLNow(ctx context.Context) time.Time {
	tz := "UTC"
	if a := layoutsApp(ctx); a != nil && a.Config.Timezone != "" {
		tz = a.Config.Timezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc)
}

// RssFeedURL is rss_feed_url/2: the absolute RSS feed URL for a source. The
// separate concatenation (rather than a single ~p sigil) works around a
// Phoenix bug (see the Elixir source); it has no effect in the Go port but
// is kept for parity.
func RssFeedURL(ctx context.Context, source *core.Source) string {
	return URL(ctx, "/sources/%v/feed", core.Deref(source.UUID)) + ".xml"
}

// OpmlFeedURL is opml_feed_url/1: the absolute, route-token-protected OPML
// feed URL.
func OpmlFeedURL(ctx context.Context) string {
	token := ""
	if a := layoutsApp(ctx); a != nil {
		if v, err := a.SettingsGetBang(ctx, "route_token"); err == nil {
			token, _ = v.(string)
		}
	}
	return URL(ctx, "/sources/opml.xml") + "?route_token=" + url.QueryEscape(token)
}

// OutputPathTemplateOverridePlaceholders returns a JSON map of media profile output path templates.
func OutputPathTemplateOverridePlaceholders(mediaProfiles []*core.MediaProfile) string {
	m := make(map[int64]string)
	for _, p := range mediaProfiles {
		m[p.ID] = p.OutputPathTemplate
	}
	data, _ := db.EncodeJSON(m)
	return string(data)
}

// TitleFilterRegexHelp returns HTML help text for title filter regex.
func TitleFilterRegexHelp() string {
	url := "https://github.com/nalgeon/sqlean/blob/main/docs/regexp.md#supported-syntax"
	classes := "underline decoration-bodydark decoration-1 hover:decoration-white"
	return fmt.Sprintf(
		`A PCRE-compatible regex. Only media with titles that match this regex will be downloaded. <a href="%s" class="%s" target="_blank">See here</a> for syntax`,
		url, classes,
	)
}

// OutputPathTemplateOverrideHelp returns HTML help text for output path template override.
func OutputPathTemplateOverrideHelp() string {
	helpButtonClasses := "underline decoration-bodydark decoration-1 hover:decoration-white cursor-pointer"
	helpButton := fmt.Sprintf(`<span class="%s" x-on:click="$dispatch('load-template')">Click here</span>`, helpButtonClasses)
	return fmt.Sprintf(
		`Must end with .{{ ext }}. Same rules as Media Profile output path templates. %s to load your media profile's output template`,
		helpButton,
	)
}

// Helper: computeDateOffset returns a date string (YYYY-MM-DD) offset by days from now.
func computeDateOffset(t time.Time, days int) string {
	offset := t.AddDate(0, 0, -days)
	return offset.Format("2006-01-02")
}

// Friendly frequency value: convert minutes to display text.
func friendlyFrequencyValue(minutes int64) string {
	freqs := FriendlyIndexFrequencies()
	for _, pair := range freqs {
		if pair[1].(int64) == minutes {
			return pair[0].(string)
		}
	}
	return fmt.Sprintf("%d minutes", minutes)
}

// FindProfileByID finds a media profile by ID in the list.
func FindProfileByID(profiles []*core.MediaProfile, id int64) *core.MediaProfile {
	idx := slices.IndexFunc(profiles, func(p *core.MediaProfile) bool { return p.ID == id })
	if idx >= 0 {
		return profiles[idx]
	}
	return nil
}

// sourcesListItem is one row of list_items_from_map/1 (core_components.ex):
// a scalar struct field, formatted like Map.from_struct/1 would show it
// (lists joined with ", "), plus whether the value looks like an http(s)
// URL (rendered as a link).
type sourcesListItem struct {
	Key   string
	Value string
	IsURL bool
}

// sourcesListItemsFromMap is list_items_from_map(Map.from_struct(@source)):
// core_components.ex's version isn't ported (it's outside this row), so
// this is a private equivalent scoped to *core.Source. It walks the schema
// fields in declaration order (skipping `db:"-"` associations, which
// Elixir's filter drops as non-Date/DateTime structs or lists of structs).
func sourcesListItemsFromMap(source *core.Source) []sourcesListItem {
	var out []sourcesListItem
	rv := reflect.ValueOf(*source)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		out = append(out, sourcesListItem{
			Key:   strings.SplitN(tag, ",", 2)[0],
			Value: InputValue(rv.Field(i).Interface()),
			IsURL: sourcesLooksLikeURL(InputValue(rv.Field(i).Interface())),
		})
	}
	return out
}

func sourcesLooksLikeURL(v string) bool {
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")
}

// sourcesJSONLiteral is the JS template literal
// (String.raw`#{Jason.Formatter.pretty_print(Jason.encode!(@source))}`) used
// by the "Copy JSON" action. The Go port has no Jason.Encoder derivation for
// *core.Source, so it builds an ordered {field: value} map from the same
// scalar fields as sourcesListItemsFromMap.
func sourcesJSONLiteral(source *core.Source) string {
	items := sourcesListItemsFromMap(source)
	var b strings.Builder
	b.WriteString("`{\n")
	for i, item := range items {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "  %q: %q", item.Key, item.Value)
	}
	b.WriteString("\n}`")
	return b.String()
}
