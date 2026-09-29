package app_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// newBuildItem creates a profile, source and media item (with everything
// preloaded) for building download options.
func newBuildItem(t *testing.T, ta *apptest.TestApp, profile store.MediaProfileParams, src store.SourceParams) *store.MediaItem {
	t.Helper()
	src.MediaProfileID = store.Ptr(apptest.MediaProfileFixture(t, ta, profile).ID)
	source := apptest.SourceFixture(t, ta, src)
	mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
	mediaItem, err := ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)
	must(t, err)
	return mediaItem
}

func buildOpts(t *testing.T, ta *apptest.TestApp, mediaItem *store.MediaItem, o app.DownloadOverrides) ytdlp.Args {
	t.Helper()
	res, err := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, o)
	must(t, err)
	return res
}

// hasOpt reports whether res holds the option described by spec: "key" is a
// flag option, "key=value" a valued one and "key*" any option with that key.
func hasOpt(res ytdlp.Args, spec string) bool {
	if k, v, ok := strings.Cut(spec, "="); ok {
		return findOption(res, k, v)
	}
	if k, ok := strings.CutSuffix(spec, "*"); ok {
		return hasKey(res, k)
	}
	return hasFlag(res, spec)
}

func wantOpts(t *testing.T, res ytdlp.Args, present, absent []string) {
	t.Helper()
	for _, s := range present {
		if !hasOpt(res, s) {
			t.Errorf("expected option %q in %v", s, res)
		}
	}
	for _, s := range absent {
		if hasOpt(res, s) {
			t.Errorf("option %q should not be in %v", s, res)
		}
	}
}

func TestDownloadOptionBuilder_Build_WhenTestingOutputOptions(t *testing.T) {
	title := store.Ptr("{{ title }}.%(ext)s")
	for _, c := range []struct {
		name         string
		profile      store.MediaProfileParams
		src          store.SourceParams
		prefix, file string
	}{
		{"it generates an expanded output path based on the given template", store.MediaProfileParams{OutputPathTemplate: title}, store.SourceParams{}, "", "%(title)S.%(ext)s"},
		{"it respects custom output path options", store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ source_custom_name }}.%(ext)s")}, store.SourceParams{CustomName: store.Ptr("my custom source")}, "", "my custom source.%(ext)s"},
		{"respects custom media_item-related output path options", store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ media_upload_date_index }}.%(ext)s")}, store.SourceParams{}, "", "99.%(ext)s"},
		{"uses source's output override if present", store.MediaProfileParams{OutputPathTemplate: title}, store.SourceParams{OutputPathTemplateOverride: store.Ptr("override.%(ext)s")}, "", "override.%(ext)s"},
		{"appends -thumb to the thumbnail name when download_thumbnail is true", store.MediaProfileParams{DownloadThumbnail: store.Ptr(true), OutputPathTemplate: title}, store.SourceParams{}, "thumbnail:", "%(title)S-thumb.%(ext)s"},
		{"appends -thumb to source's output path override, if present", store.MediaProfileParams{DownloadThumbnail: store.Ptr(true)}, store.SourceParams{OutputPathTemplateOverride: store.Ptr("override.%(ext)s")}, "thumbnail:", "override-thumb.%(ext)s"},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			mediaItem := newBuildItem(t, ta, c.profile, c.src)

			res := buildOpts(t, ta, mediaItem, app.DownloadOverrides{})

			wantOpts(t, res, []string{"output=" + c.prefix + filepath.Join(ta.Config.MediaDirectory, c.file)}, nil)
		})
	}
}

