package web

// Plain-Go half of media_profile_form.html.heex (attribute helpers that
// templ's DSL can't build inline).

import (
	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// mediaProfilesFormAttrs are the form tag's rest attributes: action, method
// (always "post" on the wire; method="patch" adds a hidden _method field,
// see mediaProfilesMethodOverride) and the Alpine advancedMode toggle.
func mediaProfilesFormAttrs(action string) templ.Attributes {
	return templ.Attributes{
		"action": action,
		"method": "post",
		"x-data": "{ advancedMode: !!JSON.parse(localStorage.getItem('advancedMode')) }",
		"x-init": "$watch('advancedMode', value => localStorage.setItem('advancedMode', JSON.stringify(value)))",
	}
}

// mediaProfilesField is f[:name]: Form.Field returns a value, but
// CoreInputProps wants a pointer.
func mediaProfilesField(f *Form, name string) *FormField {
	ff := f.Field(name)
	return &ff
}

// mediaProfileForm builds the form shown for m, with errs on redisplay.
func mediaProfileForm(m *store.MediaProfile, errs map[string][]string) *Form {
	// Nullable columns show as nil (not a typed nil pointer) when unset.
	nullable := func(v any, isNil bool) any {
		if isNil {
			return nil
		}
		return v
	}
	return NewForm("media_profile", map[string]any{
		"name":                    m.Name,
		"output_path_template":    m.OutputPathTemplate,
		"download_subs":           m.DownloadSubs,
		"download_auto_subs":      m.DownloadAutoSubs,
		"embed_subs":              m.EmbedSubs,
		"sub_langs":               m.SubLangs,
		"download_thumbnail":      m.DownloadThumbnail,
		"embed_thumbnail":         m.EmbedThumbnail,
		"download_source_images":  m.DownloadSourceImages,
		"download_metadata":       m.DownloadMetadata,
		"embed_metadata":          m.EmbedMetadata,
		"download_nfo":            m.DownloadNfo,
		"shorts_behaviour":        m.ShortsBehaviour,
		"livestream_behaviour":    m.LivestreamBehaviour,
		"preferred_resolution":    m.PreferredResolution,
		"sponsorblock_categories": m.SponsorblockCategories,
		"sponsorblock_behaviour":  nullable(m.SponsorblockBehaviour, m.SponsorblockBehaviour == nil),
		"redownload_delay_days":   nullable(m.RedownloadDelayDays, m.RedownloadDelayDays == nil),
		"media_container":         nullable(m.MediaContainer, m.MediaContainer == nil),
		"audio_track":             nullable(m.AudioTrack, m.AudioTrack == nil),
	}, errs)
}
