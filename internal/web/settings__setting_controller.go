package web

import (
	"net/http"
	"os"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// Port of lib/pinchflat_web/controllers/settings/setting_controller.ex.

// SettingControllerShow: show(conn, _params)
func (s *Server) SettingControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	setting, err := s.App.SettingsRecord(ctx)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	changeset := s.App.SettingsChangeSetting(ctx, setting, core.Attrs{})
	s.Render(w, r, http.StatusOK, LayoutApp, SettingHTMLShow(changeset))
}

// SettingControllerUpdate: update(conn, %{"setting" => setting_params})
func (s *Server) SettingControllerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	setting, err := s.App.SettingsRecord(ctx)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	settingParams := ParseForm(r, "setting")

	_, err = s.App.SettingsUpdateSetting(ctx, setting, settingParams)
	if err != nil {
		// Failed changeset re-renders with status 200
		cs, ok := core.AsChangesetError(err)
		if ok && cs != nil {
			s.Render(w, r, http.StatusOK, LayoutApp, SettingHTMLShow(cs))
			return
		}
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "Settings updated successfully.")
	s.Redirect(w, r, P(ctx, "/settings"))
}

// SettingControllerAppInfo: app_info(conn, _params)
func (s *Server) SettingControllerAppInfo(w http.ResponseWriter, r *http.Request) {
	s.Render(w, r, http.StatusOK, LayoutApp, SettingHTMLAppInfo())
}

// SettingControllerDownloadLogs: download_logs(conn, _params)
func (s *Server) SettingControllerDownloadLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logPath := s.App.Config.LogPath

	if logPath == "" {
		// No log path configured
		s.PutFlash(w, r, "error", "Log file couldn't be found")
		s.Redirect(w, r, P(ctx, "/app_info"))
		return
	}

	// Check if file exists
	if _, err := os.Stat(logPath); err != nil {
		s.PutFlash(w, r, "error", "Log file couldn't be found")
		s.Redirect(w, r, P(ctx, "/app_info"))
		return
	}

	// Open the file
	file, err := os.Open(logPath)
	if err != nil {
		s.PutFlash(w, r, "error", "Log file couldn't be found")
		s.Redirect(w, r, P(ctx, "/app_info"))
		return
	}
	defer file.Close()

	// Get file info for size/modtime
	stat, err := file.Stat()
	if err != nil {
		s.PutFlash(w, r, "error", "Log file couldn't be found")
		s.Redirect(w, r, P(ctx, "/app_info"))
		return
	}

	// Set download headers
	filename := "pinchflat-logs-" + time.Now().UTC().Format("2006-01-02") + ".txt"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	// Serve the file
	http.ServeContent(w, r, filename, stat.ModTime(), file)
}
