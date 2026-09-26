package web

// Port of lib/pinchflat_web/controllers/sources/source_html.ex: the helper
// functions and template-property definitions.

import (
	"fmt"
	"slices"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db"
)

// FriendlyIndexFrequencies returns a list of (label, value) pairs for index frequency select.
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

// FriendlyCookieBehaviours returns a list of (label, value) pairs for cookie behavior select.
func FriendlyCookieBehaviours() [][2]interface{} {
	return [][2]interface{}{
		{"Disabled", "disabled"},
		{"When Needed", "when_needed"},
		{"All Operations", "all_operations"},
	}
}

// CutoffDatePresets returns a list of (label, value) pairs for download cutoff date presets.
func CutoffDatePresets() [][2]string {
	now := time.Now().UTC()
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

// RssFeedURL returns the RSS feed URL for a source - called from templ with context.Context
// Note: In templ context is always context.Context, so we can't easily type hint it here
// These functions are called from templates where ctx is available
func RssFeedURL(source *core.Source) string {
	return fmt.Sprintf("/sources/%s/feed.xml", core.Deref(source.UUID))
}

// OpmlFeedURL returns the OPML feed URL - template helper
func OpmlFeedURL() string {
	return "/sources/opml.xml"
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
