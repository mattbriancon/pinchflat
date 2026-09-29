package ytdlp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Arg is one yt-dlp CLI option: a bare flag (Flag set, no value) or a
// key/value pair. Keys are snake_case names ("no_warnings").
type Arg struct {
	Key   string
	Value any
	Flag  bool
}

// Args is an ordered list of yt-dlp CLI options; order and duplicates are
// kept because yt-dlp cares about both.
type Args []Arg

// Flag appends a bare option: Flag("no_warnings") renders as --no-warnings.
func (a Args) Flag(key string) Args { return append(a, Arg{Key: key, Flag: true}) }

// Opt appends a key/value option: Opt("output", "x") renders as --output x.
func (a Args) Opt(key string, value any) Args { return append(a, Arg{Key: key, Value: value}) }

// Get returns the first value for key.
func (a Args) Get(key string) (any, bool) {
	for _, arg := range a {
		if arg.Key == key && !arg.Flag {
			return arg.Value, true
		}
	}
	return nil, false
}

// Contains reports whether the exact option is present.
func (a Args) Contains(e Arg) bool {
	for _, arg := range a {
		if arg.Key == e.Key && arg.Flag == e.Flag && fmt.Sprint(arg.Value) == fmt.Sprint(e.Value) {
			return true
		}
	}
	return false
}

// Strings renders the options as CLI arguments. Keys are kebab-cased and
// prefixed with "--"; a key/value key that already starts with "-" is passed
// through as is.
func (a Args) Strings() []string {
	var out []string
	for _, arg := range a {
		if arg.Flag {
			out = append(out, "--"+kebabCase(arg.Key))
			continue
		}
		key := arg.Key
		if len(key) == 0 || key[0] != '-' {
			key = "--" + kebabCase(key)
		}
		out = append(out, key, argString(arg.Value))
	}
	return out
}

func argString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return ""
	}
}

var kebabCaseReplacer = regexp.MustCompile(`[\s_]`)

// kebabCase converts "hello world" or "hello_world" to "hello-world".
func kebabCase(s string) string {
	return strings.ToLower(kebabCaseReplacer.ReplaceAllString(s, "-"))
}
