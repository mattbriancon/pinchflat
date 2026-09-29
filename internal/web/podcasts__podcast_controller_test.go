package web_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestPodcastController_OpmlFeed(t *testing.T) {
	t.Run("renders the XML document", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.SourceParams{})
		routeToken, err := c.App.GetSetting(c.Ctx, "route_token")
		if err != nil {
			t.Fatalf("get route_token: %v", err)
		}

		res := c.Get("/sources/opml.xml", map[string]string{"route_token": routeToken.(string)})

		if res.Status != 200 {
			t.Fatalf("expected status 200, got %d", res.Status)
		}
		if !strings.Contains(res.Header.Get("Content-Type"), "application/opml+xml") {
			t.Fatalf("expected content-type application/opml+xml, got %s", res.Header.Get("Content-Type"))
		}
		if res.Header.Get("Content-Disposition") != "inline" {
			t.Fatalf("expected content-disposition inline, got %s", res.Header.Get("Content-Disposition"))
		}
		if !strings.Contains(res.Body, "http://example.com/sources/"+*source.UUID+"/feed.xml") {
			t.Fatalf("expected feed URL in response, got %s", res.Body)
		}
		if !strings.Contains(res.Body, "text=\""+source.CustomName+"\"") {
			t.Fatalf("expected source custom_name in response, got %s", res.Body)
		}
	})

	t.Run("returns 401 if the route token is incorrect", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/sources/opml.xml", map[string]string{"route_token": "incorrect"})

		if res.Status != 401 {
			t.Fatalf("expected 401, got %d", res.Status)
		}
		if res.Body != "Unauthorized" {
			t.Fatalf("expected 'Unauthorized', got %s", res.Body)
		}
	})

	t.Run("returns 401 if the route token is missing", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/sources/opml.xml")

		if res.Status != 401 {
			t.Fatalf("expected 401, got %d", res.Status)
		}
		if res.Body != "Unauthorized" {
			t.Fatalf("expected 'Unauthorized', got %s", res.Body)
		}
	})
}

func TestPodcastController_RssFeed(t *testing.T) {
	t.Run("renders the XML document", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.SourceParams{})

		res := c.Get("/sources/" + *source.UUID + "/feed.xml")

		if res.Status != 200 {
			t.Fatalf("expected status 200, got %d", res.Status)
		}
		if !strings.Contains(res.Header.Get("Content-Type"), "application/rss+xml") {
			t.Fatalf("expected content-type application/rss+xml, got %s", res.Header.Get("Content-Type"))
		}
		if res.Header.Get("Content-Disposition") != "inline" {
			t.Fatalf("expected content-disposition inline, got %s", res.Header.Get("Content-Disposition"))
		}
	})
}

func TestPodcastController_FeedImage(t *testing.T) {
	t.Run("returns a feed image if one can be found", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceWithMetadataAttachmentsFixture(t, c.TestApp, store.SourceParams{})

		res := c.Get("/sources/" + *source.UUID + "/feed_image.jpg")

		if res.Status != 200 {
			t.Fatalf("expected status 200, got %d", res.Status)
		}
		if !strings.Contains(res.Header.Get("Content-Type"), "image/jpeg") {
			t.Fatalf("expected content-type image/jpeg, got %s", res.Header.Get("Content-Type"))
		}

		// Verify the response contains image data (at least some content)
		if len(res.Body) == 0 {
			t.Fatalf("expected response body to contain image data")
		}
	})

	t.Run("returns 404 if an image cannot be found", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.SourceParams{})

		res := c.Get("/sources/" + *source.UUID + "/feed_image.jpg")

		if res.Status != 404 {
			t.Fatalf("expected 404, got %d", res.Status)
		}
		if res.Body != "Image not found" {
			t.Fatalf("expected 'Image not found', got %s", res.Body)
		}
	})
}

func TestPodcastController_EpisodeImage(t *testing.T) {
	t.Run("returns an episode image if one can be found", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{})

		res := c.Get("/media/" + *mediaItem.UUID + "/episode_image.jpg")

		if res.Status != 200 {
			t.Fatalf("expected status 200, got %d", res.Status)
		}
		if !strings.Contains(res.Header.Get("Content-Type"), "image/jpeg") {
			t.Fatalf("expected content-type image/jpeg, got %s", res.Header.Get("Content-Type"))
		}

		// Verify the response contains image data (at least some content)
		if len(res.Body) == 0 {
			t.Fatalf("expected response body to contain image data")
		}
	})

	t.Run("returns 404 if an image cannot be found", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := apptest.MediaItemFixture(t, c.TestApp, store.MediaItemParams{})

		res := c.Get("/media/" + *mediaItem.UUID + "/episode_image.jpg")

		if res.Status != 404 {
			t.Fatalf("expected 404, got %d", res.Status)
		}
		if res.Body != "Image not found" {
			t.Fatalf("expected 'Image not found', got %s", res.Body)
		}
	})
}
