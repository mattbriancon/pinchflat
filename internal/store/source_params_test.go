package store_test

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
)

func validExistingSource() *store.Source {
	src := store.NewSource()
	uuid := "9f5c9a44-6a55-4d0e-9a0a-000000000000"
	src.ID = 1
	src.UUID = &uuid
	src.CollectionName = "name"
	src.CollectionID = "cid"
	src.CollectionType = store.SourceCollectionTypeChannel
	src.CustomName = "custom"
	src.OriginalURL = "https://www.youtube.com/@x"
	src.MediaProfileID = 1
	return src
}

type sourceValidateCase struct {
	name     string
	existing func() *store.Source
	p        store.SourceParams
	stage    string
	want     map[string][]string
}

func urlParams(u string) store.SourceParams { return store.SourceParams{OriginalURL: store.Ptr(u)} }

var sourceValidateCases = []sourceValidateCase{
	{"valid update", validExistingSource, store.SourceParams{CollectionName: store.Ptr("n2")}, "pre_insert", map[string][]string{}},
	{"new blank initial", store.NewSource, store.SourceParams{}, "initial", map[string][]string{
		"original_url": {"can't be blank"},
	}},
	{"new blank pre_insert", store.NewSource, store.SourceParams{}, "pre_insert", map[string][]string{
		"original_url":    {"can't be blank"},
		"custom_name":     {"can't be blank"},
		"collection_name": {"can't be blank"},
		"collection_id":   {"can't be blank"},
		"collection_type": {"can't be blank"},
	}},
	{"required cleared", validExistingSource, store.SourceParams{
		OriginalURL: store.Ptr(""),
		Clear:       store.ClearIndexFrequencyMinutes | store.ClearFastIndex | store.ClearMediaProfileID | store.ClearDownloadMedia,
	}, "initial", map[string][]string{
		"original_url":            {"can't be blank"},
		"index_frequency_minutes": {"can't be blank"},
		"fast_index":              {"can't be blank"},
		"media_profile_id":        {"can't be blank"},
		"download_media":          {"can't be blank"},
	}},
	{"collection id blank", validExistingSource, store.SourceParams{CollectionID: store.Ptr("")}, "pre_insert", map[string][]string{
		"collection_id": {"can't be blank"},
	}},
	{"whitespace name", validExistingSource, store.SourceParams{CollectionName: store.Ptr("  ")}, "pre_insert", map[string][]string{
		"collection_name": {"can't be blank"},
	}},
	{"invalid regex", validExistingSource, store.SourceParams{TitleFilterRegex: store.Ptr("*FOO")}, "pre_insert", map[string][]string{
		"title_filter_regex": {"is invalid"},
	}},
	{"valid regex", validExistingSource, store.SourceParams{TitleFilterRegex: store.Ptr("(?i)^How to Bike$")}, "pre_insert", map[string][]string{}},
	{"min >= max", validExistingSource, store.SourceParams{MinDurationSeconds: store.Ptr(200), MaxDurationSeconds: store.Ptr(100)}, "pre_insert", map[string][]string{
		"max_duration_seconds": {"must be greater than minumum duration"},
	}},
	{"min == max", validExistingSource, store.SourceParams{MinDurationSeconds: store.Ptr(100), MaxDurationSeconds: store.Ptr(100)}, "pre_insert", map[string][]string{
		"max_duration_seconds": {"must be greater than minumum duration"},
	}},
	{"min < max", validExistingSource, store.SourceParams{MinDurationSeconds: store.Ptr(100), MaxDurationSeconds: store.Ptr(200)}, "pre_insert", map[string][]string{}},
	{"only one duration", validExistingSource, store.SourceParams{MinDurationSeconds: store.Ptr(100), Clear: store.ClearMaxDurationSeconds}, "pre_insert", map[string][]string{}},
	{"negative retention", validExistingSource, store.SourceParams{RetentionPeriodDays: store.Ptr(-1)}, "pre_insert", map[string][]string{
		"retention_period_days": {"must be greater than or equal to 0"},
	}},
	{"zero retention", validExistingSource, store.SourceParams{RetentionPeriodDays: store.Ptr(0)}, "pre_insert", map[string][]string{}},
	{"bad output template", validExistingSource, store.SourceParams{OutputPathTemplateOverride: store.Ptr("foo.mp4")}, "pre_insert", map[string][]string{
		"output_path_template_override": {"must end with .{{ ext }}"},
	}},
	{"good output template", validExistingSource, store.SourceParams{OutputPathTemplateOverride: store.Ptr("{{ title }}.{{ ext }}")}, "pre_insert", map[string][]string{}},
	{"video url", validExistingSource, urlParams("https://www.youtube.com/watch?v=72maj9FLQZI"), "pre_insert", map[string][]string{
		"original_url": {"must be a channel or playlist URL"},
	}},
	{"non-youtube url", validExistingSource, urlParams("https://www.example.com/watch?v=1"), "pre_insert", map[string][]string{}},
	{"youtu.be url", validExistingSource, urlParams("https://youtu.be/72maj9FLQZI"), "pre_insert", map[string][]string{
		"original_url": {"must be a channel or playlist URL"},
	}},
	{"shorts url", validExistingSource, urlParams("https://www.youtube.com/shorts/Dq0eH-ZhQTU"), "pre_insert", map[string][]string{
		"original_url": {"must be a channel or playlist URL"},
	}},
	{"playlist url", validExistingSource, urlParams("https://www.youtube.com/playlist?list=PLpjK416fmKwRtq-9-O_NbZlkW0k6zu2Wn"), "pre_insert", map[string][]string{}},
	{"metadata missing filepath", store.NewSource, store.SourceParams{Metadata: &store.SourceMetadata{PosterFilepath: store.Ptr("/p.jpg")}}, "initial", map[string][]string{
		"original_url":               {"can't be blank"},
		"metadata.metadata_filepath": {"can't be blank"},
	}},
	{"metadata ok", validExistingSource, store.SourceParams{Metadata: &store.SourceMetadata{MetadataFilepath: "/m.json.gz"}}, "pre_insert", map[string][]string{}},
}

