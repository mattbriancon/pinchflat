package store_test

import (
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
	attrs    store.Attrs
	stage    string
	want     map[string][]string
}

var sourceValidateCases = []sourceValidateCase{
	{"valid update", validExistingSource, store.Attrs{"collection_name": "n2"}, "pre_insert", map[string][]string{}},
	{"new blank initial", store.NewSource, store.Attrs{}, "initial", map[string][]string{
		"original_url": {"can't be blank"},
	}},
	{"new blank pre_insert", store.NewSource, store.Attrs{}, "pre_insert", map[string][]string{
		"original_url":    {"can't be blank"},
		"custom_name":     {"can't be blank"},
		"collection_name": {"can't be blank"},
		"collection_id":   {"can't be blank"},
		"collection_type": {"can't be blank"},
	}},
	{"required set to nil", validExistingSource, store.Attrs{"original_url": nil, "index_frequency_minutes": nil, "fast_index": "", "media_profile_id": nil, "download_media": nil}, "initial", map[string][]string{
		"original_url":            {"can't be blank"},
		"index_frequency_minutes": {"can't be blank"},
		"fast_index":              {"can't be blank"},
		"media_profile_id":        {"can't be blank"},
		"download_media":          {"can't be blank"},
	}},
	{"collection id nil", validExistingSource, store.Attrs{"collection_id": nil}, "pre_insert", map[string][]string{
		"collection_id": {"can't be blank"},
	}},
	{"whitespace name", validExistingSource, store.Attrs{"collection_name": "  "}, "pre_insert", map[string][]string{
		"collection_name": {"can't be blank"},
	}},
	{"invalid regex", validExistingSource, store.Attrs{"title_filter_regex": "*FOO"}, "pre_insert", map[string][]string{
		"title_filter_regex": {"is invalid"},
	}},
	{"valid regex", validExistingSource, store.Attrs{"title_filter_regex": "(?i)^How to Bike$"}, "pre_insert", map[string][]string{}},
	{"min >= max", validExistingSource, store.Attrs{"min_duration_seconds": 200, "max_duration_seconds": 100}, "pre_insert", map[string][]string{
		"max_duration_seconds": {"must be greater than minumum duration"},
	}},
	{"min == max", validExistingSource, store.Attrs{"min_duration_seconds": "100", "max_duration_seconds": "100"}, "pre_insert", map[string][]string{
		"max_duration_seconds": {"must be greater than minumum duration"},
	}},
	{"min < max", validExistingSource, store.Attrs{"min_duration_seconds": 100, "max_duration_seconds": 200}, "pre_insert", map[string][]string{}},
	{"only one duration", validExistingSource, store.Attrs{"min_duration_seconds": 100, "max_duration_seconds": nil}, "pre_insert", map[string][]string{}},
	{"negative retention", validExistingSource, store.Attrs{"retention_period_days": -1}, "pre_insert", map[string][]string{
		"retention_period_days": {"must be greater than or equal to 0"},
	}},
	{"zero retention", validExistingSource, store.Attrs{"retention_period_days": 0}, "pre_insert", map[string][]string{}},
	{"bad output template", validExistingSource, store.Attrs{"output_path_template_override": "foo.mp4"}, "pre_insert", map[string][]string{
		"output_path_template_override": {"must end with .{{ ext }}"},
	}},
	{"good output template", validExistingSource, store.Attrs{"output_path_template_override": "{{ title }}.{{ ext }}"}, "pre_insert", map[string][]string{}},
	{"video url", validExistingSource, store.Attrs{"original_url": "https://www.youtube.com/watch?v=72maj9FLQZI"}, "pre_insert", map[string][]string{
		"original_url": {"must be a channel or playlist URL"},
	}},
	{"non-youtube url", validExistingSource, store.Attrs{"original_url": "https://www.example.com/watch?v=1"}, "pre_insert", map[string][]string{}},
	{"youtu.be url", validExistingSource, store.Attrs{"original_url": "https://youtu.be/72maj9FLQZI"}, "pre_insert", map[string][]string{
		"original_url": {"must be a channel or playlist URL"},
	}},
	{"shorts url", validExistingSource, store.Attrs{"original_url": "https://www.youtube.com/shorts/Dq0eH-ZhQTU"}, "pre_insert", map[string][]string{
		"original_url": {"must be a channel or playlist URL"},
	}},
	{"playlist url", validExistingSource, store.Attrs{"original_url": "https://www.youtube.com/playlist?list=PLpjK416fmKwRtq-9-O_NbZlkW0k6zu2Wn"}, "pre_insert", map[string][]string{}},
	{"cast errors", validExistingSource, store.Attrs{
		"index_frequency_minutes": "abc",
		"cookie_behaviour":        "bad",
		"fast_index":              "maybe",
		"download_cutoff_date":    "notadate",
		"collection_type":         "album",
		"media_profile_id":        "x",
	}, "pre_insert", map[string][]string{
		"index_frequency_minutes": {"is invalid"},
		"cookie_behaviour":        {"is invalid"},
		"fast_index":              {"is invalid"},
		"download_cutoff_date":    {"is invalid"},
		"collection_type":         {"is invalid"},
		"media_profile_id":        {"is invalid"},
	}},
	{"metadata missing filepath", store.NewSource, store.Attrs{"metadata": store.Attrs{"poster_filepath": "/p.jpg"}}, "initial", map[string][]string{
		"original_url":               {"can't be blank"},
		"metadata.metadata_filepath": {"can't be blank"},
	}},
	{"metadata ok", validExistingSource, store.Attrs{"metadata": map[string]string{"metadata_filepath": "/m.json.gz"}}, "pre_insert", map[string][]string{}},
	{"metadata not a map", validExistingSource, store.Attrs{"metadata": "x"}, "pre_insert", map[string][]string{
		"metadata": {"is invalid"},
	}},
}

func TestSourceParams_Validate(t *testing.T) {
	for _, tc := range sourceValidateCases {
		t.Run(tc.name, func(t *testing.T) {
			got := store.ParseSourceParams(tc.attrs).Validate(tc.existing(), tc.stage)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
