package app_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestRssFeedBuilder_Build(t *testing.T) {
	ta := apptest.NewApp(t)

	source := apptest.SourceFixture(t, ta, store.SourceParams{})

	t.Run("returns an XML document", func(t *testing.T) {
		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		if !strings.Contains(res, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Errorf("expected XML declaration, got: %v", res)
		}
	})

	t.Run("escapes illegal characters", func(t *testing.T) {
		source := apptest.SourceFixture(t, ta, store.SourceParams{CustomName: store.Ptr("A & B")})
		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		if !strings.Contains(res, `<title>A &amp; B</title>`) {
			t.Errorf("expected escaped ampersand in title, got: %v", res)
		}
	})

	t.Run("can optionally apply a limit to media items", func(t *testing.T) {
		goodMedia := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 0, "")
		must(t, err)

		if strings.Contains(res, *goodMedia.Title) {
			t.Errorf("media item should not be in output with limit 0")
		}
	})

	t.Run("can optionally specify a URL base", func(t *testing.T) {
		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "http://example.com")
		must(t, err)

		expectedURL := "http://example.com/sources/" + *source.UUID + "/feed.xml"
		if !strings.Contains(res, expectedURL) {
			t.Errorf("expected %q in output", expectedURL)
		}
	})
}

func TestRssFeedBuilder_Build_SourceXml(t *testing.T) {
	ta := apptest.NewApp(t)

	source := apptest.SourceFixture(t, ta, store.SourceParams{})

	t.Run("returns XML for static source attributes", func(t *testing.T) {
		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		if !strings.Contains(res, "<title>"+source.CustomName+"</title>") {
			t.Errorf("expected custom name in title")
		}
		if !strings.Contains(res, "<link>"+source.OriginalURL+"</link>") {
			t.Errorf("expected original URL in link")
		}
		if !strings.Contains(res, "<description>"+*source.Description+"</description>") {
			t.Errorf("expected description")
		}
		if !strings.Contains(res, "<itunes:author>"+source.CustomName+"</itunes:author>") {
			t.Errorf("expected itunes:author")
		}
		if !strings.Contains(res, "<podcast:guid>"+*source.UUID+"</podcast:guid>") {
			t.Errorf("expected podcast:guid")
		}
	})

	t.Run("returns the lastBuildDate and pubDate based off the source's timestamps", func(t *testing.T) {
		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		lastBuildDate := formatRSSDate(source.UpdatedAt.Time)
		pubDate := formatRSSDate(source.InsertedAt.Time)

		if !strings.Contains(res, "<lastBuildDate>"+lastBuildDate+"</lastBuildDate>") {
			t.Errorf("expected lastBuildDate, got: %v", res)
		}
		if !strings.Contains(res, "<pubDate>"+pubDate+"</pubDate>") {
			t.Errorf("expected pubDate, got: %v", res)
		}
	})

	t.Run("returns a self-link", func(t *testing.T) {
		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		expectedLink := `http://localhost:8945/sources/` + *source.UUID + `/feed.xml`
		if !strings.Contains(res, expectedLink) {
			t.Errorf("expected self link %q", expectedLink)
		}
	})

	t.Run("returns a link to the feed image", func(t *testing.T) {
		source := apptest.SourceWithMetadataAttachmentsFixture(t, ta, store.SourceParams{})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		parts := strings.Split(res, "<image>")
		if len(parts) < 2 {
			t.Fatalf("expected image tag")
		}
		imagePart := parts[1]

		expectedImageURL := `http://localhost:8945/sources/` + *source.UUID + `/feed_image.jpg`
		if !strings.Contains(imagePart, expectedImageURL) {
			t.Errorf("expected image URL %q in %q", expectedImageURL, imagePart)
		}
		if !strings.Contains(imagePart, "<title>"+source.CustomName+"</title>") {
			t.Errorf("expected image title")
		}
		if !strings.Contains(imagePart, "<link>"+source.OriginalURL+"</link>") {
			t.Errorf("expected image link")
		}

		if !strings.Contains(res, `<itunes:image href="http://localhost:8945/sources/`+*source.UUID+`/feed_image.jpg"></itunes:image>`) {
			t.Errorf("expected itunes:image")
		}
	})
}