func TestSourceParams_Validate(t *testing.T) {
	for _, tc := range sourceValidateCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.p.Validate(tc.existing(), tc.stage)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseSourceParams(t *testing.T) {
	form := url.Values{
		"source[index_frequency_minutes]": {"abc"},
		"source[cookie_behaviour]":        {"bad"},
		"source[fast_index]":              {"maybe"},
		"source[download_cutoff_date]":    {"notadate"},
		"source[collection_type]":         {"album"},
		"source[media_profile_id]":        {"x"},
	}
	got := store.ParseSourceParams(form).Validate(validExistingSource(), "pre_insert")
	want := map[string][]string{
		"index_frequency_minutes": {"is invalid"},
		"cookie_behaviour":        {"is invalid"},
		"fast_index":              {"is invalid"},
		"download_cutoff_date":    {"is invalid"},
		"collection_type":         {"is invalid"},
		"media_profile_id":        {"is invalid"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	p := store.ParseSourceParams(url.Values{
		"source[original_url]":            {"https://www.youtube.com/@x"},
		"source[enabled]":                 {"on"},
		"source[index_frequency_minutes]": {"30"},
		"source[min_duration_seconds]":    {""},
		"source[cookie_behaviour]":        {"when_needed"},
	})
	if *p.OriginalURL != "https://www.youtube.com/@x" || !*p.Enabled || *p.IndexFrequencyMinutes != 30 ||
		p.Clear != store.ClearMinDurationSeconds || *p.CookieBehaviour != store.SourceCookieBehaviourWhenNeeded {
		t.Errorf("unexpected params: %+v", p)
	}
}
