package web

import (
	"net/http"
	"os"
	"strings"
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

// settingControllerUnlockProPhrase is `normalized_text == "got it"`.
const settingControllerUnlockProPhrase = "got it"

// SettingControllerUnlockPro is POST /settings/pro: the upgrade modal's
// "Unlock Pro" form. It sets pro_enabled once the typed phrase matches and
// redirects back to the page it came from either way (STRATEGY.md decision
// 4: a plain form POST, not an htmx fragment update; port of
// lib/pinchflat_web/components/layouts/partials/upgrade_button_live.ex's
// handle_event("check_matching_text", ...)).
func (s *Server) SettingControllerUnlockPro(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	text := strings.ToLower(strings.TrimSpace(r.PostForm.Get("unlock-pro-textbox")))
	if text == settingControllerUnlockProPhrase {
		if _, err := s.App.SettingsSet(r.Context(), core.KW{core.Opt("pro_enabled", true)}); err != nil {
			s.Fail(w, r, err)
			return
		}
	}
	s.redirectBack(w, r, P(r.Context(), "/"))
}
