package core

import "context"

// DownloadOptionBuilder builds options for yt-dlp based on media profile settings.

// DownloadOptionBuilderBuild/2
func (a *App) DownloadOptionBuilderBuild(ctx context.Context, mediaItem *MediaItem, overrideOpts KW) (KW, error) {
	panic("unported: Pinchflat.Downloading.DownloadOptionBuilder.build/2")
}

// DownloadOptionBuilderBuildOutputPathForSource/1
func (a *App) DownloadOptionBuilderBuildOutputPathForSource(ctx context.Context, source *Source) string {
	panic("unported: Pinchflat.Downloading.DownloadOptionBuilder.build_output_path_for/1")
}

// DownloadOptionBuilderBuildOutputPathForMediaItem/1
func (a *App) DownloadOptionBuilderBuildOutputPathForMediaItem(ctx context.Context, mediaItem *MediaItem) string {
	panic("unported: Pinchflat.Downloading.DownloadOptionBuilder.build_output_path_for/1")
}

// DownloadOptionBuilderBuildQualityOptionsForSource/1
func (a *App) DownloadOptionBuilderBuildQualityOptionsForSource(ctx context.Context, source *Source) KW {
	panic("unported: Pinchflat.Downloading.DownloadOptionBuilder.build_quality_options_for/1")
}

// DownloadOptionBuilderBuildQualityOptionsForMediaItem/1
func (a *App) DownloadOptionBuilderBuildQualityOptionsForMediaItem(ctx context.Context, mediaItem *MediaItem) KW {
	panic("unported: Pinchflat.Downloading.DownloadOptionBuilder.build_quality_options_for/1")
}
