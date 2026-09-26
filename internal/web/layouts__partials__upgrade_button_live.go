package web

// Port of lib/pinchflat_web/components/layouts/partials/upgrade_button_live.ex.
//
// PinchflatWeb.UpgradeButtonLive is a LiveView with no route in the Go port
// (router.go's LiveView table lists it as "client-side Alpine only; no
// route"), so `check_matching_text`'s Settings.set(pro_enabled: true) has
// nowhere to call: unlocking only flips the client-side `proEnabled` Alpine
// flag that gates the upgrade modal (root.html.heex), matching the comment
// on the modal's button ("This shouldn't happen! ... setTimeout ...").
// pro_enabled is never persisted server-side by this control in Phase 1;
// see the report.

// upgradeButtonLiveRequiredPhrase is `normalized_text == "got it"`.
const upgradeButtonLiveRequiredPhrase = "got it"
