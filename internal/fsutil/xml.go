package fsutil

import (
	"fmt"
	"html"
)

// XMLSafe escapes invalid XML characters in value (any type; nil becomes
// "").
func XMLSafe(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return html.EscapeString(v)
	default:
		return html.EscapeString(fmt.Sprintf("%v", v))
	}
}
