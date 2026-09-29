package storetest

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/store"
)

const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// randBase64 generates a pseudo-random base64 string of the given length.
func randBase64(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = base64Chars[rand.Intn(len(base64Chars))]
	}
	return string(b)
}

// randInt generates a pseudo-random integer up to max (exclusive).
func randInt(max int) int {
	return rand.Intn(max)
}

// SourceFixture creates a Source with sensible defaults for whatever p
// leaves unset.
func SourceFixture(t testing.TB, ts *TestStore, p store.SourceParams) *store.Source {
	t.Helper()
	if p.Enabled == nil {
		p.Enabled = store.Ptr(true)
	}
	if p.CollectionName == nil {
		p.CollectionName = store.Ptr("Source #" + strconv.Itoa(randInt(1000000)+1))
	}
	if p.CollectionID == nil {
		p.CollectionID = store.Ptr(randBase64(12))
	}
	if p.CollectionType == nil {
		p.CollectionType = store.Ptr(store.SourceCollectionTypeChannel)
	}
	if p.CustomName == nil {
		p.CustomName = store.Ptr("Cool and good internal name!")
	}
	if p.Description == nil {
		p.Description = store.Ptr("This is a description")
	}
	if p.OriginalURL == nil {
		p.OriginalURL = store.Ptr("https://www.youtube.com/@" + randBase64(12))
	}
	if p.MediaProfileID == nil && p.Clear&store.ClearMediaProfileID == 0 {
		p.MediaProfileID = store.Ptr(MediaProfileFixture(t, ts, store.MediaProfileParams{}).ID)
	}
	if p.IndexFrequencyMinutes == nil && p.Clear&store.ClearIndexFrequencyMinutes == 0 {
		p.IndexFrequencyMinutes = store.Ptr(60)
	}

	// Like Elixir: a pre_insert insert, not
	// Sources.create_source (which would kick off indexing jobs).
	source, errs, err := ts.CreateSource(ts.Ctx, p)
	if err != nil || len(errs) > 0 {
		t.Fatalf("SourceFixture: %v %v", err, errs)
	}
	return source
}

// SourceAttributesReturnFixture returns the JSON string matching Elixir fixture.
func SourceAttributesReturnFixture() string {
	sourceAttributes := []map[string]any{
		{
			"id":           "video1",
			"title":        "Video 1",
			"original_url": "https://example.com/video1",
			"live_status":  "not_live",
			"description":  "desc1",
			"aspect_ratio": 1.67,
			"duration":     12.34,
			"upload_date":  "20210101",
		},
		{
			"id":           "video2",
			"title":        "Video 2",
			"original_url": "https://example.com/video2",
			"live_status":  "is_live",
			"description":  "desc2",
			"aspect_ratio": 1.67,
			"duration":     345.67,
			"upload_date":  "20220202",
		},
		{
			"id":           "video3",
			"title":        "Video 3",
			"original_url": "https://example.com/video3",
			"live_status":  "not_live",
			"description":  "desc3",
			"aspect_ratio": 1.0,
			"duration":     678.90,
			"upload_date":  "20230303",
		},
	}

	var result string
	for i, attr := range sourceAttributes {
		if i > 0 {
			result += "\n"
		}
		jsonBytes, _ := db.EncodeJSON(attr)
		result += jsonBytes
	}
	return result
}

// SourceDetailsReturnFixture returns the JSON string for source details.
func SourceDetailsReturnFixture(attrs map[string]any) string {
	channelID := randBase64(12)

	defaults := map[string]any{
		"channel_id":     channelID,
		"channel":        "Channel Name",
		"playlist_id":    channelID,
		"playlist_title": "Channel Name",
	}

	// Merge attrs
	for k, v := range attrs {
		defaults[k] = v
	}

	jsonStr, _ := db.EncodeJSON(defaults)
	return jsonStr
}
