package store

// Shared plumbing for the typed params structs (SourceParams,
// MediaProfileParams, ...): validation errors, parsing url.Values, and
// applying submitted values onto a copy of a record while noting which
// columns changed.

import (
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mattbriancon/pinchflat/internal/db"
)

// ValidationErrors is a field name -> messages map returned (as an error) when
// typed params fail validation or violate a unique index. Nested fields are
// keyed "assoc.field".
type ValidationErrors map[string][]string

func (v ValidationErrors) Error() string {
	var parts []string
	for f, msgs := range v {
		parts = append(parts, f+": "+strings.Join(msgs, ", "))
	}
	sort.Strings(parts)
	return "invalid params: " + strings.Join(parts, "; ")
}

// AsValidationErrors returns the validation errors inside err, if any.
func AsValidationErrors(err error) (map[string][]string, bool) {
	var v ValidationErrors
	if errors.As(err, &v) {
		return v, true
	}
	return nil, false
}

func addErr(errs map[string][]string, field, msg string) {
	errs[field] = append(errs[field], msg)
}

// GenerateUUID is Ecto.UUID.generate/0.
func GenerateUUID() string { return uuid.NewString() }

// --- parsing url.Values ---

// isBlank reports whether a submitted string counts as empty. Like Ecto, only
// leading whitespace is considered: a string is blank when trimming it leaves
// nothing.
func isBlank(s string) bool {
	return strings.TrimLeft(s, " \t\n\v\f\r\u0085 ") == ""
}

// fieldState is what a form said about one field.
type fieldState int

const (
	fieldAbsent fieldState = iota // not submitted (or unparseable: an error was recorded)
	fieldBlank                    // submitted empty
	fieldSet                      // submitted with a value
)

// formReader reads the fields of one form from posted values. With as ==
// "source" the field "name" is the key "source[name]"; with as == "" it is
// "name". The last value wins, like Plug. Unparseable values are recorded in
// errs as "is invalid".
type formReader struct {
	values url.Values
	as     string
	errs   map[string][]string
}

func newFormReader(values url.Values, as string) *formReader {
	return &formReader{values: values, as: as, errs: map[string][]string{}}
}

func (f *formReader) key(field string) string {
	if f.as == "" {
		return field
	}
	return f.as + "[" + field + "]"
}

func (f *formReader) get(field string) (string, bool) {
	vs := f.values[f.key(field)]
	if len(vs) == 0 {
		return "", false
	}
	return vs[len(vs)-1], true
}

func (f *formReader) invalid(field string) { addErr(f.errs, field, "is invalid") }

// str reads a string field. A blank string is stored as "" (which required
// columns reject and nullable ones turn into NULL).
func (f *formReader) str(field string, dst **string) {
	if s, ok := f.get(field); ok {
		if isBlank(s) {
			s = ""
		}
		*dst = &s
	}
}

// parseBool accepts true/1/on and false/0/off in any case.
func parseBool(s string) (value, ok bool) {
	switch strings.ToLower(s) {
	case "true", "1", "on":
		return true, true
	case "false", "0", "off":
		return false, true
	}
	return false, false
}

// boolean reads a boolean field.
func (f *formReader) boolean(field string) (bool, fieldState) {
	s, ok := f.get(field)
	switch {
	case !ok:
		return false, fieldAbsent
	case isBlank(s):
		return false, fieldBlank
	}
	b, ok := parseBool(s)
	if !ok {
		f.invalid(field)
		return false, fieldAbsent
	}
	return b, fieldSet
}

// boolOrFalse reads a boolean field for which blank means false.
func (f *formReader) boolOrFalse(field string, dst **bool) {
	switch b, st := f.boolean(field); st {
	case fieldSet:
		*dst = &b
	case fieldBlank:
		*dst = Ptr(false)
	}
}

// integer reads an int field.
func (f *formReader) integer(field string) (int, fieldState) {
	s, ok := f.get(field)
	switch {
	case !ok:
		return 0, fieldAbsent
	case isBlank(s):
		return 0, fieldBlank
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		f.invalid(field)
		return 0, fieldAbsent
	}
	return n, fieldSet
}

