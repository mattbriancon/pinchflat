package web

import (
	"net/http"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// Port of lib/pinchflat_web/controllers/settings/setting_html/apprise_server_live.ex (LiveView -> htmx).

// AppriseServerLiveSendTest: handles POST /_live/settings/apprise_test
func (s *Server) AppriseServerLiveSendTest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the apprise server value from the form
	settingParams := ParseForm(r, "setting")

	value := ""
	if v, ok := settingParams["apprise_server"]; ok {
		if str, ok := v.(string); ok {
			value = str
		}
	}

	// Send test notification using the Apprise runner
	// The test message is: title: "Pinchflat Test", body: "This is a test message from Pinchflat"
	_ = s.App.Apprise.Run(ctx, []string{value}, core.KW{
		core.Opt("title", "Pinchflat Test"),
		core.Opt("body", "This is a test message from Pinchflat"),
	})

	// Return the button with updated icon (checkmark) - it will revert after 4 seconds
	s.RenderFragment(w, r, http.StatusOK, AppriseTestButtonResult("hero-check", "Sent!"))
}
