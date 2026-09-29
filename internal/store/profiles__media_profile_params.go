package store

import (
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/mattbriancon/pinchflat/internal/db"
)

// MediaProfileParams are the user-settable columns of a media profile. A nil
// field was not submitted and leaves the profile unchanged. A string field
// pointing at "" was submitted blank: required fields fail validation and
// nullable ones are cleared.
type MediaProfileParams struct {
	Name                   *string
	OutputPathTemplate     *string
	DownloadSubs           *bool
	DownloadAutoSubs       *bool
	EmbedSubs              *bool
	SubLangs               *string
	DownloadThumbnail      *bool
	EmbedThumbnail         *bool
	DownloadSourceImages   *bool
	DownloadMetadata       *bool
	EmbedMetadata          *bool
	DownloadNfo            *bool
	SponsorblockBehaviour  *MediaProfileSponsorblockBehaviour
	SponsorblockCategories *[]string
	ShortsBehaviour        *MediaProfileShortsBehaviour
	LivestreamBehaviour    *MediaProfileLivestreamBehaviour
	AudioTrack             *string
	PreferredResolution    *MediaProfilePreferredResolution
	MediaContainer         *string
	MarkedForDeletionAt    *time.Time

	// RedownloadDelayDays is the new value; ClearRedownloadDelayDays sets it
	// to NULL (the field was submitted blank).
	RedownloadDelayDays      *int
	ClearRedownloadDelayDays bool
}

// ParseMediaProfileParams reads the media_profile[...] fields of a submitted
// form. The returned map has an "is invalid" message for each field that
// could not be converted; those fields are left nil.
func ParseMediaProfileParams(form url.Values) (MediaProfileParams, map[string][]string) {
	var p MediaProfileParams
	f := newFormReader(form, "media_profile")

	f.str("name", &p.Name)
	f.str("output_path_template", &p.OutputPathTemplate)
	f.str("sub_langs", &p.SubLangs)
	f.str("audio_track", &p.AudioTrack)
	f.str("media_container", &p.MediaContainer)
	f.boolOrFalse("download_subs", &p.DownloadSubs)
	f.boolOrFalse("download_auto_subs", &p.DownloadAutoSubs)
	f.boolOrFalse("embed_subs", &p.EmbedSubs)
	f.boolOrFalse("download_thumbnail", &p.DownloadThumbnail)
	f.boolOrFalse("embed_thumbnail", &p.EmbedThumbnail)
	f.boolOrFalse("download_source_images", &p.DownloadSourceImages)
	f.boolOrFalse("download_metadata", &p.DownloadMetadata)
	f.boolOrFalse("embed_metadata", &p.EmbedMetadata)
	f.boolOrFalse("download_nfo", &p.DownloadNfo)

	enumeration(f, "sponsorblock_behaviour", &p.SponsorblockBehaviour,
		MediaProfileSponsorblockBehaviourDisabled, MediaProfileSponsorblockBehaviourMark, MediaProfileSponsorblockBehaviourRemove)
	enumeration(f, "shorts_behaviour", &p.ShortsBehaviour,
		MediaProfileShortsBehaviourInclude, MediaProfileShortsBehaviourExclude, MediaProfileShortsBehaviourOnly)
	enumeration(f, "livestream_behaviour", &p.LivestreamBehaviour,
		MediaProfileLivestreamBehaviourInclude, MediaProfileLivestreamBehaviourExclude, MediaProfileLivestreamBehaviourOnly)
	enumeration(f, "preferred_resolution", &p.PreferredResolution,
		MediaProfilePreferredResolution4320p, MediaProfilePreferredResolution2160p, MediaProfilePreferredResolution1440p,
		MediaProfilePreferredResolution1080p, MediaProfilePreferredResolution720p, MediaProfilePreferredResolution480p,
		MediaProfilePreferredResolution360p, MediaProfilePreferredResolutionAudio)

	switch n, st := f.integer("redownload_delay_days"); st {
	case fieldSet:
		p.RedownloadDelayDays = &n
	case fieldBlank:
		p.ClearRedownloadDelayDays = true
	}
	if cats := form["media_profile[sponsorblock_categories][]"]; len(cats) > 0 {
		p.SponsorblockCategories = &cats
	} else if _, ok := f.get("sponsorblock_categories"); ok {
		f.invalid("sponsorblock_categories")
	}
	return p, f.errs
}

