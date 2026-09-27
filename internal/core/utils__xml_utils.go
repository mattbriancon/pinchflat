package core

import (
	"fmt"
	"html"
)

// XmlUtilsSafe(value)
// Escapes invalid XML characters in a string.
func XmlUtilsSafe(value interface{}) string {
	var str string
	if value == nil {
		str = ""
	} else if s, ok := value.(string); ok {
		str = s
	} else {
		str = fmt.Sprintf("%v", value)
	}

	return html.EscapeString(str)
}
