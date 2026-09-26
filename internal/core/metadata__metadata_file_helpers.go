package core

import (
	"context"
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
	panic("unported: Pinchflat.Metadata.MetadataFileHelpers.parse_upload_date/1")
}

// MetadataFileHelpersSeriesDirectoryFromMediaFilepath/1
func MetadataFileHelpersSeriesDirectoryFromMediaFilepath(mediaFilepath string) (string, error) {
	panic("unported: Pinchflat.Metadata.MetadataFileHelpers.series_directory_from_media_filepath/1")
}

// MetadataFileHelpersSeasonAndEpisodeFromMediaFilepath/1
func MetadataFileHelpersSeasonAndEpisodeFromMediaFilepath(mediaFilepath string) (string, string, error) {
	panic("unported: Pinchflat.Metadata.MetadataFileHelpers.season_and_episode_from_media_filepath/1")
}
