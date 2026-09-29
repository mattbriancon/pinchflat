package web

// The renderable half lives in custom_components__text_components.templ;
// this file holds the plain-Go logic (regex splitting, strftime, byte
// sizes) that templ's DSL can't express directly.

import (
	"context"
	"math"
	"regexp"
	"strings"
	"time"
)

// textDescURLPattern is ~r{https?://\S+}.
var textDescURLPattern = regexp.MustCompile(`https?://\S+`)

// textDescNewlinePattern is ~r{\n}.
var textDescNewlinePattern = regexp.MustCompile(`\n`)

// textDescLine is one entry from render_description/1's formatted_text: a
// bare URL, rendered as a link, or a run of text split on "\n" (each
// fragment rendered as its own <span>, "\n" itself becoming a block span).
type textDescLine struct {
	IsURL bool
	URL   string
	Parts []string
}

// textRenderDescriptionLines is render_description/1's Regex.split + Enum.map.
func textRenderDescriptionLines(text string) []textDescLine {
	var lines []textDescLine
	pieces := textSplitIncludeCaptures(textDescURLPattern, text)
	for _, piece := range pieces {
		if strings.HasPrefix(piece, "http") {
			lines = append(lines, textDescLine{IsURL: true, URL: piece})
			continue
		}
		parts := textSplitIncludeCapturesTrim(textDescNewlinePattern, piece)
		lines = append(lines, textDescLine{Parts: parts})
	}
	return lines
}

// textSplitIncludeCaptures is Regex.split(re, s, include_captures: true): the
// matched substrings are kept in the result, interleaved with the text
// between them.
func textSplitIncludeCaptures(re *regexp.Regexp, s string) []string {
	var out []string
	last := 0
	for _, m := range re.FindAllStringIndex(s, -1) {
		out = append(out, s[last:m[0]], s[m[0]:m[1]])
		last = m[1]
	}
	out = append(out, s[last:])
	return out
}

// textSplitIncludeCapturesTrim is the same, with trim: true (empty pieces
// dropped).
func textSplitIncludeCapturesTrim(re *regexp.Regexp, s string) []string {
	all := textSplitIncludeCaptures(re, s)
	out := make([]string, 0, len(all))
	for _, p := range all {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// textStrftime is a small Calendar.strftime/2 subset covering the directives
// datetime_in_zone/1 actually uses across the app ("%Y-%m-%d %H:%M:%S" and
// "%Y-%m-%d %H:%M").
func textStrftime(t time.Time, format string) string {
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' || i == len(format)-1 {
			b.WriteByte(c)
			continue
		}
		i++
		switch format[i] {
		case 'Y':
			b.WriteString(t.Format("2006"))
		case 'y':
			b.WriteString(t.Format("06"))
		case 'm':
			b.WriteString(t.Format("01"))
		case 'd':
			b.WriteString(t.Format("02"))
		case 'H':
			b.WriteString(t.Format("15"))
		case 'M':
			b.WriteString(t.Format("04"))
		case 'S':
			b.WriteString(t.Format("05"))
		case 'B':
			b.WriteString(t.Format("January"))
		case 'b':
			b.WriteString(t.Format("Jan"))
		case 'A':
			b.WriteString(t.Format("Monday"))
		case 'a':
			b.WriteString(t.Format("Mon"))
		case 'p':
			b.WriteString(t.Format("PM"))
		case 'I':
			b.WriteString(t.Format("03"))
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(format[i])
		}
	}
	return b.String()
}

// textDatetimeInZone is datetime_in_zone/1: Timex.Timezone.convert followed
// by Calendar.strftime. timezone falls back to Application.get_env(:pinchflat,
// :timezone) (Config.Timezone), read through the request's App.
func textDatetimeInZone(ctx context.Context, datetime time.Time, format string, timezone string) string {
	if timezone == "" {
		if a := layoutsApp(ctx); a != nil {
			timezone = a.Config.Timezone
		}
	}
	if timezone == "" {
		timezone = "UTC"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	return textStrftime(datetime.In(loc), format)
}

// textPluralize is pluralize/1.
func textPluralize(word string, count int, suffix string) string {
	if count == 1 {
		return word
	}
	return word + suffix
}

// textReadableFilesize is readable_filesize/1.
func textReadableFilesize(byteSize int64) (float64, string) {
	return humanByteSize(byteSize, 2)
}

func textReadableFilesizeValue(byteSize int64) float64 {
	num, _ := textReadableFilesize(byteSize)
	return num
}

func textReadableFilesizeSuffix(byteSize int64) string {
	_, suffix := textReadableFilesize(byteSize)
	return suffix
}

// textDefaultFormat is datetime_in_zone/1's `format` default.
func textDefaultFormat(format string) string {
	if format == "" {
		return "%Y-%m-%d %H:%M:%S"
	}
	return format
}

var byteSizeSuffixes = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

// humanByteSize converts number to a human-readable byte size, rounded to
// precision decimal places.
func humanByteSize(number int64, precision int) (float64, string) {
	value := float64(number)
	suffix := "B"
	for _, s := range byteSizeSuffixes {
		if value < 1024 {
			suffix = s
			break
		}
		value /= 1024
	}

	scale := math.Pow(10, float64(precision))
	return math.Round(value*scale) / scale, suffix
}
