package web

import (
	"net/http"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// Port of lib/pinchflat_web/controllers/settings/setting_html/apprise_server_live.ex (LiveView -> htmx).

// AppriseServerLiveSendTest: handle_event("send_apprise_test", ...) for
// POST /_live/settings/apprise_test. There is no persistent LiveView
// process, so the value that would have been kept in sync via the
// "apprise_server_changed" event is instead posted along with the click
// (the button's hx-include grabs the input's current value).
func (s *Server) AppriseServerLiveSendTest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	settingParams := ParseForm(r, "setting")
	value, _ := settingParams["apprise_server"].(string)

	// backend_runner().run([assigns.value], title: ..., body: ...)
	_ = s.App.Apprise.Run(ctx, []string{value}, core.KW{
		core.Opt("title", "Pinchflat Test"),
		core.Opt("body", "This is a test message from Pinchflat"),
	})

	// assign(socket, %{icon_name: "hero-check", tooltip: "Sent!"}) -- the
	// value itself is unchanged, we just re-render with the new icon.
	s.RenderFragment(w, r, http.StatusOK, SettingHTMLAppriseServerLiveFragment(value, "hero-check", "Sent!"))
}
