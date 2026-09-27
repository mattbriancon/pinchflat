package core

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

// DownloadOptionBuilder builds options for yt-dlp based on media profile settings.

// DownloadOptionBuilderBuild/2
func (a *App) DownloadOptionBuilderBuild(ctx context.Context, mediaItem *MediaItem, overrideOpts KW) (KW, error) {
	mediaProfile := mediaItem.Source.MediaProfile

	builtOptions := downloadOptionBuilderDefaultOptions(overrideOpts)
	builtOptions = append(builtOptions, downloadOptionBuilderSubtitleOptions(mediaProfile)...)
	builtOptions = append(builtOptions, downloadOptionBuilderThumbnailOptions(ctx, a, mediaItem)...)
	builtOptions = append(builtOptions, downloadOptionBuilderMetadataOptions(mediaProfile)...)
	builtOptions = append(builtOptions, a.QualityOptionBuilderBuild(ctx, mediaProfile)...)
	builtOptions = append(builtOptions, downloadOptionBuilderSponsorblockOptions(mediaProfile)...)
	builtOptions = append(builtOptions, downloadOptionBuilderOutputOptions(ctx, a, mediaItem)...)
	builtOptions = append(builtOptions, a.downloadOptionBuilderConfigFileOptions(ctx, mediaItem)...)

	return builtOptions, nil
}

// DownloadOptionBuilderBuildOutputPathForSource/1
func (a *App) DownloadOptionBuilderBuildOutputPathForSource(ctx context.Context, source *Source) string {
	return a.DownloadOptionBuilderBuildOutputPathForMediaItem(ctx, &MediaItem{Source: source})
}

// DownloadOptionBuilderBuildOutputPathForMediaItem/1
func (a *App) DownloadOptionBuilderBuildOutputPathForMediaItem(ctx context.Context, mediaItem *MediaItem) string {
	outputPathTemplate := a.SourcesOutputPathTemplate(ctx, mediaItem.Source)
	return downloadOptionBuilderBuildOutputPath(ctx, a, outputPathTemplate, mediaItem)
}

// DownloadOptionBuilderBuildQualityOptionsForSource/1
func (a *App) DownloadOptionBuilderBuildQualityOptionsForSource(ctx context.Context, source *Source) KW {
	return a.DownloadOptionBuilderBuildQualityOptionsForMediaItem(ctx, &MediaItem{Source: source})
}

// DownloadOptionBuilderBuildQualityOptionsForMediaItem/1
func (a *App) DownloadOptionBuilderBuildQualityOptionsForMediaItem(ctx context.Context, mediaItem *MediaItem) KW {
	mediaProfile := mediaItem.Source.MediaProfile
	return a.QualityOptionBuilderBuild(ctx, mediaProfile)
}

// --- Private helpers ---

func downloadOptionBuilderDefaultOptions(overrideOpts KW) KW {
	overwriteBehaviour := overrideOpts.GetOr("overwrite_behaviour", "force_overwrites")

	return KW{
		Flag("no_progress"),
		KV{Key: overwriteBehaviour.(string), Flag: true},
		// This makes the date metadata conform to what jellyfin expects
		Opt("parse_metadata", "%(upload_date>%Y-%m-%d)s:(?P<meta_date>.+)"),
	}
}

func downloadOptionBuilderSubtitleOptions(mediaProfile *MediaProfile) KW {
	var result KW

	// Check download_subs
	if mediaProfile.DownloadSubs {
		result = append(result, Flag("write_subs"), Opt("convert_subs", "srt"))
	}

	// Check download_auto_subs with download_subs or embed_subs
	if mediaProfile.DownloadAutoSubs && (mediaProfile.DownloadSubs || mediaProfile.EmbedSubs) {
		result = append(result, Flag("write_auto_subs"))
	}

	// Check embed_subs (but not if preferred_resolution is audio)
	if mediaProfile.EmbedSubs && mediaProfile.PreferredResolution != MediaProfilePreferredResolutionAudio {
		result = append(result, Flag("embed_subs"))
	}

	// Check sub_langs with download_subs or embed_subs
	if mediaProfile.SubLangs != "" && (mediaProfile.DownloadSubs || mediaProfile.EmbedSubs) {
		result = append(result, Opt("sub_langs", mediaProfile.SubLangs))
	}

	return result
}

func downloadOptionBuilderThumbnailOptions(ctx context.Context, a *App, mediaItem *MediaItem) KW {
	mediaProfile := mediaItem.Source.MediaProfile
	var result KW

	if mediaProfile.DownloadThumbnail {
		thumbnailSaveLocation := downloadOptionBuilderDetermineThumbnailLocation(ctx, a, mediaItem)
		result = append(result, Flag("write_thumbnail"), Opt("convert_thumbnail", "jpg"), Opt("output", "thumbnail:"+thumbnailSaveLocation))
	}

	if mediaProfile.EmbedThumbnail {
		result = append(result, Flag("embed_thumbnail"), Opt("convert_thumbnail", "jpg"))
	}

	return result
}

