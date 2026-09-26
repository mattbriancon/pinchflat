package core

import "context"
import "time"

// YtDlpMedia represents a piece of media parsed from yt-dlp's output.
// Not a database table, just a struct used for parsing yt-dlp responses.
type YtDlpMedia struct {
	MediaID                string    `json:"media_id"`
	Title                  string    `json:"title"`
	Description            string    `json:"description"`
	OriginalURL            string    `json:"original_url"`
	Livestream             bool      `json:"livestream"`
	ShortFormContent       bool      `json:"short_form_content"`
	UploadedAt             time.Time `json:"uploaded_at"`
	DurationSeconds        int       `json:"duration_seconds"`
	PredictedMediaFilepath string    `json:"predicted_media_filepath"`
	PlaylistIndex          *int      `json:"playlist_index"`
}

// YtDlpMediaDownload/3
func (a *App) YtDlpMediaDownload(ctx context.Context, url string, commandOpts KW, addlOpts KW) (map[string]any, error) {
	panic("unported: Pinchflat.YtDlp.Media.download/3")
}

// YtDlpMediaGetDownloadableStatus/2
func (a *App) YtDlpMediaGetDownloadableStatus(ctx context.Context, url string, addlOpts KW) (string, error) {
	panic("unported: Pinchflat.YtDlp.Media.get_downloadable_status/2")
}

// YtDlpMediaDownloadThumbnail/3
func (a *App) YtDlpMediaDownloadThumbnail(ctx context.Context, url string, commandOpts KW, addlOpts KW) (string, error) {
	panic("unported: Pinchflat.YtDlp.Media.download_thumbnail/3")
}

// YtDlpMediaGetMediaAttributes/3
func (a *App) YtDlpMediaGetMediaAttributes(ctx context.Context, url string, commandOpts KW, addlOpts KW) (*YtDlpMedia, error) {
	panic("unported: Pinchflat.YtDlp.Media.get_media_attributes/3")
}

// YtDlpMediaIndexingOutputTemplate/0
func YtDlpMediaIndexingOutputTemplate() string {
	panic("unported: Pinchflat.YtDlp.Media.indexing_output_template/0")
}

// YtDlpMediaResponseToStruct/1
func YtDlpMediaResponseToStruct(response map[string]any) *YtDlpMedia {
	panic("unported: Pinchflat.YtDlp.Media.response_to_struct/1")
}

// YtDlpMediaShortFormContent/1
func YtDlpMediaShortFormContent(response map[string]any) bool {
	panic("unported: Pinchflat.YtDlp.Media.short_form_content?/1")
}

// YtDlpMediaParseUploadedAt/1
func YtDlpMediaParseUploadedAt(response map[string]any) *time.Time {
	panic("unported: Pinchflat.YtDlp.Media.parse_uploaded_at/1")
}

// YtDlpMediaParseDownloadableStatus/1
func YtDlpMediaParseDownloadableStatus(response map[string]any) (string, error) {
	panic("unported: Pinchflat.YtDlp.Media.parse_downloadable_status/1")
}
