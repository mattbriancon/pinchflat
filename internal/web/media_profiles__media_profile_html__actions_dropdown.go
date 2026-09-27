package web

// Plain-Go half of media_profile_html/actions_dropdown.html.heex: building
// the "Copy JSON" Alpine click handler. The Elixir source calls
// Jason.Formatter.pretty_print(Jason.encode!(@media_profile)), whose custom
// Jason.Encoder (lib/pinchflat/profiles/media_profile.ex) drops
// __meta__/__struct__/sources. copyWithCallbacks is provided by
// web/assets/js/alpine_helpers.js.

import (
	"encoding/json"
	"reflect"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// mediaProfilesJSONFields is MediaProfile.json_exluded_fields/0, applied in
// reverse: every db column except "sources" (the association).
func mediaProfilesJSONFields(p *core.MediaProfile) map[string]any {
	out := map[string]any{}
	v := reflect.ValueOf(p).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		out[tag] = v.Field(i).Interface()
	}
	return out
}

// mediaProfilesCopyJSON is Jason.Formatter.pretty_print(Jason.encode!(...)).
func mediaProfilesCopyJSON(p *core.MediaProfile) string {
	b, err := json.MarshalIndent(mediaProfilesJSONFields(p), "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// mediaProfilesCopyWithCallbacksJS builds the Alpine x-on:click body.
func mediaProfilesCopyWithCallbacksJS(p *core.MediaProfile) string {
	return "\n    copyWithCallbacks(\n" +
		"      String.raw`" + mediaProfilesCopyJSON(p) + "`,\n" +
		"      () => copied = true, \n" +
		"      () => copied = false\n" +
		"    )\n  "
}
