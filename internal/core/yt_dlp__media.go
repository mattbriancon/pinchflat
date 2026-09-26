package core

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
