package app_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestQualityOptionBuilder_Build(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		audioTrack string
		wantFormat string
	}{
		{
			name:       "includes format options if audio_track is set to original",
			audioTrack: "original",
			wantFormat: "bestvideo+bestaudio[format_note*=original]/bestvideo*+bestaudio/best",
		},
		{
			name:       "includes format options if audio_track is set to default",
			audioTrack: "default",
			wantFormat: "bestvideo+bestaudio[format_note*='(default)']/bestvideo*+bestaudio/best",
		},
		{
			name:       "includes format options if audio_track is set to a language code",
			audioTrack: "en",
			wantFormat: "bestvideo+bestaudio[language^=en]/bestvideo*+bestaudio/best",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"audio_track": tt.audioTrack})
			res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

			found := findOption(res, "format", tt.wantFormat)
			if !found {
				t.Errorf("expected format option with value %q in %v", tt.wantFormat, res)
			}
		})
	}
}

func TestQualityOptionBuilder_BuildAudio(t *testing.T) {
	t.Parallel()

	t.Run("includes quality options for audio only", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"preferred_resolution": "audio"})
		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		if !hasFlag(res, "extract_audio") {
			t.Errorf("expected extract_audio flag in %v", res)
		}
		if !findOption(res, "format_sort", "+acodec:m4a") {
			t.Errorf("expected format_sort option with value \"+acodec:m4a\" in %v", res)
		}
		if hasKey(res, "remux_video") {
			t.Errorf("expected no remux_video option in %v", res)
		}
	})

	t.Run("includes custom format target for audio if specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"preferred_resolution": "audio"})
		updatedProfile, err := ta.App.UpdateMediaProfile(ta.Ctx, mediaProfile, store.Attrs{"media_container": "flac", "preferred_resolution": "audio"})
		if err != nil {
			t.Fatalf("failed to update media profile: %v", err)
		}

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, updatedProfile)
		if !findOption(res, "audio_format", "flac") {
			t.Errorf("expected audio_format option with value \"flac\" in %v", res)
		}
	})

	t.Run("includes custom format options", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"preferred_resolution": "audio"})
		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		if !findOption(res, "format", "bestaudio/best") {
			t.Errorf("expected format option with value \"bestaudio/best\" in %v", res)
		}
	})
}

func TestQualityOptionBuilder_BuildNonAudio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resolution string
	}{
		{name: "360p", resolution: "360"},
		{name: "480p", resolution: "480"},
		{name: "720p", resolution: "720"},
		{name: "1080p", resolution: "1080"},
		{name: "1440p", resolution: "1440"},
		{name: "2160p", resolution: "2160"},
		{name: "4320p", resolution: "4320"},
	}

	t.Run("includes quality options", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ta := apptest.NewApp(t)
				mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"preferred_resolution": tt.name})
				res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

				wantFormatSort := "res:" + tt.resolution + ",+codec:avc:m4a"
				if !findOption(res, "format_sort", wantFormatSort) {
					t.Errorf("expected format_sort option with value %q in %v", wantFormatSort, res)
				}
				if !findOption(res, "remux_video", "mp4") {
					t.Errorf("expected remux_video option with value \"mp4\" in %v", res)
				}
			})
		}
	})

	t.Run("includes custom quality options if specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		_, err := ta.App.SetSetting(ta.Ctx, store.KW{store.Opt("video_codec_preference", "av01")})
		if err != nil {
			t.Fatalf("failed to set video_codec_preference: %v", err)
		}
		_, err = ta.App.SetSetting(ta.Ctx, store.KW{store.Opt("audio_codec_preference", "aac")})
		if err != nil {
			t.Fatalf("failed to set audio_codec_preference: %v", err)
		}

		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"preferred_resolution": "1080p"})
		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		if !findOption(res, "format_sort", "res:1080,+codec:av01:aac") {
			t.Errorf("expected format_sort option with value \"res:1080,+codec:av01:aac\" in %v", res)
		}
	})

	t.Run("includes custom remux target for videos if specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"preferred_resolution": "480p"})
		updatedProfile, err := ta.App.UpdateMediaProfile(ta.Ctx, mediaProfile, store.Attrs{"media_container": "mkv"})
		if err != nil {
			t.Fatalf("failed to update media profile: %v", err)
		}

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, updatedProfile)
		if !findOption(res, "remux_video", "mkv") {
			t.Errorf("expected remux_video option with value \"mkv\" in %v", res)
		}
	})

	t.Run("includes custom format options", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{"preferred_resolution": "480p"})
		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		if !findOption(res, "format", "bestvideo*+bestaudio/best") {
			t.Errorf("expected format option with value \"bestvideo*+bestaudio/best\" in %v", res)
		}
	})
}

// findOption searches for an option with a specific key and value.
func findOption(options store.KW, key, value string) bool {
	for _, opt := range options {
		if opt.Key == key && opt.Value == value {
			return true
		}
	}
	return false
}

// hasFlag checks if an option with a specific key and store.Flag set to true exists.
func hasFlag(options store.KW, key string) bool {
	for _, opt := range options {
		if opt.Key == key && opt.Flag {
			return true
		}
	}
	return false
}

// hasKey checks if any option with a specific key exists.
func hasKey(options store.KW, key string) bool {
	for _, opt := range options {
		if opt.Key == key {
			return true
		}
	}
	return false
}