func TestDownloadOptionBuilder_Build_WhenTestingDefaultOptions(t *testing.T) {
	profile := store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")}

	t.Run("it includes default options", func(t *testing.T) {
		ta := apptest.NewApp(t)
		res := buildOpts(t, ta, newBuildItem(t, ta, profile, store.SourceParams{}), app.DownloadOverrides{})

		wantOpts(t, res, []string{"no_progress", "force_overwrites", "parse_metadata=%(upload_date>%Y-%m-%d)s:(?P<meta_date>.+)"}, nil)
	})

	t.Run("includes override options if specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		res := buildOpts(t, ta, newBuildItem(t, ta, profile, store.SourceParams{}), app.DownloadOverrides{OverwriteBehaviour: "no_force_overwrites"})

		wantOpts(t, res, []string{"no_force_overwrites"}, []string{"force_overwrites"})
	})
}

// TestDownloadOptionBuilder_Build_ProfileOptions covers how media profile
// settings (subtitles, thumbnails, metadata, quality, sponsorblock) map to
// yt-dlp options.
func TestDownloadOptionBuilder_Build_ProfileOptions(t *testing.T) {
	yes, no := store.Ptr(true), store.Ptr(false)
	sponsor := func(b store.MediaProfileSponsorblockBehaviour, cats ...string) store.MediaProfileParams {
		return store.MediaProfileParams{SponsorblockBehaviour: store.Ptr(b), SponsorblockCategories: store.Ptr(cats)}
	}
	audio := store.Ptr(store.MediaProfilePreferredResolutionAudio)
	const videoQuality = "format_sort=res:1080,+codec:avc:m4a"

	for _, c := range []struct {
		name            string
		profile         store.MediaProfileParams
		present, absent []string
	}{
		// subtitles
		{"includes :write_subs option when specified", store.MediaProfileParams{DownloadSubs: yes}, []string{"write_subs"}, nil},
		{"forces SRT format when download_subs is true", store.MediaProfileParams{DownloadSubs: yes}, []string{"convert_subs=srt"}, nil},
		{"includes :write_auto_subs option when download_subs is set", store.MediaProfileParams{DownloadSubs: yes, DownloadAutoSubs: yes}, []string{"write_auto_subs"}, nil},
		{"includes :write_auto_subs option when embed_subs is set", store.MediaProfileParams{EmbedSubs: yes, DownloadAutoSubs: yes}, []string{"write_auto_subs"}, nil},
		{"doesn't include :write_auto_subs option when download_subs and embed_subs is false", store.MediaProfileParams{DownloadSubs: no, EmbedSubs: no, DownloadAutoSubs: yes}, nil, []string{"write_auto_subs"}},
		{"includes :embed_subs option when specified", store.MediaProfileParams{EmbedSubs: yes}, []string{"embed_subs"}, nil},
		{"doesn't include :embed_subs option when preferred_resolution is :audio", store.MediaProfileParams{EmbedSubs: yes, PreferredResolution: audio}, nil, []string{"embed_subs"}},
		{"includes sub_langs option when download_subs is true", store.MediaProfileParams{DownloadSubs: yes, SubLangs: store.Ptr("en")}, []string{"sub_langs=en"}, nil},
		{"includes sub_langs option when embed_subs is true", store.MediaProfileParams{EmbedSubs: yes, SubLangs: store.Ptr("en")}, []string{"sub_langs=en"}, nil},
		{"doesn't include sub_langs option when neither downloading nor embedding", store.MediaProfileParams{EmbedSubs: no, DownloadSubs: no, SubLangs: store.Ptr("en")}, nil, []string{"sub_langs=en"}},
		// thumbnails
		{"includes :write_thumbnail option when specified", store.MediaProfileParams{DownloadThumbnail: yes}, []string{"write_thumbnail"}, nil},
		{"converts thumbnail to jpg when download_thumbnail is true", store.MediaProfileParams{DownloadThumbnail: yes}, []string{"convert_thumbnail=jpg"}, nil},
		{"includes :embed_thumbnail option when specified", store.MediaProfileParams{EmbedThumbnail: yes}, []string{"embed_thumbnail"}, nil},
		{"convertes thumbnail to jpg when embed_thumbnail is true", store.MediaProfileParams{EmbedThumbnail: yes}, []string{"convert_thumbnail=jpg"}, nil},
		{"doesn't include thumbnail options when not specified", store.MediaProfileParams{EmbedThumbnail: no, DownloadThumbnail: no}, nil, []string{"write_thumbnail", "embed_thumbnail"}},
		// metadata
		{"includes :write_info_json option when specified", store.MediaProfileParams{DownloadMetadata: yes}, []string{"write_info_json", "clean_info_json"}, nil},
		{"includes :embed_metadata option when specified", store.MediaProfileParams{EmbedMetadata: yes}, []string{"embed_metadata"}, nil},
		{"doesn't include metadata options when not specified", store.MediaProfileParams{EmbedMetadata: no, DownloadMetadata: no}, nil, []string{"write_info_json", "clean_info_json", "embed_metadata"}},
		// quality and format
		{"includes video options for video profiles", store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")}, []string{videoQuality, "remux_video=mp4"}, nil},
		{"includes quality options for audio only", store.MediaProfileParams{PreferredResolution: audio}, []string{"extract_audio", "format_sort=+acodec:m4a"}, []string{"remux_video*"}},
		// sponsorblock
		{"includes :sponsorblock_remove option when specified", sponsor(store.MediaProfileSponsorblockBehaviourRemove, "sponsor", "intro"), []string{"sponsorblock_remove=sponsor,intro"}, nil},
		{"includes :sponsorblock_mark option when specified", sponsor(store.MediaProfileSponsorblockBehaviourMark, "sponsor", "intro"), []string{"sponsorblock_mark=sponsor,intro"}, nil},
		{"does not include any sponsorblock option without categories", sponsor(store.MediaProfileSponsorblockBehaviourRemove), nil, []string{"sponsorblock_remove*", "sponsorblock_mark*"}},
		{"does not include any sponsorblock options when disabled", store.MediaProfileParams{SponsorblockBehaviour: store.Ptr(store.MediaProfileSponsorblockBehaviourDisabled)}, nil, []string{"sponsorblock_remove*", "sponsorblock_mark*"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			res := buildOpts(t, ta, newBuildItem(t, ta, c.profile, store.SourceParams{}), app.DownloadOverrides{})

			wantOpts(t, res, c.present, c.absent)
		})
	}
}

func TestDownloadOptionBuilder_BuildOutputPathFor(t *testing.T) {
	setup := func(t *testing.T, src store.SourceParams) (*apptest.TestApp, *store.MediaItem) {
		ta := apptest.NewApp(t)
		return ta, newBuildItem(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")}, src)
	}
	wantPath := func(t *testing.T, ta *apptest.TestApp, got, file string) {
		t.Helper()
		if expected := filepath.Join(ta.Config.MediaDirectory, file); got != expected {
			t.Errorf("expected %s, got %s", expected, got)
		}
	}

	t.Run("builds an output path for a media item", func(t *testing.T) {
		ta, mediaItem := setup(t, store.SourceParams{})
		wantPath(t, ta, ta.App.DownloadOptionBuilderBuildOutputPathForMediaItem(ta.Ctx, mediaItem), "%(title)S.%(ext)s")
	})

	t.Run("builds an output path for a source", func(t *testing.T) {
		ta, mediaItem := setup(t, store.SourceParams{})
		wantPath(t, ta, ta.App.DownloadOptionBuilderBuildOutputPathForSource(ta.Ctx, mediaItem.Source), "%(title)S.%(ext)s")
	})

	t.Run("uses source's output override if present", func(t *testing.T) {
		ta, mediaItem := setup(t, store.SourceParams{})
		updatedSource, err := ta.App.SourcesUpdateSource(ta.Ctx, mediaItem.Source, store.SourceParams{OutputPathTemplateOverride: store.Ptr("override.%(ext)s")}, true)
		must(t, err)

		wantPath(t, ta, ta.App.DownloadOptionBuilderBuildOutputPathForSource(ta.Ctx, updatedSource), "override.%(ext)s")
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingConfigFileOptions(t *testing.T) {
	// configFile names (relative to the configs dir) of a config file to
	// write, "" for none.
	base := func(*store.MediaItem) string { return "base-config.txt" }
	for _, c := range []struct {
		name    string
		file    func(*store.MediaItem) string
		content string
		want    bool
	}{
		{"includes base config file if it's present", base, "base config", true},
		{"includes media profile config file if it's present", func(m *store.MediaItem) string { return fmt.Sprintf("media-profile-%d-config.txt", m.Source.MediaProfileID) }, "profile config", true},
		{"includes source config file if it's present", func(m *store.MediaItem) string { return fmt.Sprintf("source-%d-config.txt", m.SourceID) }, "source config", true},
		{"includes media item config file if it's present", func(m *store.MediaItem) string { return fmt.Sprintf("media-item-%d-config.txt", m.ID) }, "media item config", true},
		{"does not include config file options if they are not present", func(*store.MediaItem) string { return "" }, "", false},
		{"does not return a config file if it's blank", base, " \n \n ", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			mediaItem := newBuildItem(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")}, store.SourceParams{})
			configPath := filepath.Join(ta.Config.ExtrasDirectory, "yt-dlp-configs", c.file(mediaItem))
			if c.file(mediaItem) != "" {
				must(t, fsutil.WriteFileAll(configPath, c.content))
			}

			res := buildOpts(t, ta, mediaItem, app.DownloadOverrides{})

			if c.want {
				wantOpts(t, res, []string{"config_locations=" + configPath}, nil)
			} else {
				wantOpts(t, res, nil, []string{"config_locations*"})
			}
		})
	}

	t.Run("returns config files in order of precedence", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := newBuildItem(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")}, store.SourceParams{})
		baseDir := filepath.Join(ta.Config.ExtrasDirectory, "yt-dlp-configs")

		baseFilepath := filepath.Join(baseDir, "base-config.txt")
		sourceFilepath := filepath.Join(baseDir, fmt.Sprintf("source-%d-config.txt", mediaItem.SourceID))
		mediaItemFilepath := filepath.Join(baseDir, fmt.Sprintf("media-item-%d-config.txt", mediaItem.ID))
		mediaProfileFilepath := filepath.Join(baseDir, fmt.Sprintf("media-profile-%d-config.txt", mediaItem.Source.MediaProfileID))

		for _, p := range []string{baseFilepath, sourceFilepath, mediaItemFilepath, mediaProfileFilepath} {
			must(t, fsutil.WriteFileAll(p, "config"))
		}

		var gotOrder []string
		for _, kv := range buildOpts(t, ta, mediaItem, app.DownloadOverrides{}) {
			if kv.Key == "config_locations" {
				gotOrder = append(gotOrder, kv.Value.(string))
			}
		}

		expectedOrder := []string{baseFilepath, mediaProfileFilepath, sourceFilepath, mediaItemFilepath}
		if len(gotOrder) != len(expectedOrder) {
			t.Fatalf("expected %v, got %v", expectedOrder, gotOrder)
		}
		for i := range expectedOrder {
			if gotOrder[i] != expectedOrder[i] {
				t.Errorf("expected order %v, got %v", expectedOrder, gotOrder)
				break
			}
		}
	})
}

func TestDownloadOptionBuilder_BuildQualityOptionsFor(t *testing.T) {
	quality := []string{"format_sort=res:1080,+codec:avc:m4a", "remux_video=mp4"}

	t.Run("builds quality options for a media item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, err := ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)
		must(t, err)

		wantOpts(t, ta.App.DownloadOptionBuilderBuildQualityOptionsForMediaItem(ta.Ctx, mediaItem), quality, nil)
	})

	t.Run("builds quality options for a source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, err := ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)
		must(t, err)

		wantOpts(t, ta.App.DownloadOptionBuilderBuildQualityOptionsForSource(ta.Ctx, mediaItem.Source), quality, nil)
	})
}
