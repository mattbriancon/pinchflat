package web

import (
	"net/http"
	"os"
	"time"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// SettingControllerShow renders the settings page.
func (s *Server) SettingControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	setting, err := s.App.GetSettingsRecord(ctx)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	values := map[string]any{
		"onboarding":                       setting.Onboarding,
		"yt_dlp_version":                   setting.YtDlpVersion,
		"video_codec_preference":           setting.VideoCodecPreference,
		"audio_codec_preference":           setting.AudioCodecPreference,
		"youtube_api_key":                  setting.YoutubeAPIKey,
		"extractor_sleep_interval_seconds": setting.ExtractorSleepIntervalSeconds,
		"download_throughput_limit":        setting.DownloadThroughputLimit,
		"restrict_filenames":               setting.RestrictFilenames,
	}
	form := NewForm("setting", values, nil)
	s.Render(w, r, http.StatusOK, LayoutApp, SettingHTMLShowWithForm(form))
}

// SettingControllerUpdate updates the app settings.
func (s *Server) SettingControllerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	setting, err := s.App.GetSettingsRecord(ctx)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	_ = r.ParseForm()
	p, parseErrs := store.ParseSettingParams(r.PostForm)
	if len(parseErrs) > 0 {
		// Build Values map for form redisplay
		values := map[string]any{
			"onboarding":                       p.Onboarding,
			"yt_dlp_version":                   p.YtDlpVersion,
			"video_codec_preference":           p.VideoCodecPreference,
			"audio_codec_preference":           p.AudioCodecPreference,
			"youtube_api_key":                  p.YoutubeAPIKey,
			"extractor_sleep_interval_seconds": p.ExtractorSleepIntervalSeconds,
			"download_throughput_limit":        p.DownloadThroughputLimit,
			"restrict_filenames":               p.RestrictFilenames,
		}
		form := NewForm("setting", values, parseErrs)
		s.Render(w, r, http.StatusOK, LayoutApp, SettingHTMLShowWithForm(form))
		return
	}

	updated, errs, err := s.App.UpdateSetting(ctx, setting, p)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	if len(errs) > 0 {
		// Build Values map for form redisplay
		values := map[string]any{
			"onboarding":                       p.Onboarding,
			"yt_dlp_version":                   p.YtDlpVersion,
			"video_codec_preference":           p.VideoCodecPreference,
			"audio_codec_preference":           p.AudioCodecPreference,
			"youtube_api_key":                  p.YoutubeAPIKey,
			"extractor_sleep_interval_seconds": p.ExtractorSleepIntervalSeconds,
			"download_throughput_limit":        p.DownloadThroughputLimit,
			"restrict_filenames":               p.RestrictFilenames,
		}
		form := NewForm("setting", values, errs)
		s.Render(w, r, http.StatusOK, LayoutApp, SettingHTMLShowWithForm(form))
		return
	}

	// Update succeeded
	_ = updated
	s.PutFlash(w, r, "info", "Settings updated successfully.")
	s.Redirect(w, r, P(ctx, "/settings"))
}

// SettingControllerAppInfo renders the app info page.
func (s *Server) SettingControllerAppInfo(w http.ResponseWriter, r *http.Request) {
	s.Render(w, r, http.StatusOK, LayoutApp, SettingHTMLAppInfo())
}

// SettingControllerDownloadLogs streams the configured log file as a download.
func (s *Server) SettingControllerDownloadLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logPath := s.App.Config.LogPath
	file, stat, err := openLogFile(logPath)
	if err != nil {
		s.PutFlash(w, r, "error", "Log file couldn't be found")
		s.Redirect(w, r, P(ctx, "/app_info"))
		return
	}
	defer file.Close()

	filename := "pinchflat-logs-" + time.Now().UTC().Format("2006-01-02") + ".txt"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	http.ServeContent(w, r, filename, stat.ModTime(), file)
}

// openLogFile opens logPath for reading, failing if it's empty or missing.
func openLogFile(logPath string) (*os.File, os.FileInfo, error) {
	if logPath == "" {
		return nil, nil, os.ErrNotExist
	}
	file, err := os.Open(logPath)
	if err != nil {
		return nil, nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	return file, stat, nil
}
