package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestDownloadOptionBuilder_Build_WhenTestingOutputOptions(t *testing.T) {
	t.Run("it generates an expanded output path based on the given template", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"output_path_template": "{{ title }}.%(ext)s"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == "/tmp/test/media/%(title)S.%(ext)s" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected output option in result")
		}
	})

	t.Run("it respects custom output path options", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"output_path_template": "{{ source_custom_name }}.%(ext)s"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		expected := "/tmp/test/media/" + mediaItem.Source.CustomName + ".%(ext)s"
		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected output option with custom name in result")
		}
	})

	t.Run("respects custom media_item-related output path options", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"output_path_template": "{{ media_upload_date_index }}.%(ext)s"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		expected := "/tmp/test/media/99.%(ext)s"
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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"output_path_template": "{{ title }}.%(ext)s"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)
		ta.App.SourcesUpdateSource(ta.Ctx, mediaItem.Source, core.Attrs{"output_path_template_override": "override.%(ext)s"}, core.KW{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		expected := "/tmp/test/media/override.%(ext)s"
		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected output option with override in result")
		}
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingDefaultOptions(t *testing.T) {
	t.Run("it includes default options", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"output_path_template": "{{ title }}.%(ext)s"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"output_path_template": "{{ title }}.%(ext)s"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{core.Opt("overwrite_behaviour", "no_force_overwrites")})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_subs": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_subs": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile1 := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_subs": true, "download_auto_subs": true})
		mediaProfile2 := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_subs": true, "download_auto_subs": true})
		source1 := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile1.ID})
		source2 := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile2.ID})
		mediaItem1 := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source1.ID})
		mediaItem2 := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source2.ID})
		mediaItem1, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem1)
		mediaItem2, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem2)

		res1, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem1, core.KW{})
		res2, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem2, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_subs": false, "embed_subs": false, "download_auto_subs": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_subs": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_subs": true, "preferred_resolution": "audio"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_subs": true, "sub_langs": "en"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_subs": true, "sub_langs": "en"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_subs": false, "download_subs": false, "sub_langs": "en"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_thumbnail": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_thumbnail": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == "thumbnail:/tmp/test/media/%(title)S-thumb.%(ext)s" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected thumbnail output option in result")
		}
	})

	t.Run("appends -thumb to source's output path override, if present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_thumbnail": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		ta.App.SourcesUpdateSource(ta.Ctx, source, core.Attrs{"output_path_template_override": "override.%(ext)s"}, core.KW{})
		source, _ = ta.App.PreloadSourceMediaProfile(ta.Ctx, source)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "output" && kv.Value == "thumbnail:/tmp/test/media/override-thumb.%(ext)s" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected thumbnail output option with override in result")
		}
	})

	t.Run("converts thumbnail to jpg when download_thumbnail is true", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_thumbnail": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_thumbnail": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_thumbnail": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_thumbnail": false, "download_thumbnail": false})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_metadata": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_metadata": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"embed_metadata": false, "download_metadata": false})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"preferred_resolution": "audio"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{
			"sponsorblock_behaviour":  "remove",
			"sponsorblock_categories": []string{"sponsor", "intro"},
		})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{
			"sponsorblock_behaviour":  "mark",
			"sponsorblock_categories": []string{"sponsor", "intro"},
		})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{
			"sponsorblock_behaviour":  "remove",
			"sponsorblock_categories": []string{},
		})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{"sponsorblock_behaviour": "disabled"})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		path := ta.App.DownloadOptionBuilderBuildOutputPathForMediaItem(ta.Ctx, mediaItem)

		if path != "/tmp/test/media/%(title)S.%(ext)s" {
			t.Errorf("expected /tmp/test/media/%%(title)S.%%(ext)s, got %s", path)
		}
	})

	t.Run("builds an output path for a source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		source := mediaItem.Source

		path := ta.App.DownloadOptionBuilderBuildOutputPathForSource(ta.Ctx, source)

		if path != "/tmp/test/media/%(title)S.%(ext)s" {
			t.Errorf("expected /tmp/test/media/%%(title)S.%%(ext)s, got %s", path)
		}
	})

	t.Run("uses source's output override if present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
		ta.App.SourcesUpdateSource(ta.Ctx, mediaItem.Source, core.Attrs{"output_path_template_override": "override.%(ext)s"}, core.KW{})

		path := ta.App.DownloadOptionBuilderBuildOutputPathForSource(ta.Ctx, mediaItem.Source)

		if path != "/tmp/test/media/override.%(ext)s" {
			t.Errorf("expected /tmp/test/media/override.%%(ext)s, got %s", path)
		}
	})
}

func TestDownloadOptionBuilder_Build_WhenTestingConfigFileOptions(t *testing.T) {
	t.Run("includes base config file if it's present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected config_locations option")
		}
	})

	t.Run("includes media profile config file if it's present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected config_locations option")
		}
	})

	t.Run("includes source config file if it's present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected config_locations option")
		}
	})

	t.Run("includes media item config file if it's present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		found := false
		for _, kv := range res {
			if kv.Key == "config_locations" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected config_locations option")
		}
	})

	t.Run("does not include config file options if they are not present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

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
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

		res, _ := ta.App.DownloadOptionBuilderBuild(ta.Ctx, mediaItem, core.KW{})

		if len(res) == 0 {
			t.Errorf("expected result")
		}
	})
}

func TestDownloadOptionBuilder_BuildQualityOptionsFor(t *testing.T) {
	t.Run("builds quality options for a media item", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})

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
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{})
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
