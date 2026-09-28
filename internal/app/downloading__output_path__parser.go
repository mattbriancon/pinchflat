package app

import (
	"fmt"
	"regexp"
	"strings"
)

// OutputPathParser parses liquid-ish-style strings into rendered strings.

// OutputPathParserParse renders s, looking each {{ identifier }} up with
// valueFetchFn.
func OutputPathParserParse(s string, variables map[string]string, valueFetchFn OutputPathParserFetcher) (string, error) {
	parsed, err := outputPathParseDoParse(s)
	if err != nil {
		return "", err
	}
	return outputPathParserBuildString(parsed, variables, valueFetchFn), nil
}

// OutputPathParserFetcher is a function type for custom value fetching in parse templates.
type OutputPathParserFetcher func(identifier string, variables map[string]string) string

// parsedElement represents an element in the parsed template
type parsedElement struct {
	typ   string // "text" or "interpolation"
	value string // the actual value or identifier
}

// outputPathInterpolation matches {{ identifier }}, allowing spaces inside
// the braces.
var outputPathInterpolation = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

// doParse is the internal parser function that extracts elements
func outputPathParseDoParse(s string) ([]parsedElement, error) {

	var result []parsedElement
	lastPos := 0

	matches := outputPathInterpolation.FindAllStringSubmatchIndex(s, -1)
	for _, match := range matches {
		// match[0:2] is the full match, match[2:4] is the first capture group (identifier)
		fullStart := match[0]
		fullEnd := match[1]
		identStart := match[2]
		identEnd := match[3]

		// Add text before this interpolation (if any)
		if fullStart > lastPos {
			text := s[lastPos:fullStart]
			result = append(result, parsedElement{typ: "text", value: text})
		}

		// Add the interpolation
		identifier := s[identStart:identEnd]
		result = append(result, parsedElement{typ: "interpolation", value: identifier})

		lastPos = fullEnd
	}

	// Add any remaining text
	if lastPos < len(s) {
		result = append(result, parsedElement{typ: "text", value: s[lastPos:]})
	}

	// Every {{ must have been part of a valid interpolation (not left
	// unclosed or wrapping an invalid identifier).
	if strings.Count(s, "{{") != len(matches) {
		return nil, fmt.Errorf("expected end of string")
	}

	return result, nil
}

// outputPathParserBuildString builds the final string from parsed elements
func outputPathParserBuildString(parsed []parsedElement, variables map[string]string, valueFetchFn OutputPathParserFetcher) string {
	var result strings.Builder

	for _, element := range parsed {
		if element.typ == "text" {
			result.WriteString(element.value)
		} else if element.typ == "interpolation" {
			// Call the fetcher function with the identifier
			value := valueFetchFn(element.value, variables)
			result.WriteString(value)
		}
	}

	return result.String()
}
