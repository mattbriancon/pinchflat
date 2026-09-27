package fsutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var kebabCaseReplacer = regexp.MustCompile(`[\s_]`)

// ToKebabCase converts a string to kebab-case, e.g. "hello world" or
// "hello_world" -> "hello-world".
func ToKebabCase(s string) string {
	return strings.ToLower(kebabCaseReplacer.ReplaceAllString(s, "-"))
}

// RandomString returns a random lower-case hex string of the given length.
func RandomString(length int) string {
	randomBytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(randomBytes); err != nil {
		return ""
	}

	hexStr := hex.EncodeToString(randomBytes)
	if len(hexStr) > length {
		return hexStr[:length]
	}
	return hexStr
}

// DoubleBrace wraps a string in double braces: "x" -> "{{ x }}".
func DoubleBrace(s string) string {
	return fmt.Sprintf("{{ %s }}", s)
}
