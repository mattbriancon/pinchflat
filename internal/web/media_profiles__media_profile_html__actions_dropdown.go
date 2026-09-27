package web

// Plain-Go half of media_profile_html/actions_dropdown.html.heex: building
// the "Copy JSON" Alpine click handler. The Elixir source calls
// Jason.Formatter.pretty_print(Jason.encode!(@media_profile)), whose custom
// Jason.Encoder (lib/pinchflat/profiles/media_profile.ex) drops
// __meta__/__struct__/sources. copyWithCallbacks is provided by
// web/assets/js/alpine_helpers.js.

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// mediaProfilesCopyJSON is Jason.Formatter.pretty_print(Jason.encode!(...)),
// using the MediaProfile Jason.Encoder port in core (json_encoders.go) and,
// like Jason, no HTML escaping.
func mediaProfilesCopyJSON(p *core.MediaProfile) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		return "{}"
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// mediaProfilesCopyWithCallbacksJS builds the Alpine x-on:click body.
func mediaProfilesCopyWithCallbacksJS(p *core.MediaProfile) string {
	return "\n    copyWithCallbacks(\n" +
		"      String.raw`" + mediaProfilesCopyJSON(p) + "`,\n" +
		"      () => copied = true, \n" +
		"      () => copied = false\n" +
		"    )\n  "
}
