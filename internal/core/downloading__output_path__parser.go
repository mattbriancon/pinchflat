package core

import (
	"fmt"
	"regexp"
	"strings"
)

// OutputPathParser parses liquid-ish-style strings into rendered strings.

// OutputPathParserFetcher is a function type for custom value fetching in parse templates.
type OutputPathParserFetcher func(identifier string, variables map[string]string) string

// parsedElement represents an element in the parsed template
type parsedElement struct {
	typ   string // "text" or "interpolation"
	value string // the actual value or identifier
}

// OutputPathParserParse/3
func OutputPathParserParse(s string, variables map[string]string, valueFetchFn OutputPathParserFetcher) (string, error) {
	parsed, err := outputPathParseDoParse(s)
	if err != nil {
		return "", err
	}

	return outputPathParserBuildString(parsed, variables, valueFetchFn), nil
}

// doParse is the internal parser function that extracts elements
func outputPathParseDoParse(s string) ([]parsedElement, error) {
	// Pattern: {{ optional_whitespace identifier optional_whitespace }}
	// identifier: [a-zA-Z_][a-zA-Z0-9_]*
	pattern := `\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`
	re := regexp.MustCompile(pattern)

	var result []parsedElement
	lastPos := 0

	for _, match := range re.FindAllStringSubmatchIndex(s, -1) {
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

	// Check if we have any interpolations that failed to parse
	// (i.e., {{ but not properly closed, or invalid identifiers)
	if hasUnparsedInterpolations(s, re) {
		return nil, fmt.Errorf("expected end of string")
	}

	return result, nil
}

// hasUnparsedInterpolations checks if there are any unparsed {{ ... }} patterns
func hasUnparsedInterpolations(s string, validPattern *regexp.Regexp) bool {
	// Check if there are any {{ that weren't matched by the valid pattern
	bracePattern := regexp.MustCompile(`\{\{`)
	allMatches := bracePattern.FindAllStringIndex(s, -1)
	validMatches := validPattern.FindAllStringIndex(s, -1)

	// Count {{ occurrences
	openCount := len(allMatches)
	// Count valid interpolations
	validCount := len(validMatches)

	return openCount != validCount
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

// OutputPathParserDefaultFetcher/2
func OutputPathParserDefaultFetcher(identifier string, variables map[string]string) string {
	if value, ok := variables[identifier]; ok {
		return value
	}
	return ""
}
