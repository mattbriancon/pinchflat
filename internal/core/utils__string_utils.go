package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// StringUtilsToKebabCase(string)
// Converts a string to kebab-case (e.g., "hello world" -> "hello-world").
func StringUtilsToKebabCase(str string) string {
	// Replace spaces and underscores with hyphens
	re := regexp.MustCompile(`[\s_]`)
	result := re.ReplaceAllString(str, "-")
	return strings.ToLower(result)
}

// StringUtilsRandomString(length)
// Returns a random string of the given length. Base 16 encoded, lower case.
// Default length is 32.
func StringUtilsRandomString(length int) string {
	// Generate random bytes - we need half the length since hex encoding doubles it
	numBytes := (length + 1) / 2
	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return ""
	}

	// Encode as hex (lowercase)
	hexStr := hex.EncodeToString(randomBytes)

	// Return only the requested length
	if len(hexStr) > length {
		return hexStr[:length]
	}
	return hexStr
}

// StringUtilsDoubleBrace(string)
// Wraps a string in double braces.
func StringUtilsDoubleBrace(str string) string {
	return fmt.Sprintf("{{ %s }}", str)
}
