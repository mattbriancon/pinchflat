package core_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/db"
)

func TestRssFeedBuilder_Build(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	source := coretest.SourceFixture(t, ta, core.Attrs{})

	t.Run("returns XML", func(t *testing.T) {

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		if !strings.Contains(res, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Errorf("missing XML declaration")
		}
	})

	t.Run("escapes ampersands", func(t *testing.T) {

		src := coretest.SourceFixture(t, ta, core.Attrs{"custom_name": "A & B"})
		res, err := ta.RssFeedBuilderBuild(ta.Ctx, src, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		if !strings.Contains(res, `<title>A &amp; B</title>`) {
			t.Errorf("missing escaped ampersand")
		}
	})

	t.Run("applies limit", func(t *testing.T) {

		goodMedia := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{core.Opt("limit", 0)})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		if strings.Contains(res, *goodMedia.Title) {
			t.Errorf("limit 0 should exclude media")
		}
	})

	t.Run("uses custom URL base", func(t *testing.T) {

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{core.Opt("url_base", "http://example.com")})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		expectedURL := "http://example.com/sources/" + *source.UUID + "/feed.xml"
		if !strings.Contains(res, expectedURL) {
			t.Errorf("missing custom URL")
		}
	})
}

func TestRssFeedBuilder_Build_SourceXml(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	source := coretest.SourceFixture(t, ta, core.Attrs{})

	t.Run("includes source attributes", func(t *testing.T) {

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		if !strings.Contains(res, "<title>"+source.CustomName+"</title>") {
			t.Errorf("missing custom name")
		}
		if !strings.Contains(res, "<link>"+source.OriginalURL+"</link>") {
			t.Errorf("missing original URL")
		}
		if !strings.Contains(res, "<description>"+*source.Description+"</description>") {
			t.Errorf("missing description")
		}
		if !strings.Contains(res, "<itunes:author>"+source.CustomName+"</itunes:author>") {
			t.Errorf("missing itunes:author")
		}
		if !strings.Contains(res, "<podcast:guid>"+*source.UUID+"</podcast:guid>") {
			t.Errorf("missing podcast:guid")
		}
	})

	t.Run("sets build and pub dates", func(t *testing.T) {

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		lastBuildDate := formatRSSDate(source.UpdatedAt.Time)
		pubDate := formatRSSDate(source.InsertedAt.Time)

		if !strings.Contains(res, "<lastBuildDate>"+lastBuildDate+"</lastBuildDate>") {
			t.Errorf("missing lastBuildDate")
		}
		if !strings.Contains(res, "<pubDate>"+pubDate+"</pubDate>") {
			t.Errorf("missing pubDate")
		}
	})

	t.Run("includes self link", func(t *testing.T) {

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		expectedLink := `http://localhost:8945/sources/` + *source.UUID + `/feed.xml`
		if !strings.Contains(res, expectedLink) {
			t.Errorf("missing self link")
		}
	})

	t.Run("includes image when available", func(t *testing.T) {

		src := coretest.SourceWithMetadataAttachmentsFixture(t, ta, core.Attrs{})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, src, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		parts := strings.Split(res, "<image>")
		if len(parts) < 2 {
			t.Fatalf("missing image tag")
		}
		imagePart := parts[1]

		expectedImageURL := `http://localhost:8945/sources/` + *src.UUID + `/feed_image.jpg`
		if !strings.Contains(imagePart, expectedImageURL) {
			t.Errorf("missing image URL")
		}
		if !strings.Contains(imagePart, "<title>"+src.CustomName+"</title>") {
			t.Errorf("missing image title")
		}
		if !strings.Contains(imagePart, "<link>"+src.OriginalURL+"</link>") {
			t.Errorf("missing image link")
		}

		if !strings.Contains(res, `<itunes:image href="http://localhost:8945/sources/`+*src.UUID+`/feed_image.jpg"></itunes:image>`) {
			t.Errorf("missing itunes:image")
		}
	})
}