func downloadOptionBuilderMetadataOptions(mediaProfile *MediaProfile) KW {
	var result KW

	if mediaProfile.DownloadMetadata {
		result = append(result, Flag("write_info_json"), Flag("clean_info_json"))
	}

	if mediaProfile.EmbedMetadata {
		result = append(result, Flag("embed_metadata"))
	}

	return result
}

func downloadOptionBuilderSponsorblockOptions(mediaProfile *MediaProfile) KW {
	categories := mediaProfile.SponsorblockCategories.V
	if mediaProfile.SponsorblockBehaviour == nil || len(categories) == 0 {
		return KW{}
	}

	behaviour := *mediaProfile.SponsorblockBehaviour
	categoryStr := strings.Join(categories, ",")

	switch behaviour {
	case MediaProfileSponsorblockBehaviourRemove:
		return KW{Opt("sponsorblock_remove", categoryStr)}
	case MediaProfileSponsorblockBehaviourMark:
		return KW{Opt("sponsorblock_mark", categoryStr)}
	}

	return KW{}
}

func downloadOptionBuilderOutputOptions(ctx context.Context, a *App, mediaItem *MediaItem) KW {
	outputPath := a.DownloadOptionBuilderBuildOutputPathForMediaItem(ctx, mediaItem)
	return KW{Opt("output", outputPath)}
}

func downloadOptionBuilderBuildOutputPath(ctx context.Context, a *App, templateString string, mediaItem *MediaItem) string {
	additionalOptionsMap := downloadOptionBuilderOutputOptionsMap(mediaItem)
	outputPath, _ := OutputPathBuilderBuild(templateString, additionalOptionsMap)
	return filepath.Join(a.Config.MediaDirectory, outputPath)
}

func downloadOptionBuilderOutputOptionsMap(mediaItem *MediaItem) map[string]string {
	source := mediaItem.Source

	return map[string]string{
		"media_item_id":           strconv.FormatInt(mediaItem.ID, 10),
		"source_id":               strconv.FormatInt(source.ID, 10),
		"media_profile_id":        strconv.FormatInt(source.MediaProfileID, 10),
		"source_custom_name":      source.CustomName,
		"source_collection_id":    source.CollectionID,
		"source_collection_name":  source.CollectionName,
		"source_collection_type":  string(source.CollectionType),
		"media_playlist_index":    padInt(mediaItem.PlaylistIndex, 2, "0"),
		"media_upload_date_index": padInt(mediaItem.UploadDateIndex, 2, "0"),
	}
}

// Inserts "-thumb" before the file extension
func downloadOptionBuilderDetermineThumbnailLocation(ctx context.Context, a *App, mediaItem *MediaItem) string {
	outputPathTemplate := a.SourcesOutputPathTemplate(ctx, mediaItem.Source)

	// Split by dots, keeping the separators
	parts := strings.Split(outputPathTemplate, ".")

	// Insert "-thumb" before the last extension
	if len(parts) > 1 {
		lastIdx := len(parts) - 1
		parts[lastIdx] = "-thumb." + parts[lastIdx]
	} else {
		parts = append(parts, "-thumb")
	}

	modifiedTemplate := strings.Join(parts, "")

	return downloadOptionBuilderBuildOutputPath(ctx, a, modifiedTemplate, mediaItem)
}

func (a *App) downloadOptionBuilderConfigFileOptions(ctx context.Context, mediaItem *MediaItem) KW {
	baseDir := filepath.Join(a.Config.ExtrasDirectory, "yt-dlp-configs")

	filenames := []string{
		fmt.Sprintf("media-item-%d-config.txt", mediaItem.ID),
		fmt.Sprintf("source-%d-config.txt", mediaItem.SourceID),
		fmt.Sprintf("media-profile-%d-config.txt", mediaItem.Source.MediaProfileID),
		"base-config.txt",
	}

	var configFilepaths []string
	for _, filename := range filenames {
		filepath := filepath.Join(baseDir, filename)
		if fsutil.ExistsAndNonEmpty(filepath) {
			configFilepaths = append(configFilepaths, filepath)
		}
	}

	var result KW
	// Reverse order to get the right precedence
	for i := len(configFilepaths) - 1; i >= 0; i-- {
		result = append(result, Opt("config_locations", configFilepaths[i]))
	}

	return result
}

func padInt(integer int, count int, padding string) string {
	s := strconv.Itoa(integer)
	for len(s) < count {
		s = padding + s
	}
	return s
}
