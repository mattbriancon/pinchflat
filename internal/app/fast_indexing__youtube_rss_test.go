package app_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestYoutubeRss_Enabled(t *testing.T) {
	t.Parallel()
	t.Run("returns true", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		if !ta.YoutubeRssEnabled(ta.Ctx) {
			t.Error("expected true")
		}
	})
}

func TestYoutubeRss_GetRecentMediaIDs(t *testing.T) {
	t.Parallel()
	t.Run("builds URL for channel", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "channel", "collection_id": "channel_id"})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			if !strings.Contains(url, "https://www.youtube.com/feeds/videos.xml?channel_id=channel_id") {
				t.Errorf("url=%s, want channel pattern", url)
			}
			return "", nil
		})

		_, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("builds URL for playlist", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"collection_type": "playlist", "collection_id": "playlist_id"})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			if !strings.Contains(url, "https://www.youtube.com/feeds/videos.xml?playlist_id=playlist_id") {
				t.Errorf("url=%s, want playlist pattern", url)
			}
			return "", nil
		})

		_, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns fetch error", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "", ErrTest
		})

		_, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err == nil || err.Error() != "Failed to fetch RSS feed" {
			t.Errorf("err=%v, want 'Failed to fetch RSS feed'", err)
		}
	})

	t.Run("returns media IDs", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId><yt:videoId>test_2</yt:videoId>", nil
		})

		result, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 2 || result[0] != "test_1" || result[1] != "test_2" {
			t.Errorf("result=%v, want [test_1 test_2]", result)
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "<yt:videoId> test_1 </yt:videoId><yt:videoId> test_2 </yt:videoId>", nil
		})

		result, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 2 || result[0] != "test_1" || result[1] != "test_2" {
			t.Errorf("result=%v, want [test_1 test_2]", result)
		}
	})

	t.Run("filters empty IDs", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId><yt:videoId></yt:videoId>", nil
		})

		result, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "test_1" {
			t.Errorf("result=%v, want [test_1]", result)
		}
	})

	t.Run("deduplicates IDs", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts store.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId><yt:videoId>test_1</yt:videoId>", nil
		})

		result, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "test_1" {
			t.Errorf("result=%v, want [test_1]", result)
		}
	})
}
