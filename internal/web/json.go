package web

import (
	"bytes"
	"encoding/json"
	"strings"
)

// prettyJSON is Jason.Formatter.pretty_print(Jason.encode!(v)): the record's
// core MarshalJSON (the Jason.Encoder ports), two-space indented, without
// HTML escaping (Jason doesn't escape <, > or &).
func prettyJSON(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "{}"
	}
	return strings.TrimSuffix(buf.String(), "\n")
}
