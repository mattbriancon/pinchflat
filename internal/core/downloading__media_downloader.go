package core

import "context"

// MediaDownloader is the integration layer for downloading media.

// MediaDownloaderResult is the non-error outcome of download_for_media_item/2:
// {:ok, media_item} (Recovered false) or {:recovered, media_item, message}.
type MediaDownloaderResult struct {
	MediaItem *MediaItem
	Recovered bool
	Message   string
}

// MediaDownloaderError is {:error, error_atom, message}. Reason is the atom
// name: "unsuitable_for_download", "download_failed", "unrecoverable" or "unknown".
type MediaDownloaderError struct {
	Reason  string
	Message string
}

func (e *MediaDownloaderError) Error() string { return e.Reason + ": " + e.Message }

// download_for_media_item/2
func (a *App) MediaDownloaderDownloadForMediaItem(ctx context.Context, mediaItem *MediaItem, overrideOpts KW) (*MediaDownloaderResult, error) {
	panic("unported: Pinchflat.Downloading.MediaDownloader.download_for_media_item/2")
}