func TestRssFeedBuilder_Build_MediaXml(t *testing.T) {
	t.Parallel()
	t.Run("filters persisted media", func(t *testing.T) {

		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		goodMedia := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID})
		badMedia := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": "/tmp/existing_file.mp3"})
		pendingMedia := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": nil})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		if goodMedia.Title != nil && !strings.Contains(res, "<title>"+*goodMedia.Title+"</title>") {
			t.Errorf("missing good media")
		}
		if badMedia.Title != nil && strings.Contains(res, "<title>"+*badMedia.Title+"</title>") {
			t.Errorf("should exclude non-existent file")
		}
		if pendingMedia.Title != nil && strings.Contains(res, "<title>"+*pendingMedia.Title+"</title>") {
			t.Errorf("should exclude pending media")
		}
	})

	t.Run("includes media attributes", func(t *testing.T) {

		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID, "title": "Test Media Item", "description": "Test Description"})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("missing item tag")
		}
		itemXml := parts[1]

		if !strings.Contains(itemXml, "<guid isPermaLink=\"false\">"+*mediaItem.UUID+"</guid>") {
			t.Errorf("missing guid")
		}
		if mediaItem.Title != nil && !strings.Contains(itemXml, "<title>"+*mediaItem.Title+"</title>") {
			t.Errorf("missing title")
		}
		if !strings.Contains(itemXml, "<link>"+mediaItem.OriginalURL+"</link>") {
			t.Errorf("missing link")
		}
		if mediaItem.Description != nil && !strings.Contains(itemXml, "<description>"+*mediaItem.Description+"</description>") {
			t.Errorf("missing description")
		}
		if !strings.Contains(itemXml, "<itunes:author>"+source.CustomName+"</itunes:author>") {
			t.Errorf("missing itunes:author")
		}
		if mediaItem.Title != nil && !strings.Contains(itemXml, "<itunes:subtitle>"+*mediaItem.Title+"</itunes:subtitle>") {
			t.Errorf("missing itunes:subtitle")
		}
		if mediaItem.Description != nil && !strings.Contains(itemXml, "<itunes:summary><![CDATA["+*mediaItem.Description+"]]></itunes:summary>") {
			t.Errorf("missing itunes:summary")
		}
	})

	t.Run("sets pubDate", func(t *testing.T) {

		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		uploadedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID, "uploaded_at": db.UTCDateTime{Time: uploadedAt}})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("missing item tag")
		}
		itemXml := parts[1]

		if !strings.Contains(itemXml, "<pubDate>Wed, 01 Jan 2020 00:00:00 +0000</pubDate>") {
			t.Errorf("missing pubDate")
		}
	})

	t.Run("includes enclosure", func(t *testing.T) {

		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID, "media_size_bytes": int64(1234)})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("missing item tag")
		}
		itemXml := parts[1]

		if !strings.Contains(itemXml, "<enclosure") {
			t.Errorf("missing enclosure")
		}
		if !strings.Contains(itemXml, `http://localhost:8945/media/`+*mediaItem.UUID+`/stream.mp4`) {
			t.Errorf("missing stream URL")
		}
		if !strings.Contains(itemXml, `length="1234"`) {
			t.Errorf("missing length")
		}
		if !strings.Contains(itemXml, `type="video/mp4"`) {
			t.Errorf("missing type")
		}
	})

	t.Run("includes image tags when available", func(t *testing.T) {

		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID, "media_size_bytes": int64(1234), "title": "Test Media"})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("missing item tag")
		}
		itemXml := parts[1]

		if !strings.Contains(itemXml, `<itunes:image href="http://localhost:8945/media/`+*mediaItem.UUID+`/episode_image.jpg"></itunes:image>`) {
			t.Errorf("missing itunes:image")
		}

		if !strings.Contains(itemXml, `<podcast:images srcset="http://localhost:8945/media/`+*mediaItem.UUID+`/episode_image.jpg" />`) {
			t.Errorf("missing podcast:images")
		}
	})

	t.Run("omits image tags when none", func(t *testing.T) {

		ta := coretest.NewApp(t)
		defer ta.App.DB.Close()

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID})
		os.Remove(*mediaItem.ThumbnailFilepath)

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("missing item tag")
		}
		itemXml := parts[1]

		if strings.Contains(itemXml, "itunes:image") {
			t.Errorf("should not have itunes:image")
		}
		if strings.Contains(itemXml, "podcast:images") {
			t.Errorf("should not have podcast:images")
		}
	})
}

func formatRSSDate(t time.Time) string {
	return t.Format("Mon, 02 Jan 2006 15:04:05 -0700")
}
