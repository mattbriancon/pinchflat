package store_test

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestMediaItemParams_Validate(t *testing.T) {
	existing := func(title string) *store.MediaItem {
		m := store.NewMediaItem()
		m.MediaID, m.OriginalURL, m.Title, m.SourceID = "id", "http://x", store.Ptr(title), 1
		return m
	}
	blank := "can't be blank"

	tests := []struct {
		name     string
		existing *store.MediaItem
		p        store.MediaItemParams
		want     map[string][]string
	}{
		{"new item with nothing", nil, store.MediaItemParams{}, map[string][]string{
			"title": {blank}, "original_url": {blank}, "media_id": {blank},
		}},
		{"existing item unchanged", existing("a title"), store.MediaItemParams{}, map[string][]string{}},
		{"complete new item", nil, store.MediaItemParams{
			Title: store.Ptr("t"), MediaID: store.Ptr("m"), OriginalURL: store.Ptr("u"),
			SourceID: store.Ptr(int64(1)), UploadedAt: store.Ptr(time.Now()),
		}, map[string][]string{}},
		{"blank title", existing("a title"), store.MediaItemParams{Title: store.Ptr("")}, map[string][]string{"title": {blank}}},
		{"whitespace title", existing("a title"), store.MediaItemParams{Title: store.Ptr("  \t")}, map[string][]string{"title": {blank}}},
		{"cleared title", existing("a title"), store.MediaItemParams{Clear: store.ClearTitle}, map[string][]string{"title": {blank}}},
		{"youtube restriction title", existing("a title"), store.MediaItemParams{Title: store.Ptr("youtube video #123")}, map[string][]string{"title": {"has invalid format"}}},
		{"restriction is case sensitive", existing("a title"), store.MediaItemParams{Title: store.Ptr("Youtube video #123")}, map[string][]string{}},
		{"unchanged restricted title isn't rechecked", existing("youtube video #1"), store.MediaItemParams{Title: store.Ptr("youtube video #1")}, map[string][]string{}},
		{"blank url", existing("t"), store.MediaItemParams{OriginalURL: store.Ptr("")}, map[string][]string{"original_url": {blank}}},
		{"blank media_id", existing("t"), store.MediaItemParams{MediaID: store.Ptr(" ")}, map[string][]string{"media_id": {blank}}},
		{"everything blank", existing("t"), store.MediaItemParams{
			MediaID: store.Ptr(""), OriginalURL: store.Ptr(""), Title: store.Ptr(""),
		}, map[string][]string{"title": {blank}, "original_url": {blank}, "media_id": {blank}}},
		{"cleared uploaded_at", existing("t"), store.MediaItemParams{Clear: store.ClearUploadedAt}, map[string][]string{"uploaded_at": {blank}}},
		{"cleared short_form_content", existing("t"), store.MediaItemParams{Clear: store.ClearShortFormContent}, map[string][]string{"short_form_content": {blank}}},
		{"blank metadata path", existing("t"), store.MediaItemParams{
			Metadata: &store.MediaMetadataParams{MetadataFilepath: store.Ptr(""), ThumbnailFilepath: store.Ptr("x")},
		}, map[string][]string{"metadata.metadata_filepath": {blank}}},
		{"empty new metadata", existing("t"), store.MediaItemParams{Metadata: &store.MediaMetadataParams{}}, map[string][]string{
			"metadata.metadata_filepath": {blank}, "metadata.thumbnail_filepath": {blank},
		}},
		{"complete metadata", existing("t"), store.MediaItemParams{
			Metadata: &store.MediaMetadataParams{MetadataFilepath: store.Ptr("a"), ThumbnailFilepath: store.Ptr("b")},
		}, map[string][]string{}},
		{"metadata keeps existing values", func() *store.MediaItem {
			m := existing("t")
			m.Metadata = &store.MediaMetadata{ID: 1, MetadataFilepath: "a", ThumbnailFilepath: "b"}
			return m
		}(), store.MediaItemParams{Metadata: &store.MediaMetadataParams{ThumbnailFilepath: store.Ptr("")}}, map[string][]string{
			"metadata.thumbnail_filepath": {blank},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Validate(tt.existing); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMediaItemParams(t *testing.T) {
	tests := []struct {
		name       string
		values     url.Values
		want       store.MediaItemParams
		wantErrors map[string][]string
	}{
		{"nothing submitted", url.Values{}, store.MediaItemParams{}, map[string][]string{}},
		{"strings", url.Values{"media_item[title]": {"a", "New Title"}, "media_item[description]": {""}},
			store.MediaItemParams{Title: store.Ptr("New Title"), Description: store.Ptr("")}, map[string][]string{}},
		{"blank required booleans are cleared", url.Values{"media_item[livestream]": {""}, "media_item[short_form_content]": {" "}},
			store.MediaItemParams{Clear: store.ClearLivestream | store.ClearShortFormContent}, map[string][]string{}},
		{"toggles on and off", url.Values{"media_item[prevent_download]": {"false", "on"}, "media_item[prevent_culling]": {"false"}},
			store.MediaItemParams{PreventDownload: store.Ptr(true), PreventCulling: store.Ptr(false)}, map[string][]string{}},
		{"blank prevent_download is false", url.Values{"media_item[prevent_download]": {""}},
			store.MediaItemParams{PreventDownload: store.Ptr(false)}, map[string][]string{}},
		{"blank prevent_culling is NULL", url.Values{"media_item[prevent_culling]": {" "}},
			store.MediaItemParams{Clear: store.ClearPreventCulling}, map[string][]string{}},
		{"unparseable toggle", url.Values{"media_item[prevent_download]": {"maybe"}},
			store.MediaItemParams{}, map[string][]string{"prevent_download": {"is invalid"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := store.ParseMediaItemParams(tt.values)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("params = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(errs, tt.wantErrors) {
				t.Errorf("errors = %v, want %v", errs, tt.wantErrors)
			}
		})
	}
}
