package core_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestYoutubeRss_Enabled(t *testing.T) {
	t.Run("returns true", func(t *testing.T) {
		ta := coretest.NewApp(t)

		if !ta.YoutubeRssEnabled(ta.Ctx) {
			t.Error("expected YoutubeRssEnabled to return true")
		}
	})
}

func TestYoutubeRss_GetRecentMediaIDs(t *testing.T) {
	t.Run("calls the expected URL for channel sources", func(t *testing.T) {
		t.Skip("BLOCKED: ProfilesCreateMediaProfile unported")
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"collection_type": "channel", "collection_id": "channel_id"})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "https://www.youtube.com/feeds/videos.xml?channel_id=channel_id") {
				t.Errorf("expected URL to contain expected pattern, got: %s", url)
			}
			return "", nil
		})

		_, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("calls the expected URL for playlist sources", func(t *testing.T) {
		t.Skip("BLOCKED: ProfilesCreateMediaProfile unported")
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"collection_type": "playlist", "collection_id": "playlist_id"})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "https://www.youtube.com/feeds/videos.xml?playlist_id=playlist_id") {
				t.Errorf("expected URL to contain expected pattern, got: %s", url)
			}
			return "", nil
		})

		_, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("returns an error if the HTTP request fails", func(t *testing.T) {
		t.Skip("BLOCKED: ProfilesCreateMediaProfile unported")
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "", ErrTest
		})

		_, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err == nil || err.Error() != "Failed to fetch RSS feed" {
			t.Errorf("expected 'Failed to fetch RSS feed' error, got %v", err)
		}
	})

	t.Run("returns the media IDs from the RSS feed", func(t *testing.T) {
		t.Skip("BLOCKED: ProfilesCreateMediaProfile unported")
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId><yt:videoId>test_2</yt:videoId>", nil
		})

		result, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(result) != 2 || result[0] != "test_1" || result[1] != "test_2" {
			t.Errorf("expected [test_1 test_2], got %v", result)
		}
	})

	t.Run("strips whitespace from media IDs", func(t *testing.T) {
		t.Skip("BLOCKED: ProfilesCreateMediaProfile unported")
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId> test_1 </yt:videoId><yt:videoId> test_2 </yt:videoId>", nil
		})

		result, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(result) != 2 || result[0] != "test_1" || result[1] != "test_2" {
			t.Errorf("expected [test_1 test_2], got %v", result)
		}
	})

	t.Run("removes empty media IDs", func(t *testing.T) {
		t.Skip("BLOCKED: ProfilesCreateMediaProfile unported")
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId><yt:videoId></yt:videoId>", nil
		})

		result, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(result) != 1 || result[0] != "test_1" {
			t.Errorf("expected [test_1], got %v", result)
		}
	})

	t.Run("removes duplicate media IDs", func(t *testing.T) {
		t.Skip("BLOCKED: ProfilesCreateMediaProfile unported")
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "<yt:videoId>test_1</yt:videoId><yt:videoId>test_1</yt:videoId>", nil
		})

		result, err := ta.YoutubeRssGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(result) != 1 || result[0] != "test_1" {
			t.Errorf("expected [test_1], got %v", result)
		}
	})
}
