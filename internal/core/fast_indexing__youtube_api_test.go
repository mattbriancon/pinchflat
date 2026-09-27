package core_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestYoutubeApi_Enabled(t *testing.T) {
	t.Run("returns true if the user has set YouTube API keys", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		if !ta.YoutubeApiEnabled(ta.Ctx) {
			t.Error("expected YoutubeApiEnabled to return true")
		}
	})

	t.Run("returns true with a single API key", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "test_key")})

		if !ta.YoutubeApiEnabled(ta.Ctx) {
			t.Error("expected YoutubeApiEnabled to return true")
		}
	})

	t.Run("returns false if the user has not set any API keys", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", nil)})

		if ta.YoutubeApiEnabled(ta.Ctx) {
			t.Error("expected YoutubeApiEnabled to return false")
		}
	})

	t.Run("returns false if only empty or whitespace keys are provided", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "  ,  ,")})

		if ta.YoutubeApiEnabled(ta.Ctx) {
			t.Error("expected YoutubeApiEnabled to return false")
		}
	})
}

func TestYoutubeApi_GetRecentMediaIDs(t *testing.T) {
	t.Run("rotates through API keys", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "key=key1") {
				t.Errorf("expected URL to contain key=key1, got: %s", url)
			}
			return "{}", nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "key=key2") {
				t.Errorf("expected URL to contain key=key2, got: %s", url)
			}
			return "{}", nil
		})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "key=key1") {
				t.Errorf("expected URL to contain key=key1, got: %s", url)
			}
			return "{}", nil
		})

		// three calls to verify rotation
		ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
	})

	t.Run("calls the expected URL", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			apiBase := "https://youtube.googleapis.com/youtube/v3/playlistItems"
			expectedURL := apiBase + "?part=contentDetails&maxResults=50&playlistId=" + source.CollectionID + "&key=key1"

			if url != expectedURL {
				t.Errorf("expected URL %s, got %s", expectedURL, url)
			}

			// Check headers
			accept, ok := headers.Get("accept")
			if !ok || accept != "application/json" {
				t.Errorf("expected headers to contain accept=application/json, got %v", headers)
			}

			return "{}", nil
		})

		_, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("replaces channel IDs with playlist IDs if needed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"collection_id": "UC_ABC123"})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "playlistId=UU_ABC123&") {
				t.Errorf("expected URL to contain playlistId=UU_ABC123&, got: %s", url)
			}
			return "{}", nil
		})

		_, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("returns an empty list if no media is returned", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "{}", nil
		})

		result, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty list, got %v", result)
		}
	})

	t.Run("returns media IDs if present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return `{
  "items": [
    {"contentDetails": {"videoId": "test_1"}},
    {"contentDetails": {"videoId": "test_2"}}
  ]
}`, nil
		})

		result, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(result) != 2 || result[0] != "test_1" || result[1] != "test_2" {
			t.Errorf("expected [test_1 test_2], got %v", result)
		}
	})

	t.Run("returns an error if the HTTP request fails", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "", ErrTest
		})

		_, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err == nil {
			t.Error("expected an error")
		}
	})
}

var ErrTest = errorTest{}

type errorTest struct{}

func (e errorTest) Error() string {
	return "error"
}
