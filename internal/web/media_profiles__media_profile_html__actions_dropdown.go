package web

import "github.com/mattbriancon/pinchflat/internal/store"

// Plain-Go half of media_profile_html/actions_dropdown.html.heex: building
// the "Copy JSON" Alpine click handler. The Elixir source calls
// Jason.Formatter.pretty_print(Jason.encode!(@media_profile)), whose custom
// Jason.Encoder (lib/pinchflat/profiles/media_profile.ex) drops
// __meta__/__struct__/sources. copyWithCallbacks is provided by
// web/assets/js/alpine_helpers.js.

// mediaProfilesCopyWithCallbacksJS builds the Alpine x-on:click body.
func mediaProfilesCopyWithCallbacksJS(p *store.MediaProfile) string {
	return "\n    copyWithCallbacks(\n" +
		"      String.raw`" + prettyJSON(p) + "`,\n" +
		"      () => copied = true, \n" +
		"      () => copied = false\n" +
		"    )\n  "
}
