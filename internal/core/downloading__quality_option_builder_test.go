package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestQualityOptionBuilder_Build(t *testing.T) {
	t.Run("includes format options if audio_track is set to original", func(t *testing.T) {
		ta := coretest.NewApp(t)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"audio_track": "original"})

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		// Check that the format option is present with the correct value
		found := false
		for _, kv := range res {
			if kv.Key == "format" && kv.Value == "bestvideo+bestaudio[format_note*=original]/bestvideo*+bestaudio/best" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected format option with value \"bestvideo+bestaudio[format_note*=original]/bestvideo*+bestaudio/best\" in %v", res)
		}
	})

	t.Run("includes format options if audio_track is set to default", func(t *testing.T) {
		ta := coretest.NewApp(t)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"audio_track": "default"})

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		// Check that the format option is present with the correct value
		found := false
		for _, kv := range res {
			if kv.Key == "format" && kv.Value == "bestvideo+bestaudio[format_note*='(default)']/bestvideo*+bestaudio/best" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected format option with value \"bestvideo+bestaudio[format_note*='(default)']/bestvideo*+bestaudio/best\" in %v", res)
		}
	})

	t.Run("includes format options if audio_track is set to a language code", func(t *testing.T) {
		ta := coretest.NewApp(t)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"audio_track": "en"})

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		// Check that the format option is present with the correct value
		found := false
		for _, kv := range res {
			if kv.Key == "format" && kv.Value == "bestvideo+bestaudio[language^=en]/bestvideo*+bestaudio/best" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected format option with value \"bestvideo+bestaudio[language^=en]/bestvideo*+bestaudio/best\" in %v", res)
		}
	})
}

func TestQualityOptionBuilder_BuildAudio(t *testing.T) {
	t.Run("includes quality options for audio only", func(t *testing.T) {
		ta := coretest.NewApp(t)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"preferred_resolution": "audio"})

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		// Check for extract_audio flag
		foundExtractAudio := false
		foundFormatSort := false
		foundRemuxVideo := false

		for _, kv := range res {
			if kv.Key == "extract_audio" && kv.Flag {
				foundExtractAudio = true
			}
			if kv.Key == "format_sort" && kv.Value == "+acodec:m4a" {
				foundFormatSort = true
			}
			if kv.Key == "remux_video" {
				foundRemuxVideo = true
			}
		}

		if !foundExtractAudio {
			t.Errorf("expected extract_audio flag in %v", res)
		}
		if !foundFormatSort {
			t.Errorf("expected format_sort option with value \"+acodec:m4a\" in %v", res)
		}
		if foundRemuxVideo {
			t.Errorf("expected no remux_video option in %v", res)
		}
	})

	t.Run("includes custom format target for audio if specified", func(t *testing.T) {
		ta := coretest.NewApp(t)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"preferred_resolution": "audio"})
		updatedProfile, err := ta.App.ProfilesUpdateMediaProfile(ta.Ctx, mediaProfile, core.Attrs{"media_container": "flac", "preferred_resolution": "audio"})
		if err != nil {
			t.Fatalf("failed to update media profile: %v", err)
		}

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, updatedProfile)

		// Check that the audio_format option is present with the correct value
		found := false
		for _, kv := range res {
			if kv.Key == "audio_format" && kv.Value == "flac" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected audio_format option with value \"flac\" in %v", res)
		}
	})

	t.Run("includes custom format options", func(t *testing.T) {
		ta := coretest.NewApp(t)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"preferred_resolution": "audio"})

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		// Check that the format option is present with the correct value
		found := false
		for _, kv := range res {
			if kv.Key == "format" && kv.Value == "bestaudio/best" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected format option with value \"bestaudio/best\" in %v", res)
		}
	})
}

func TestQualityOptionBuilder_BuildNonAudio(t *testing.T) {
	t.Run("includes quality options", func(t *testing.T) {
		resolutions := []string{"360", "480", "720", "1080", "1440", "2160", "4320"}

		for _, resolution := range resolutions {
			t.Run(resolution+"p", func(t *testing.T) {
				ta := coretest.NewApp(t)
				resAttr := resolution + "p"
				mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"preferred_resolution": resAttr})

				res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

				// Check for format_sort with the correct resolution
				foundFormatSort := false
				foundRemuxVideo := false

				for _, kv := range res {
					if kv.Key == "format_sort" && kv.Value == "res:"+resolution+",+codec:avc:m4a" {
						foundFormatSort = true
					}
					if kv.Key == "remux_video" && kv.Value == "mp4" {
						foundRemuxVideo = true
					}
				}

				if !foundFormatSort {
					t.Errorf("expected format_sort option with value \"res:%s,+codec:avc:m4a\" in %v", resolution, res)
				}
				if !foundRemuxVideo {
					t.Errorf("expected remux_video option with value \"mp4\" in %v", res)
				}
			})
		}
	})

	t.Run("includes custom quality options if specified", func(t *testing.T) {
		ta := coretest.NewApp(t)

		_, err := ta.App.SettingsSet(ta.Ctx, core.KW{core.Opt("video_codec_preference", "av01")})
		if err != nil {
			t.Fatalf("failed to set video_codec_preference: %v", err)
		}
		_, err = ta.App.SettingsSet(ta.Ctx, core.KW{core.Opt("audio_codec_preference", "aac")})
		if err != nil {
			t.Fatalf("failed to set audio_codec_preference: %v", err)
		}

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"preferred_resolution": "1080p"})

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		// Check for format_sort with the correct codecs
		found := false
		for _, kv := range res {
			if kv.Key == "format_sort" && kv.Value == "res:1080,+codec:av01:aac" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected format_sort option with value \"res:1080,+codec:av01:aac\" in %v", res)
		}
	})

	t.Run("includes custom remux target for videos if specified", func(t *testing.T) {
		ta := coretest.NewApp(t)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"preferred_resolution": "480p"})
		updatedProfile, err := ta.App.ProfilesUpdateMediaProfile(ta.Ctx, mediaProfile, core.Attrs{"media_container": "mkv"})
		if err != nil {
			t.Fatalf("failed to update media profile: %v", err)
		}

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, updatedProfile)

		// Check that the remux_video option is present with the correct value
		found := false
		for _, kv := range res {
			if kv.Key == "remux_video" && kv.Value == "mkv" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected remux_video option with value \"mkv\" in %v", res)
		}
	})

	t.Run("includes custom format options", func(t *testing.T) {
		ta := coretest.NewApp(t)

		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"preferred_resolution": "480p"})

		res := ta.App.QualityOptionBuilderBuild(ta.Ctx, mediaProfile)

		// Check that the format option is present with the correct value
		found := false
		for _, kv := range res {
			if kv.Key == "format" && kv.Value == "bestvideo*+bestaudio/best" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected format option with value \"bestvideo*+bestaudio/best\" in %v", res)
		}
	})
}
