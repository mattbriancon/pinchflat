package web

import "regexp"

type searchFragment struct {
	text        string
	highlighted bool
}

// splitSearchTerms splits a string on [PF_HIGHLIGHT] and [/PF_HIGHLIGHT] tags,
// tracking whether each segment is highlighted.
func splitSearchTerms(text string) []searchFragment {
	if text == "" {
		return []searchFragment{}
	}

	re := regexp.MustCompile(`\[PF_HIGHLIGHT\]|\[/PF_HIGHLIGHT\]`)
	parts := re.Split(text, -1)
	captures := re.FindAllString(text, -1)

	var result []searchFragment
	highlighted := false

	for i, part := range parts {
		if part != "" {
			result = append(result, searchFragment{
				text:        part,
				highlighted: highlighted,
			})
		}

		// Process the next capture if it exists
		if i < len(captures) {
			if captures[i] == "[PF_HIGHLIGHT]" {
				highlighted = true
			} else if captures[i] == "[/PF_HIGHLIGHT]" {
				highlighted = false
			}
		}
	}

	return result
}