// Validate returns the validation errors of applying p to existing (field
// name -> messages; empty when valid). Only changed fields are checked for
// format and range, like the changeset it replaces.
func (p MediaProfileParams) Validate(existing *MediaProfile) map[string][]string {
	errs := map[string][]string{}
	add := func(field, msg string) { errs[field] = append(errs[field], msg) }
	next, _ := p.apply(existing)

	if strings.TrimSpace(next.Name) == "" {
		add("name", "can't be blank")
	}
	if strings.TrimSpace(next.OutputPathTemplate) == "" {
		add("output_path_template", "can't be blank")
	} else if p.OutputPathTemplate != nil && next.OutputPathTemplate != existing.OutputPathTemplate {
		re, err := db.CompileRegex(mediaProfileExtRegex())
		if err != nil {
			panic("media profile ext regex: " + err.Error())
		}
		if ok, _ := re.MatchString(next.OutputPathTemplate); !ok {
			add("output_path_template", "must end with .{{ ext }}")
		}
	}
	if d := p.RedownloadDelayDays; d != nil && *d < 0 && !reflect.DeepEqual(existing.RedownloadDelayDays, d) {
		add("redownload_delay_days", "must be greater than or equal to 0")
	}
	return errs
}

// Apply returns a copy of m with p's fields applied.
func (p MediaProfileParams) Apply(m *MediaProfile) *MediaProfile {
	next, _ := p.apply(m)
	return next
}

// apply also returns the names of the columns whose value changed.
func (p MediaProfileParams) apply(m *MediaProfile) (*MediaProfile, []string) {
	next := *m
	var cols []string
	set := func(col string, changed bool) {
		if changed {
			cols = append(cols, col)
		}
	}

	if p.Name != nil {
		set("name", *p.Name != next.Name)
		next.Name = *p.Name
	}
	if p.OutputPathTemplate != nil {
		set("output_path_template", *p.OutputPathTemplate != next.OutputPathTemplate)
		next.OutputPathTemplate = *p.OutputPathTemplate
	}
	if p.SubLangs != nil {
		set("sub_langs", *p.SubLangs != next.SubLangs)
		next.SubLangs = *p.SubLangs
	}
	for _, b := range []struct {
		col string
		src *bool
		dst *bool
	}{
		{"download_subs", p.DownloadSubs, &next.DownloadSubs},
		{"download_auto_subs", p.DownloadAutoSubs, &next.DownloadAutoSubs},
		{"embed_subs", p.EmbedSubs, &next.EmbedSubs},
		{"download_thumbnail", p.DownloadThumbnail, &next.DownloadThumbnail},
		{"embed_thumbnail", p.EmbedThumbnail, &next.EmbedThumbnail},
		{"download_source_images", p.DownloadSourceImages, &next.DownloadSourceImages},
		{"download_metadata", p.DownloadMetadata, &next.DownloadMetadata},
		{"embed_metadata", p.EmbedMetadata, &next.EmbedMetadata},
		{"download_nfo", p.DownloadNfo, &next.DownloadNfo},
	} {
		if b.src != nil {
			set(b.col, *b.src != *b.dst)
			*b.dst = *b.src
		}
	}
	if p.ShortsBehaviour != nil {
		set("shorts_behaviour", *p.ShortsBehaviour != next.ShortsBehaviour)
		next.ShortsBehaviour = *p.ShortsBehaviour
	}
	if p.LivestreamBehaviour != nil {
		set("livestream_behaviour", *p.LivestreamBehaviour != next.LivestreamBehaviour)
		next.LivestreamBehaviour = *p.LivestreamBehaviour
	}
	if p.PreferredResolution != nil {
		set("preferred_resolution", *p.PreferredResolution != next.PreferredResolution)
		next.PreferredResolution = *p.PreferredResolution
	}
	if p.SponsorblockBehaviour != nil {
		var v *MediaProfileSponsorblockBehaviour
		if *p.SponsorblockBehaviour != "" {
			v = p.SponsorblockBehaviour
		}
		set("sponsorblock_behaviour", !reflect.DeepEqual(next.SponsorblockBehaviour, v))
		next.SponsorblockBehaviour = v
	}
	if p.SponsorblockCategories != nil {
		v := db.NewJSON(*p.SponsorblockCategories)
		set("sponsorblock_categories", !reflect.DeepEqual(next.SponsorblockCategories, v))
		next.SponsorblockCategories = v
	}
	for _, s := range []struct {
		col string
		src *string
		dst **string
	}{
		{"audio_track", p.AudioTrack, &next.AudioTrack},
		{"media_container", p.MediaContainer, &next.MediaContainer},
	} {
		if s.src != nil {
			var v *string
			if *s.src != "" {
				v = s.src
			}
			set(s.col, !reflect.DeepEqual(*s.dst, v))
			*s.dst = v
		}
	}
	if p.RedownloadDelayDays != nil || p.ClearRedownloadDelayDays {
		set("redownload_delay_days", !reflect.DeepEqual(next.RedownloadDelayDays, p.RedownloadDelayDays))
		next.RedownloadDelayDays = p.RedownloadDelayDays
	}
	if p.MarkedForDeletionAt != nil {
		v := &db.UTCDateTime{Time: p.MarkedForDeletionAt.UTC().Truncate(time.Second)}
		set("marked_for_deletion_at", next.MarkedForDeletionAt == nil || !next.MarkedForDeletionAt.Equal(v.Time))
		next.MarkedForDeletionAt = v
	}
	return &next, cols
}
