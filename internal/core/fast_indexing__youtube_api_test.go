package core_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestYoutubeApi_Enabled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		keyVal any
		want   bool
	}{
		{"multiple keys", "key1, key2", true},
		{"single key", "test_key", true},
		{"nil", nil, false},
		{"whitespace only", "  ,  ,", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ta := coretest.NewApp(t)
			ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", tt.keyVal)})

			got := ta.YoutubeApiEnabled(ta.Ctx)
			if got != tt.want {
				t.Errorf("enabled=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestYoutubeApi_GetRecentMediaIDs(t *testing.T) {
	t.Parallel()
	t.Run("rotates keys", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "key=key1") {
				t.Errorf("url missing key1: %s", url)
			}
			return "{}", nil
		})
		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "key=key2") {
				t.Errorf("url missing key2: %s", url)
			}
			return "{}", nil
		})
		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "key=key1") {
				t.Errorf("url missing key1: %s", url)
			}
			return "{}", nil
		})

		ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
	})

	t.Run("builds correct URL", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			apiBase := "https://youtube.googleapis.com/youtube/v3/playlistItems"
			expectedURL := apiBase + "?part=contentDetails&maxResults=50&playlistId=" + source.CollectionID + "&key=key1"

			if url != expectedURL {
				t.Errorf("url=%s, want %s", url, expectedURL)
			}

			accept, ok := headers.Get("accept")
			if !ok || accept != "application/json" {
				t.Errorf("accept=%v, want application/json", accept)
			}

			return "{}", nil
		})

		_, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("converts channel to playlist ID", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"collection_id": "UC_ABC123"})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			if !strings.Contains(url, "playlistId=UU_ABC123&") {
				t.Errorf("url missing converted ID: %s", url)
			}
			return "{}", nil
		})

		_, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns empty when no media", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "{}", nil
		})

		result, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("result=%v, want empty", result)
		}
	})

	t.Run("returns media IDs", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return `{"items":[{"contentDetails":{"videoId":"test_1"}},{"contentDetails":{"videoId":"test_2"}}]}`, nil
		})

		result, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 2 || result[0] != "test_1" || result[1] != "test_2" {
			t.Errorf("result=%v, want [test_1 test_2]", result)
		}
	})

	t.Run("propagates errors", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		ta.SettingsSet(ta.Ctx, core.KW{core.Opt("youtube_api_key", "key1, key2")})

		ta.HTTPMock.Get.Expect(func(url string, headers, opts core.KW) (string, error) {
			return "", ErrTest
		})

		_, err := ta.YoutubeApiGetRecentMediaIDs(ta.Ctx, source)
		if err == nil {
			t.Error("expected error")
		}
	})
}

var ErrTest = errorTest{}

type errorTest struct{}

func (e errorTest) Error() string {
	return "error"
}
