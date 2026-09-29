package app_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestDownloadOptionBuilder_Build_WhenTestingOutputOptions(t *testing.T) {
	t.Run("it generates an expanded output path based on the given template", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		expected := filepath.Join(ta.Config.MediaDirectory, "%(title)S.%(ext)s")
		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected output option in result, got %v", res)
		}
	})

	t.Run("it respects custom output path options", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ source_custom_name }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		expected := filepath.Join(ta.Config.MediaDirectory, mediaItem.Source.CustomName+".%(ext)s")
		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected output option with custom name in result, got %v", res)
		}
	})

	t.Run("respects custom media_item-related output path options", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ media_upload_date_index }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		expected := filepath.Join(ta.Config.MediaDirectory, "99.%(ext)s")
		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected output option with upload date index in result, got %v", res)
		}
	})

	t.Run("uses source's output override if present", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)
		ta.App.SourcesUpdateSource(ta.Ctx, mediaItem.Source, store.Attrs{"output_path_template_override": "override.%(ext)s"}, store.KW{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		expected := filepath.Join(ta.Config.MediaDirectory, "override.%(ext)s")
		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected output option with override in result, got %v", res)
		}
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingDefaultOptions(t *testing.T) {
	t.Run("it includes default options", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		hasNoProgress := false
		hasForceOverwrites := false
		hasParseMetadata := false
		for _, kv := range res {
			if kv.Key == "no_progress" && kv.Flag {
				hasNoProgress = true
			}
			if kv.Key == "force_overwrites" && kv.Flag {
				hasForceOverwrites = true
			}
			if kv.Key == "parse_metadata" && kv.Value == "%(upload_date>%Y-%m-%d)s:(?P<meta_date>.+)" {
				hasParseMetadata = true
			}
		}
		if !hasNoProgress || !hasForceOverwrites || !hasParseMetadata {
			t.Errorf("missing default options in result")
		}
	})

	t.Run("includes override options if specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{store.Opt("overwrite_behaviour", "no_force_overwrites")})

		hasForceOverwrites := false
		hasNoForceOverwrites := false
		for _, kv := range res {
			if kv.Key == "force_overwrites" && kv.Flag {
				hasForceOverwrites = true
			}
			if kv.Key == "no_force_overwrites" && kv.Flag {
				hasNoForceOverwrites = true
			}
		}
		if hasForceOverwrites || !hasNoForceOverwrites {
			t.Errorf("override options not respected")
		}
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingSubtitleOptions(t *testing.T) {
	t.Run("includes :write_subs option when specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSubs: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "write_subs" && kv.Flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected write_subs option in result")
		}
	})

	t.Run("forces SRT format when download_subs is true", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSubs: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "convert_subs" && kv.Value == "srt" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected convert_subs option in result")
		}
	})

	t.Run("includes :write_auto_subs option when specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile1 := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSubs: store.Ptr(true), DownloadAutoSubs: store.Ptr(true)})
		mediaProfile2 := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedSubs: store.Ptr(true), DownloadAutoSubs: store.Ptr(true)})
		source1 := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile1.ID})
		source2 := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile2.ID})
		mediaItem1 := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source1.ID})
		mediaItem2 := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source2.ID})
		mediaItem1, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem1)
		mediaItem2, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem2)

		res1, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem1, store.KW{})
		res2, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem2, store.KW{})

		found1 := false
		found2 := false
		for _, kv := range res1 {
			if kv.Key == "write_auto_subs" && kv.Flag {
				found1 = true
				break
			}
		}
		for _, kv := range res2 {
			if kv.Key == "write_auto_subs" && kv.Flag {
				found2 = true
				break
			}
		}
		if !found1 || !found2 {
			t.Errorf("expected write_auto_subs option in results")
		}
	})

	t.Run("doesn't include :write_auto_subs option when download_subs and embed_subs is false", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSubs: store.Ptr(false), EmbedSubs: store.Ptr(false), DownloadAutoSubs: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "write_auto_subs" && kv.Flag {
				found = true
				break
			}
		}
		if found {
			t.Errorf("write_auto_subs should not be in result")
		}
	})

	t.Run("includes :embed_subs option when specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedSubs: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "embed_subs" && kv.Flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected embed_subs option in result")
		}
	})

	t.Run("doesn't include :embed_subs option when preferred_resolution is :audio", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedSubs: store.Ptr(true), PreferredResolution: store.Ptr(store.MediaProfilePreferredResolutionAudio)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "embed_subs" && kv.Flag {
				found = true
				break
			}
		}
		if found {
			t.Errorf("embed_subs should not be in result when audio")
		}
	})

	t.Run("includes sub_langs option when download_subs is true", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSubs: store.Ptr(true), SubLangs: store.Ptr("en")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "sub_langs" && kv.Value == "en" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected sub_langs option in result")
		}
	})

	t.Run("includes sub_langs option when embed_subs is true", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedSubs: store.Ptr(true), SubLangs: store.Ptr("en")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "sub_langs" && kv.Value == "en" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected sub_langs option in result")
		}
	})

	t.Run("doesn't include sub_langs option when neither downloading nor embedding", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedSubs: store.Ptr(false), DownloadSubs: store.Ptr(false), SubLangs: store.Ptr("en")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "sub_langs" && kv.Value == "en" {
				found = true
				break
			}
		}
		if found {
			t.Errorf("sub_langs should not be in result")
		}
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingThumbnailOptions(t *testing.T) {
	t.Run("includes :write_thumbnail option when specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadThumbnail: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "write_thumbnail" && kv.Flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected write_thumbnail option in result")
		}
	})

	t.Run("appends -thumb to the thumbnail name when download_thumbnail is true", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadThumbnail: store.Ptr(true), OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		expected := "thumbnail:" + filepath.Join(ta.Config.MediaDirectory, "%(title)S-thumb.%(ext)s")
		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected thumbnail output option in result, got %v", res)
		}
	})

	t.Run("appends -thumb to source's output path override, if present", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadThumbnail: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		ta.App.SourcesUpdateSource(ta.Ctx, source, store.Attrs{"output_path_template_override": "override.%(ext)s"}, store.KW{})
		source, _ = ta.App.PreloadSourceMediaProfile(ta.Ctx, source)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		expected := "thumbnail:" + filepath.Join(ta.Config.MediaDirectory, "override-thumb.%(ext)s")
		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected thumbnail output option with override in result, got %v", res)
		}
	})

	t.Run("converts thumbnail to jpg when download_thumbnail is true", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadThumbnail: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "convert_thumbnail" && kv.Value == "jpg" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected convert_thumbnail option in result")
		}
	})

	t.Run("includes :embed_thumbnail option when specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedThumbnail: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "embed_thumbnail" && kv.Flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected embed_thumbnail option in result")
		}
	})

	t.Run("convertes thumbnail to jpg when embed_thumbnail is true", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedThumbnail: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "convert_thumbnail" && kv.Value == "jpg" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected convert_thumbnail option in result")
		}
	})

	t.Run("doesn't include these options when not specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedThumbnail: store.Ptr(false), DownloadThumbnail: store.Ptr(false)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		hasWriteThumbnail := false
		hasEmbedThumbnail := false
		for _, kv := range res {
			if kv.Key == "write_thumbnail" && kv.Flag {
				hasWriteThumbnail = true
			}
			if kv.Key == "embed_thumbnail" && kv.Flag {
				hasEmbedThumbnail = true
			}
		}
		if hasWriteThumbnail || hasEmbedThumbnail {
			t.Errorf("thumbnail options should not be in result")
		}
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingMetadataOptions(t *testing.T) {
	t.Run("includes :write_info_json option when specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadMetadata: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		hasWriteInfoJson := false
		hasCleanInfoJson := false
		for _, kv := range res {
			if kv.Key == "write_info_json" && kv.Flag {
				hasWriteInfoJson = true
			}
			if kv.Key == "clean_info_json" && kv.Flag {
				hasCleanInfoJson = true
			}
		}
		if !hasWriteInfoJson || !hasCleanInfoJson {
			t.Errorf("expected metadata options in result")
		}
	})

	t.Run("includes :embed_metadata option when specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedMetadata: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "embed_metadata" && kv.Flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected embed_metadata option in result")
		}
	})

	t.Run("doesn't include these options when not specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{EmbedMetadata: store.Ptr(false), DownloadMetadata: store.Ptr(false)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		hasWriteInfoJson := false
		hasCleanInfoJson := false
		hasEmbedMetadata := false
		for _, kv := range res {
			if kv.Key == "write_info_json" && kv.Flag {
				hasWriteInfoJson = true
			}
			if kv.Key == "clean_info_json" && kv.Flag {
				hasCleanInfoJson = true
			}
			if kv.Key == "embed_metadata" && kv.Flag {
				hasEmbedMetadata = true
			}
		}
		if hasWriteInfoJson || hasCleanInfoJson || hasEmbedMetadata {
			t.Errorf("metadata options should not be in result")
		}
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingMediaQualityAndFormatOptions(t *testing.T) {
	t.Run("includes video options for video profiles", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		hasFormatSort := false
		hasRemuxVideo := false
		for _, kv := range res {
			if kv.Key == "format_sort" && kv.Value == "res:1080,+codec:avc:m4a" {
				hasFormatSort = true
			}
			if kv.Key == "remux_video" && kv.Value == "mp4" {
				hasRemuxVideo = true
			}
		}
		if !hasFormatSort || !hasRemuxVideo {
			t.Errorf("expected video quality options in result")
		}
	})

	t.Run("includes quality options for audio only", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{PreferredResolution: store.Ptr(store.MediaProfilePreferredResolutionAudio)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		hasExtractAudio := false
		hasFormatSort := false
		hasRemuxVideo := false
		for _, kv := range res {
			if kv.Key == "extract_audio" && kv.Flag {
				hasExtractAudio = true
			}
			if kv.Key == "format_sort" && kv.Value == "+acodec:m4a" {
				hasFormatSort = true
			}
			if kv.Key == "remux_video" {
				hasRemuxVideo = true
			}
		}
		if !hasExtractAudio || !hasFormatSort || hasRemuxVideo {
			t.Errorf("expected audio quality options in result")
		}
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingSponsorblockOptions(t *testing.T) {
	t.Run("includes :sponsorblock_remove option when specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{
			SponsorblockBehaviour:  store.Ptr(store.MediaProfileSponsorblockBehaviourRemove),
			SponsorblockCategories: store.Ptr([]string{"sponsor", "intro"}),
		})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "sponsorblock_remove" && kv.Value == "sponsor,intro" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected sponsorblock_remove option in result")
		}
	})

	t.Run("includes :sponsorblock_mark option when specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{
			SponsorblockBehaviour:  store.Ptr(store.MediaProfileSponsorblockBehaviourMark),
			SponsorblockCategories: store.Ptr([]string{"sponsor", "intro"}),
		})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "sponsorblock_mark" && kv.Value == "sponsor,intro" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected sponsorblock_mark option in result")
		}
	})

	t.Run("does not include any sponsorblock option without categories", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{
			SponsorblockBehaviour:  store.Ptr(store.MediaProfileSponsorblockBehaviourRemove),
			SponsorblockCategories: store.Ptr([]string{}),
		})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		hasSponsorblock := false
		for _, kv := range res {
			if kv.Key == "sponsorblock_remove" || kv.Key == "sponsorblock_mark" {
				hasSponsorblock = true
				break
			}
		}
		if hasSponsorblock {
			t.Errorf("sponsorblock options should not be in result")
		}
	})

	t.Run("does not include any sponsorblock options when disabled", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{SponsorblockBehaviour: store.Ptr(store.MediaProfileSponsorblockBehaviourDisabled)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		hasSponsorblock := false
		for _, kv := range res {
			if kv.Key == "sponsorblock_remove" || kv.Key == "sponsorblock_mark" {
				hasSponsorblock = true
				break
			}
		}
		if hasSponsorblock {
			t.Errorf("sponsorblock options should not be in result")
		}
	})
}

