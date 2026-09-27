package web_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestMediaItemTableLive_InitialRendering(t *testing.T) {
	t.Run("shows message when no records", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Get(fmt.Sprintf("/_live/sources/%d/media", source.ID), map[string]string{"media_state": "pending"})
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Nothing Here!") {
			t.Errorf("expected 'Nothing Here!' in response")
		}
		if strings.Contains(html, "Showing") {
			t.Errorf("did not expect 'Showing' in response")
		}
	})

	t.Run("shows records when present", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
		mediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID, "media_filepath": nil})

		res := c.Get(fmt.Sprintf("/_live/sources/%d/media", source.ID), map[string]string{"media_state": "pending"})
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Showing") {
			t.Errorf("expected 'Showing' in response")
		}
		if !strings.Contains(html, "Title") {
			t.Errorf("expected 'Title' in response")
		}
		if !strings.Contains(html, core.Deref(mediaItem.Title)) {
			t.Errorf("expected media item title in response")
		}
	})
}

func TestMediaItemTableLive_MediaState(t *testing.T) {
	t.Run("shows pending media when pending", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
		downloadedMediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID})
		pendingMediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID, "media_filepath": nil})

		res := c.Get(fmt.Sprintf("/_live/sources/%d/media", source.ID), map[string]string{"media_state": "pending"})
		html := res.HTML(t, 200)

		if !strings.Contains(html, core.Deref(pendingMediaItem.Title)) {
			t.Errorf("expected pending media item title in response")
		}
		if strings.Contains(html, core.Deref(downloadedMediaItem.Title)) {
			t.Errorf("did not expect downloaded media item title in response")
		}
	})

	t.Run("shows downloaded media when downloaded", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
		downloadedMediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID})
		pendingMediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID, "media_filepath": nil})

		res := c.Get(fmt.Sprintf("/_live/sources/%d/media", source.ID), map[string]string{"media_state": "downloaded"})
		html := res.HTML(t, 200)

		if !strings.Contains(html, core.Deref(downloadedMediaItem.Title)) {
			t.Errorf("expected downloaded media item title in response")
		}
		if strings.Contains(html, core.Deref(pendingMediaItem.Title)) {
			t.Errorf("did not expect pending media item title in response")
		}
	})

	t.Run("shows records that aren't pending or downloaded when other", func(t *testing.T) {
		c := webtest.New(t)
		mediaProfile := coretest.MediaProfileFixture(t, c.TestApp, core.Attrs{"shorts_behaviour": "exclude"})
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{"media_profile_id": mediaProfile.ID})

		downloadedMediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID})
		pendingMediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID, "media_filepath": nil})
		otherMediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{
			"source_id":          source.ID,
			"media_filepath":     nil,
			"short_form_content": true,
		})

		res := c.Get(fmt.Sprintf("/_live/sources/%d/media", source.ID), map[string]string{"media_state": "other"})
		html := res.HTML(t, 200)

		if !strings.Contains(html, core.Deref(otherMediaItem.Title)) {
			t.Errorf("expected other media item title in response")
		}
		if strings.Contains(html, core.Deref(downloadedMediaItem.Title)) {
			t.Errorf("did not expect downloaded media item title in response")
		}
		if strings.Contains(html, core.Deref(pendingMediaItem.Title)) {
			t.Errorf("did not expect pending media item title in response")
		}
	})

	t.Run("shows 'Manually Ignored' column when other", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
		coretest.MediaItemFixture(t, c.TestApp, core.Attrs{
			"source_id":        source.ID,
			"prevent_download": true,
			"media_filepath":   nil,
		})

		res := c.Get(fmt.Sprintf("/_live/sources/%d/media", source.ID), map[string]string{"media_state": "other"})
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Manually Ignored?") {
			t.Errorf("expected 'Manually Ignored?' in response")
		}
	})
}