func TestRssFeedBuilder_Build_MediaXml(t *testing.T) {
	t.Run("only includes media persisted to disk", func(t *testing.T) {
		ta := apptest.NewApp(t)

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		goodMedia := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
		badMedia := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), MediaFilepath: store.Ptr("/tmp/existing_file.mp3")})
		pendingMedia := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		if goodMedia.Title != nil && !strings.Contains(res, "<title>"+*goodMedia.Title+"</title>") {
			t.Errorf("expected good media title in output")
		}
		if badMedia.Title != nil && strings.Contains(res, "<title>"+*badMedia.Title+"</title>") {
			t.Errorf("bad media should not be in output")
		}
		if pendingMedia.Title != nil && strings.Contains(res, "<title>"+*pendingMedia.Title+"</title>") {
			t.Errorf("pending media should not be in output")
		}
	})

	t.Run("returns XML for static media attributes", func(t *testing.T) {
		ta := apptest.NewApp(t)

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Title: store.Ptr("Test Media Item"), Description: store.Ptr("Test Description")})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("expected item tag")
		}
		itemXml := parts[1]

		if !strings.Contains(itemXml, "<guid isPermaLink=\"false\">"+*mediaItem.UUID+"</guid>") {
			t.Errorf("expected guid")
		}
		if mediaItem.Title != nil && !strings.Contains(itemXml, "<title>"+*mediaItem.Title+"</title>") {
			t.Errorf("expected title")
		}
		if !strings.Contains(itemXml, "<link>"+mediaItem.OriginalURL+"</link>") {
			t.Errorf("expected link")
		}
		if mediaItem.Description != nil && !strings.Contains(itemXml, "<description>"+*mediaItem.Description+"</description>") {
			t.Errorf("expected description")
		}
		if !strings.Contains(itemXml, "<itunes:author>"+source.CustomName+"</itunes:author>") {
			t.Errorf("expected itunes:author")
		}
		if mediaItem.Title != nil && !strings.Contains(itemXml, "<itunes:subtitle>"+*mediaItem.Title+"</itunes:subtitle>") {
			t.Errorf("expected itunes:subtitle")
		}
		if mediaItem.Description != nil && !strings.Contains(itemXml, "<itunes:summary><![CDATA["+*mediaItem.Description+"]]></itunes:summary>") {
			t.Errorf("expected itunes:summary with CDATA")
		}
	})

	t.Run("returns pubDate based off the media's uploaded_at", func(t *testing.T) {
		ta := apptest.NewApp(t)

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		uploadedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), UploadedAt: store.Ptr(uploadedAt)})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("expected item tag")
		}
		itemXml := parts[1]

		if !strings.Contains(itemXml, "<pubDate>Wed, 01 Jan 2020 00:00:00 +0000</pubDate>") {
			t.Errorf("expected pubDate")
		}
	})

	t.Run("returns an enclosure tag with the media's stream URL", func(t *testing.T) {
		ta := apptest.NewApp(t)

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), MediaSizeBytes: store.Ptr(int64(1234))})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("expected item tag")
		}
		itemXml := parts[1]

		if !strings.Contains(itemXml, "<enclosure") {
			t.Errorf("expected enclosure tag")
		}
		if !strings.Contains(itemXml, `http://localhost:8945/media/`+*mediaItem.UUID+`/stream.mp4`) {
			t.Errorf("expected media stream URL")
		}
		if !strings.Contains(itemXml, `length="1234"`) {
			t.Errorf("expected length attribute")
		}
		if !strings.Contains(itemXml, `type="video/mp4"`) {
			t.Errorf("expected type attribute")
		}
	})

	t.Run("returns image tags if the media has a thumbnail", func(t *testing.T) {
		ta := apptest.NewApp(t)

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), MediaSizeBytes: store.Ptr(int64(1234)), Title: store.Ptr("Test Media")})

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("expected item tag")
		}
		itemXml := parts[1]

		if !strings.Contains(itemXml, `<itunes:image href="http://localhost:8945/media/`+*mediaItem.UUID+`/episode_image.jpg"></itunes:image>`) {
			t.Errorf("expected itunes:image")
		}

		if !strings.Contains(itemXml, `<podcast:images srcset="http://localhost:8945/media/`+*mediaItem.UUID+`/episode_image.jpg" />`) {
			t.Errorf("expected podcast:images")
		}
	})

	t.Run("does not return image tags if the media does not have a thumbnail", func(t *testing.T) {
		ta := apptest.NewApp(t)

		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
		os.Remove(*mediaItem.ThumbnailFilepath)

		res, err := ta.RssFeedBuilderBuild(ta.Ctx, source, 2000, "")
		must(t, err)

		parts := strings.Split(res, "<item>")
		if len(parts) < 2 {
			t.Fatalf("expected item tag")
		}
		itemXml := parts[1]

		if strings.Contains(itemXml, "itunes:image") {
			t.Errorf("should not have itunes:image tag")
		}
		if strings.Contains(itemXml, "podcast:images") {
			t.Errorf("should not have podcast:images tag")
		}
	})
}

func formatRSSDate(t time.Time) string {
	return t.Format("Mon, 02 Jan 2006 15:04:05 -0700")
}