func TestDownloadOptionBuilder_BuildOutputPathFor(t *testing.T) {
	t.Run("builds an output path for a media item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		path := ta.App.DownloadOptionBuilderBuildOutputPathForMediaItem(ta.Ctx, mediaItem)

		expected := filepath.Join(ta.Config.MediaDirectory, "%(title)S.%(ext)s")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	})

	t.Run("builds an output path for a source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		path := ta.App.DownloadOptionBuilderBuildOutputPathForSource(ta.Ctx, mediaItem.Source)

		expected := filepath.Join(ta.Config.MediaDirectory, "%(title)S.%(ext)s")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	})

	t.Run("uses source's output override if present", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)
		updatedSource, _ := ta.App.SourcesUpdateSource(ta.Ctx, mediaItem.Source, store.Attrs{"output_path_template_override": "override.%(ext)s"}, store.KW{})

		path := ta.App.DownloadOptionBuilderBuildOutputPathForSource(ta.Ctx, updatedSource)

		expected := filepath.Join(ta.Config.MediaDirectory, "override.%(ext)s")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingConfigFileOptions(t *testing.T) {
	newConfigMediaItem := func(t *testing.T, ta *apptest.TestApp) *store.MediaItem {
		t.Helper()
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{OutputPathTemplate: store.Ptr("{{ title }}.%(ext)s")})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{"source_id": source.ID})
		mediaItem, err := ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("PreloadMediaItemFull: %v", err)
		}
		return mediaItem
	}

	t.Run("includes base config file if it's present", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := newConfigMediaItem(t, ta)
		baseDir := filepath.Join(ta.Config.ExtrasDirectory, "yt-dlp-configs")
		configPath := filepath.Join(baseDir, "base-config.txt")
		if err := fsutil.WriteFileAll(configPath, "base config"); err != nil {
			t.Fatalf("write config: %v", err)
		}

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" && kv.Value == configPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected config_locations option %q in %v", configPath, res)
		}
	})

	t.Run("includes media profile config file if it's present", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := newConfigMediaItem(t, ta)
		baseDir := filepath.Join(ta.Config.ExtrasDirectory, "yt-dlp-configs")
		configPath := filepath.Join(baseDir, fmt.Sprintf("media-profile-%d-config.txt", mediaItem.Source.MediaProfileID))
		if err := fsutil.WriteFileAll(configPath, "profile config"); err != nil {
			t.Fatalf("write config: %v", err)
		}

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" && kv.Value == configPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected config_locations option %q in %v", configPath, res)
		}
	})

	t.Run("includes source config file if it's present", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := newConfigMediaItem(t, ta)
		baseDir := filepath.Join(ta.Config.ExtrasDirectory, "yt-dlp-configs")
		configPath := filepath.Join(baseDir, fmt.Sprintf("source-%d-config.txt", mediaItem.SourceID))
		if err := fsutil.WriteFileAll(configPath, "source config"); err != nil {
			t.Fatalf("write config: %v", err)
		}

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" && kv.Value == configPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected config_locations option %q in %v", configPath, res)
		}
	})

	t.Run("includes media item config file if it's present", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := newConfigMediaItem(t, ta)
		baseDir := filepath.Join(ta.Config.ExtrasDirectory, "yt-dlp-configs")
		configPath := filepath.Join(baseDir, fmt.Sprintf("media-item-%d-config.txt", mediaItem.ID))
		if err := fsutil.WriteFileAll(configPath, "media item config"); err != nil {
			t.Fatalf("write config: %v", err)
		}

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" && kv.Value == configPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected config_locations option %q in %v", configPath, res)
		}
	})

	t.Run("does not include config file options if they are not present", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := newConfigMediaItem(t, ta)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" {
				found = true
				break
			}
		}
		if found {
			t.Errorf("config_locations should not be present")
		}
	})

	t.Run("does not return a config file if it's blank", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := newConfigMediaItem(t, ta)
		baseDir := filepath.Join(ta.Config.ExtrasDirectory, "yt-dlp-configs")
		configPath := filepath.Join(baseDir, "base-config.txt")
		if err := fsutil.WriteFileAll(configPath, " \n \n "); err != nil {
			t.Fatalf("write config: %v", err)
		}

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" {
				found = true
				break
			}
		}
		if found {
			t.Errorf("config_locations should not be present for blank file")
		}
	})

	t.Run("returns config files in order of precedence", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := newConfigMediaItem(t, ta)
		baseDir := filepath.Join(ta.Config.ExtrasDirectory, "yt-dlp-configs")

		baseFilepath := filepath.Join(baseDir, "base-config.txt")
		sourceFilepath := filepath.Join(baseDir, fmt.Sprintf("source-%d-config.txt", mediaItem.SourceID))
		mediaItemFilepath := filepath.Join(baseDir, fmt.Sprintf("media-item-%d-config.txt", mediaItem.ID))
		mediaProfileFilepath := filepath.Join(baseDir, fmt.Sprintf("media-profile-%d-config.txt", mediaItem.Source.MediaProfileID))

		for _, p := range []string{baseFilepath, sourceFilepath, mediaItemFilepath, mediaProfileFilepath} {
			if err := fsutil.WriteFileAll(p, "config"); err != nil {
				t.Fatalf("write config: %v", err)
			}
		}

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, store.KW{})

		var gotOrder []string
		for _, kv := range res {
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
	t.Run("builds quality options for a media item", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		options := ta.App.DownloadOptionBuilderBuildQualityOptionsForMediaItem(ta.Ctx, mediaItem)

		hasFormatSort := false
		hasRemuxVideo := false
		for _, kv := range options {
			if kv.Key == "format_sort" && kv.Value == "res:1080,+codec:avc:m4a" {
				hasFormatSort = true
			}
			if kv.Key == "remux_video" && kv.Value == "mp4" {
				hasRemuxVideo = true
			}
		}
		if !hasFormatSort || !hasRemuxVideo {
			t.Errorf("expected quality options in result")
		}
	})

	t.Run("builds quality options for a source", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.Attrs{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)
		source := mediaItem.Source

		options := ta.App.DownloadOptionBuilderBuildQualityOptionsForSource(ta.Ctx, source)

		hasFormatSort := false
		hasRemuxVideo := false
		for _, kv := range options {
			if kv.Key == "format_sort" && kv.Value == "res:1080,+codec:avc:m4a" {
				hasFormatSort = true
			}
			if kv.Key == "remux_video" && kv.Value == "mp4" {
				hasRemuxVideo = true
			}
		}
		if !hasFormatSort || !hasRemuxVideo {
			t.Errorf("expected quality options in result")
		}
	})
}
