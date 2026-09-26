package core

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

// MetadataFileHelpersMetadataDirectoryFor/1
func (a *App) MetadataFileHelpersMetadataDirectoryFor(ctx context.Context, databaseRecord any) (string, error) {
	panic("unported: Pinchflat.Metadata.MetadataFileHelpers.metadata_directory_for/1")
}

// MetadataFileHelpersCompressAndStoreMetadataFor/2
func (a *App) MetadataFileHelpersCompressAndStoreMetadataFor(ctx context.Context, databaseRecord any, metadataMap map[string]any) (string, error) {
	panic("unported: Pinchflat.Metadata.MetadataFileHelpers.compress_and_store_metadata_for/2")
}

// MetadataFileHelpersReadCompressedMetadata/1
func (a *App) MetadataFileHelpersReadCompressedMetadata(ctx context.Context, filepath string) (map[string]any, error) {
	panic("unported: Pinchflat.Metadata.MetadataFileHelpers.read_compressed_metadata/1")
}

// MetadataFileHelpersDownloadAndStoreThumbnailFor/1
func (a *App) MetadataFileHelpersDownloadAndStoreThumbnailFor(ctx context.Context, mediaItemWithPreloads *MediaItem) (*string, error) {
	panic("unported: Pinchflat.Metadata.MetadataFileHelpers.download_and_store_thumbnail_for/1")
}

// MetadataFileHelpersParseUploadDate/1
func MetadataFileHelpersParseUploadDate(uploadDate string) (time.Time, error) {
	if len(uploadDate) < 8 {
		return time.Time{}, fmt.Errorf("Invalid upload date: %s", uploadDate)
	}

	year := uploadDate[0:4]
	month := uploadDate[4:6]
	day := uploadDate[6:8]

	dt, err := time.Parse("2006-01-02T15:04:05Z", fmt.Sprintf("%s-%s-%sT00:00:00Z", year, month, day))
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid upload date: %s", uploadDate)
	}

	return dt, nil
}

// MetadataFileHelpersSeriesDirectoryFromMediaFilepath/1
func MetadataFileHelpersSeriesDirectoryFromMediaFilepath(mediaFilepath string) (string, error) {
	panic("unported: Pinchflat.Metadata.MetadataFileHelpers.series_directory_from_media_filepath/1")
}

// MetadataFileHelpersSeasonAndEpisodeFromMediaFilepath/1
func MetadataFileHelpersSeasonAndEpisodeFromMediaFilepath(mediaFilepath string) (string, string, error) {
	// matches s + 1 or more digits + e + 1 or more digits (case-insensitive)
	seasonEpisodeRegex := regexp.MustCompile(`(?i)s(\d+)e(\d+)`)

	matches := seasonEpisodeRegex.FindStringSubmatch(mediaFilepath)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("indeterminable")
	}

	return matches[1], matches[2], nil
}
