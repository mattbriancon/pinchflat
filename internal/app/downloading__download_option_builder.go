package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// DownloadOptionBuilder builds options for yt-dlp based on media profile settings.

// DownloadOptionBuilderBuild/2
func (a *App) DownloadOptionBuilderBuild(ctx context.Context, mediaItem *store.MediaItem, overrideOpts store.KW) (store.KW, error) {
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
func (a *App) DownloadOptionBuilderBuildOutputPathForSource(ctx context.Context, source *store.Source) string {
	return a.DownloadOptionBuilderBuildOutputPathForMediaItem(ctx, &store.MediaItem{Source: source})
}

// DownloadOptionBuilderBuildOutputPathForMediaItem/1
func (a *App) DownloadOptionBuilderBuildOutputPathForMediaItem(ctx context.Context, mediaItem *store.MediaItem) string {
	outputPathTemplate := a.SourcesOutputPathTemplate(ctx, mediaItem.Source)
	return downloadOptionBuilderBuildOutputPath(ctx, a, outputPathTemplate, mediaItem)
}

// DownloadOptionBuilderBuildQualityOptionsForSource/1
func (a *App) DownloadOptionBuilderBuildQualityOptionsForSource(ctx context.Context, source *store.Source) store.KW {
	return a.DownloadOptionBuilderBuildQualityOptionsForMediaItem(ctx, &store.MediaItem{Source: source})
}

// DownloadOptionBuilderBuildQualityOptionsForMediaItem/1
func (a *App) DownloadOptionBuilderBuildQualityOptionsForMediaItem(ctx context.Context, mediaItem *store.MediaItem) store.KW {
	mediaProfile := mediaItem.Source.MediaProfile
	return a.QualityOptionBuilderBuild(ctx, mediaProfile)
}

// --- Private helpers ---

func downloadOptionBuilderDefaultOptions(overrideOpts store.KW) store.KW {
	overwriteBehaviour := overrideOpts.GetOr("overwrite_behaviour", "force_overwrites")

	return store.KW{
		store.Flag("no_progress"),
		store.KV{Key: overwriteBehaviour.(string), Flag: true},
		// This makes the date metadata conform to what jellyfin expects
		store.Opt("parse_metadata", "%(upload_date>%Y-%m-%d)s:(?P<meta_date>.+)"),
	}
}

func downloadOptionBuilderSubtitleOptions(mediaProfile *store.MediaProfile) store.KW {
	var result store.KW

	// Check download_subs
	if mediaProfile.DownloadSubs {
		result = append(result, store.Flag("write_subs"), store.Opt("convert_subs", "srt"))
	}

	// Check download_auto_subs with download_subs or embed_subs
	if mediaProfile.DownloadAutoSubs && (mediaProfile.DownloadSubs || mediaProfile.EmbedSubs) {
		result = append(result, store.Flag("write_auto_subs"))
	}

	// Check embed_subs (but not if preferred_resolution is audio)
	if mediaProfile.EmbedSubs && mediaProfile.PreferredResolution != store.MediaProfilePreferredResolutionAudio {
		result = append(result, store.Flag("embed_subs"))
	}

	// Check sub_langs with download_subs or embed_subs
	if mediaProfile.SubLangs != "" && (mediaProfile.DownloadSubs || mediaProfile.EmbedSubs) {
		result = append(result, store.Opt("sub_langs", mediaProfile.SubLangs))
	}

	return result
}

func downloadOptionBuilderThumbnailOptions(ctx context.Context, a *App, mediaItem *store.MediaItem) store.KW {
	mediaProfile := mediaItem.Source.MediaProfile
	var result store.KW

	if mediaProfile.DownloadThumbnail {
		thumbnailSaveLocation := downloadOptionBuilderDetermineThumbnailLocation(ctx, a, mediaItem)
		result = append(result, store.Flag("write_thumbnail"), store.Opt("convert_thumbnail", "jpg"), store.Opt("output", "thumbnail:"+thumbnailSaveLocation))
	}

	if mediaProfile.EmbedThumbnail {
		result = append(result, store.Flag("embed_thumbnail"), store.Opt("convert_thumbnail", "jpg"))
	}

	return result
}

func downloadOptionBuilderMetadataOptions(mediaProfile *store.MediaProfile) store.KW {
	var result store.KW

	if mediaProfile.DownloadMetadata {
		result = append(result, store.Flag("write_info_json"), store.Flag("clean_info_json"))
	}

	if mediaProfile.EmbedMetadata {
		result = append(result, store.Flag("embed_metadata"))
	}

	return result
}

func downloadOptionBuilderSponsorblockOptions(mediaProfile *store.MediaProfile) store.KW {
	categories := mediaProfile.SponsorblockCategories.V
	if mediaProfile.SponsorblockBehaviour == nil || len(categories) == 0 {
		return store.KW{}
	}

	behaviour := *mediaProfile.SponsorblockBehaviour
	categoryStr := strings.Join(categories, ",")

	switch behaviour {
	case store.MediaProfileSponsorblockBehaviourRemove:
		return store.KW{store.Opt("sponsorblock_remove", categoryStr)}
	case store.MediaProfileSponsorblockBehaviourMark:
		return store.KW{store.Opt("sponsorblock_mark", categoryStr)}
	}

	return store.KW{}
}

func downloadOptionBuilderOutputOptions(ctx context.Context, a *App, mediaItem *store.MediaItem) store.KW {
	outputPath := a.DownloadOptionBuilderBuildOutputPathForMediaItem(ctx, mediaItem)
	return store.KW{store.Opt("output", outputPath)}
}

func downloadOptionBuilderBuildOutputPath(ctx context.Context, a *App, templateString string, mediaItem *store.MediaItem) string {
	additionalOptionsMap := downloadOptionBuilderOutputOptionsMap(mediaItem)
	outputPath, _ := OutputPathBuilderBuild(templateString, additionalOptionsMap)
	return filepath.Join(a.Config.MediaDirectory, outputPath)
}

func downloadOptionBuilderOutputOptionsMap(mediaItem *store.MediaItem) map[string]string {
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
func downloadOptionBuilderDetermineThumbnailLocation(ctx context.Context, a *App, mediaItem *store.MediaItem) string {
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

func (a *App) downloadOptionBuilderConfigFileOptions(ctx context.Context, mediaItem *store.MediaItem) store.KW {
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

	var result store.KW
	// Reverse order to get the right precedence
	for i := len(configFilepaths) - 1; i >= 0; i-- {
		result = append(result, store.Opt("config_locations", configFilepaths[i]))
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