// integer64 reads an int64 field.
func (f *formReader) integer64(field string) (int64, fieldState) {
	s, ok := f.get(field)
	switch {
	case !ok:
		return 0, fieldAbsent
	case isBlank(s):
		return 0, fieldBlank
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		f.invalid(field)
		return 0, fieldAbsent
	}
	return n, fieldSet
}

// date reads a "2006-01-02" field.
func (f *formReader) date(field string) (db.Date, fieldState) {
	s, ok := f.get(field)
	switch {
	case !ok:
		return db.Date{}, fieldAbsent
	case isBlank(s):
		return db.Date{}, fieldBlank
	}
	t, err := db.ParseTime(strings.TrimSpace(s))
	if err != nil {
		f.invalid(field)
		return db.Date{}, fieldAbsent
	}
	return truncateDate(t), fieldSet
}

// enumeration reads a string field restricted to allowed. A blank value is
// "" (a required column then rejects it).
func enumeration[T ~string](f *formReader, field string, dst **T, allowed ...T) {
	s, ok := f.get(field)
	if !ok {
		return
	}
	if isBlank(s) {
		*dst = Ptr(T(""))
		return
	}
	for _, a := range allowed {
		if string(a) == s {
			*dst = Ptr(a)
			return
		}
	}
	f.invalid(field)
}

// setOrClear stores v into dst when st is fieldSet, and ORs flag into clear
// when st is fieldBlank.
func setOrClear[T any, C ~uint32](dst **T, v T, st fieldState, clear *C, flag C) {
	switch st {
	case fieldSet:
		*dst = &v
	case fieldBlank:
		*clear |= flag
	}
}

// --- applying params onto a record ---

// changes is the set of columns a params struct changed on a record.
type changes map[string]bool

func (c changes) list() []string {
	out := make([]string, 0, len(c))
	for col := range c {
		out = append(out, col)
	}
	sort.Strings(out)
	return out
}

// set writes a submitted value into a non-null column.
func setValue[T comparable](c changes, col string, dst *T, v *T) {
	if v != nil && *dst != *v {
		c[col] = true
		*dst = *v
	}
}

// setNullable writes a submitted value into a nullable column; clear (or, for
// strings, a blank value: see setNullableString) sets it to NULL.
func setNullable[T comparable](c changes, col string, dst **T, v *T, clear bool) {
	switch {
	case clear:
		if *dst != nil {
			c[col] = true
			*dst = nil
		}
	case v != nil:
		if *dst == nil || **dst != *v {
			c[col] = true
			*dst = Ptr(*v)
		}
	}
}

// setString writes a submitted string into a non-null column, where a blank
// string is "".
func setString(c changes, col string, dst *string, v *string) {
	if v == nil {
		return
	}
	s := *v
	if isBlank(s) {
		s = ""
	}
	setValue(c, col, dst, &s)
}

// setNullableString writes a submitted string into a nullable column, where a
// blank string (or clear) is NULL.
func setNullableString(c changes, col string, dst **string, v *string, clear bool) {
	if v != nil && isBlank(*v) {
		clear = true
	}
	setNullable(c, col, dst, v, clear)
}

// setTime writes a submitted time into a non-null datetime column, truncated
// to the second like Ecto's :utc_datetime.
func setTime(c changes, col string, dst *db.UTCDateTime, v *time.Time) {
	if v == nil {
		return
	}
	t := utcSecond(*v)
	if !dst.Equal(t.Time) {
		c[col] = true
		*dst = t
	}
}

// setNullableTime is setTime for a nullable column.
func setNullableTime(c changes, col string, dst **db.UTCDateTime, v *time.Time, clear bool) {
	switch {
	case clear:
		if *dst != nil {
			c[col] = true
			*dst = nil
		}
	case v != nil:
		t := utcSecond(*v)
		if *dst == nil || !(*dst).Equal(t.Time) {
			c[col] = true
			*dst = &t
		}
	}
}

func utcSecond(t time.Time) db.UTCDateTime {
	return db.UTCDateTime{Time: t.UTC().Truncate(time.Second)}
}

func truncateDate(t time.Time) db.Date {
	return db.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
}
