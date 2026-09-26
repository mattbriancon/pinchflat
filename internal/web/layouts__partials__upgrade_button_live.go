package web

// Port of lib/pinchflat_web/components/layouts/partials/upgrade_button_live.ex.
//
// The text box and button live in upgrade_modal.templ: Alpine enables the
// button once the text matches, and every edit is posted to /_live/upgrade,
// which is handle_event("check_matching_text", ...).

import (
	"net/http"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// upgradeButtonLiveRequiredPhrase is `normalized_text == "got it"`.
const upgradeButtonLiveRequiredPhrase = "got it"

// UpgradeButtonLiveCheckMatchingText sets pro_enabled once the phrase is typed.
func (s *Server) UpgradeButtonLiveCheckMatchingText(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	text := strings.ToLower(strings.TrimSpace(r.PostForm.Get("unlock-pro-textbox")))
	if text == upgradeButtonLiveRequiredPhrase {
		if _, err := s.App.SettingsSet(r.Context(), core.KW{core.Opt("pro_enabled", true)}); err != nil {
			s.Fail(w, r, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
