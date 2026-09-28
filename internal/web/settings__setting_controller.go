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

	changeset := s.App.ChangeSetting(ctx, setting, store.Attrs{})
	s.Render(w, r, http.StatusOK, LayoutApp, SettingHTMLShow(changeset))
}

// SettingControllerUpdate updates the app settings.
func (s *Server) SettingControllerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	setting, err := s.App.GetSettingsRecord(ctx)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	settingParams := ParseForm(r, "setting")

	_, err = s.App.UpdateSetting(ctx, setting, settingParams)
	if err != nil {
		// Failed changeset re-renders with status 200
		cs, ok := store.AsChangesetError(err)
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
