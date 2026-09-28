package store

// A small port of Ecto.Changeset, enough for Pinchflat's schemas. Error
// messages match Ecto's so ported tests and templates see identical text.
//
// Hand-written W0 infrastructure; not a manifest row.

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mattbriancon/pinchflat/internal/db"
)

// Attrs are the params passed to a changeset (Ecto's attrs map). Keys are
// field names (the db column names). Values may be strings (from forms) or
// Go values (from code).
type Attrs map[string]any

// FieldError is one validation error.
type FieldError struct {
	Field   string
	Message string
	// Opts are interpolated into Message's %{key} placeholders.
	Opts map[string]any
}

// Changeset tracks changes to a struct (a pointer to a schema struct) and
// validation errors.
type Changeset struct {
	// Data is the original struct pointer. Never mutated by the changeset.
	Data    any
	Changes map[string]any
	Errors  []FieldError
	// Action is set when a repo operation fails ("insert"/"update") or
	// explicitly (e.g. "validate") so forms know to display errors.
	Action string

	assocs  map[string]*Changeset
	uniques []uniqueConstraint
	fields  map[string]fieldInfo
}

type uniqueConstraint struct {
	columns []string
	field   string
	message string
}

// ChangesetError is returned by Insert/Update when the changeset is invalid
// or violates a constraint. Use errors.As to get at the changeset.
type ChangesetError struct{ Changeset *Changeset }

func (e *ChangesetError) Error() string {
	var parts []string
	for f, msgs := range e.Changeset.ErrorMap() {
		parts = append(parts, f+": "+strings.Join(msgs, ", "))
	}
	sort.Strings(parts)
	return "invalid changeset: " + strings.Join(parts, "; ")
}

// AsChangesetError returns the changeset inside err, if any.
func AsChangesetError(err error) (*Changeset, bool) {
	var ce *ChangesetError
	if errors.As(err, &ce) {
		return ce.Changeset, true
	}
	return nil, false
}

// Change starts a changeset with no casting (Ecto's change/2).
func Change(data any, changes Attrs) *Changeset {
	cs := newChangeset(data)
	for k, v := range changes {
		cs.PutChange(k, v)
	}
	return cs
}

func newChangeset(data any) *Changeset {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("changeset data must be a pointer to a struct, got %T", data))
	}
	return &Changeset{Data: data, Changes: map[string]any{}, fields: fieldsOf(rv.Type().Elem())}
}

// Cast casts permitted attrs onto data (Ecto's cast/3). Empty strings become
// nil. Values equal to the current field value are not recorded as changes.
func Cast(data any, attrs Attrs, permitted []string) *Changeset {
	cs := newChangeset(data)
	for _, field := range permitted {
		raw, ok := attrs[field]
		if !ok {
			continue
		}
		fi, ok := cs.fields[field]
		if !ok {
			panic(fmt.Sprintf("cast: %T has no field %q", data, field))
		}
		v, err := castValue(fi, raw)
		if err != nil {
			cs.AddError(field, "is invalid", map[string]any{"validation": "cast"})
			continue
		}
		if equalValues(cs.dataValue(field), v) {
			continue
		}
		cs.Changes[field] = v
	}
	return cs
}

// Valid reports whether there are no errors (including in assocs).
func (cs *Changeset) Valid() bool {
	if len(cs.Errors) > 0 {
		return false
	}
	for _, a := range cs.assocs {
		if !a.Valid() {
			return false
		}
	}
	return true
}

// GetChange returns the change for field, or nil.
func (cs *Changeset) GetChange(field string) any { return cs.Changes[field] }

// HasChange reports whether field has a change.
func (cs *Changeset) HasChange(field string) bool { _, ok := cs.Changes[field]; return ok }

// GetField returns the change for field if present, else the data value.
// Nil pointers come back as untyped nil.
func (cs *Changeset) GetField(field string) any {
	if v, ok := cs.Changes[field]; ok {
		return v
	}
	return cs.dataValue(field)
}

// PutChange records a change, converting v to the field's type.
func (cs *Changeset) PutChange(field string, v any) *Changeset {
	fi, ok := cs.fields[field]
	if !ok {
		panic(fmt.Sprintf("put_change: %T has no field %q", cs.Data, field))
	}
	cv, err := castValue(fi, v)
	if err != nil {
		panic(fmt.Sprintf("put_change %s: %v", field, err))
	}
	cs.Changes[field] = cv
	return cs
}

// DynamicDefault puts fn(cs) into field when the field is nil
// (Pinchflat.Utils.ChangesetUtils.dynamic_default/3).
func (cs *Changeset) DynamicDefault(field string, fn func(*Changeset) any) *Changeset {
	if isBlankValue(cs.GetField(field), false) {
		cs.PutChange(field, fn(cs))
	}
	return cs
}

// GenerateUUID is Ecto.UUID.generate/0.
func GenerateUUID() string { return uuid.NewString() }

// AddError adds an error (Ecto's add_error/4).
func (cs *Changeset) AddError(field, message string, opts ...map[string]any) *Changeset {
	fe := FieldError{Field: field, Message: message}
	if len(opts) > 0 {
		fe.Opts = opts[0]
	}
	cs.Errors = append(cs.Errors, fe)
	return cs
}

// ValidateRequired: nil, empty or whitespace-only strings are "can't be blank".
func (cs *Changeset) ValidateRequired(fields ...string) *Changeset {
	for _, f := range fields {
		if cs.hasErrorOn(f) {
			continue
		}
		if isBlankValue(cs.GetField(f), true) {
			cs.AddError(f, "can't be blank", map[string]any{"validation": "required"})
		}
	}
	return cs
}

// ValidateFormat checks a changed string field against a pattern with
// PCRE-like semantics (Elixir ~r regexes). message defaults to "has invalid format".
func (cs *Changeset) ValidateFormat(field, pattern string, message ...string) *Changeset {
	v, ok := cs.Changes[field]
	if !ok || v == nil {
		return cs
	}
	s, ok := derefString(v)
	if !ok {
		return cs
	}
	re, err := db.CompileRegex(pattern)
	if err != nil {
		panic(fmt.Sprintf("validate_format %s: %v", field, err))
	}
	if m, _ := re.MatchString(s); !m {
		msg := "has invalid format"
		if len(message) > 0 {
			msg = message[0]
		}
		cs.AddError(field, msg, map[string]any{"validation": "format"})
	}
	return cs
}

// NumberOpts are validate_number/3 options. Nil fields are ignored.
type NumberOpts struct {
	GreaterThan, GreaterThanOrEqualTo, LessThan, LessThanOrEqualTo, EqualTo *float64
	Message                                                                 string
}

// Num is a helper for NumberOpts literals: NumberOpts{GreaterThanOrEqualTo: Num(0)}.
func Num(f float64) *float64 { return &f }

// ValidateNumber validates a changed numeric field.
func (cs *Changeset) ValidateNumber(field string, o NumberOpts) *Changeset {
	v, ok := cs.Changes[field]
	if !ok || v == nil {
		return cs
	}
	n, ok := toFloat(v)
	if !ok {
		return cs
	}
	check := func(bound *float64, fails func(float64, float64) bool, msg string) bool {
		if bound != nil && fails(n, *bound) {
			if o.Message != "" {
				msg = o.Message
			}
			cs.AddError(field, msg, map[string]any{"validation": "number", "number": formatNumber(*bound)})
			return true
		}
		return false
	}
	_ = check(o.GreaterThan, func(n, b float64) bool { return !(n > b) }, "must be greater than %{number}") ||
		check(o.GreaterThanOrEqualTo, func(n, b float64) bool { return !(n >= b) }, "must be greater than or equal to %{number}") ||
		check(o.LessThan, func(n, b float64) bool { return !(n < b) }, "must be less than %{number}") ||
		check(o.LessThanOrEqualTo, func(n, b float64) bool { return !(n <= b) }, "must be less than or equal to %{number}") ||
		check(o.EqualTo, func(n, b float64) bool { return n != b }, "must be equal to %{number}")
	return cs
}

// ValidateInclusion checks a changed field is one of values ("is invalid").
func (cs *Changeset) ValidateInclusion(field string, values ...any) *Changeset {
	v, ok := cs.Changes[field]
	if !ok || v == nil {
		return cs
	}
	for _, allowed := range values {
		if equalValues(v, allowed) {
			return cs
		}
	}
	cs.AddError(field, "is invalid", map[string]any{"validation": "inclusion"})
	return cs
}

// ValidateChange runs fn on a changed, non-nil field; fn returns error messages.
func (cs *Changeset) ValidateChange(field string, fn func(value any) []string) *Changeset {
	v, ok := cs.Changes[field]
	if !ok || v == nil {
		return cs
	}
	for _, msg := range fn(v) {
		cs.AddError(field, msg)
	}
	return cs
}

// UniqueConstraint maps a UNIQUE violation on columns to an error on
// errorKey (defaults to the first column) with "has already been taken".
func (cs *Changeset) UniqueConstraint(columns []string, errorKey string, message ...string) *Changeset {
	if errorKey == "" {
		errorKey = columns[0]
	}
	msg := "has already been taken"
	if len(message) > 0 {
		msg = message[0]
	}
	cs.uniques = append(cs.uniques, uniqueConstraint{columns: columns, field: errorKey, message: msg})
	return cs
}

// CastAssoc casts attrs[field] (a map) with fn into a child changeset for a
// has_one association (Ecto's cast_assoc/3 with on_replace: :update). The
// child is inserted/updated by Insert/Update after the parent.
func (cs *Changeset) CastAssoc(attrs Attrs, field string, current any, fn func(data any, attrs Attrs) *Changeset) *Changeset {
	raw, ok := attrs[field]
	if !ok || raw == nil {
		return cs
	}
	child, ok := toAttrs(raw)
	if !ok {
		cs.AddError(field, "is invalid", map[string]any{"validation": "assoc"})
		return cs
	}
	if cs.assocs == nil {
		cs.assocs = map[string]*Changeset{}
	}
	cs.assocs[field] = fn(current, child)
	return cs
}

// Assoc returns the child changeset cast for field, if any.
func (cs *Changeset) Assoc(field string) *Changeset { return cs.assocs[field] }

// ErrorMap returns field -> interpolated messages, like the Elixir tests'
// errors_on/1. Assoc errors are nested under "field.child".
func (cs *Changeset) ErrorMap() map[string][]string {
	out := map[string][]string{}
	for _, e := range cs.Errors {
		out[e.Field] = append(out[e.Field], Interpolate(e.Message, e.Opts))
	}
	for name, a := range cs.assocs {
		for f, msgs := range a.ErrorMap() {
			out[name+"."+f] = append(out[name+"."+f], msgs...)
		}
	}
	return out
}

// ErrorsOn returns the interpolated messages for one field.
func (cs *Changeset) ErrorsOn(field string) []string { return cs.ErrorMap()[field] }

var placeholderRe = regexp.MustCompile(`%\{(\w+)\}`)

// Interpolate replaces %{key} with opts[key].
func Interpolate(msg string, opts map[string]any) string {
	return placeholderRe.ReplaceAllStringFunc(msg, func(m string) string {
		key := m[2 : len(m)-1]
		if v, ok := opts[key]; ok {
			return fmt.Sprint(v)
		}
		return key
	})
}

// Apply returns a copy of Data with Changes applied (Ecto's apply_changes/1).
func (cs *Changeset) Apply() any {
	rv := reflect.ValueOf(cs.Data).Elem()
	cp := reflect.New(rv.Type())
	cp.Elem().Set(rv)
	for field, v := range cs.Changes {
		setField(cp.Elem().FieldByIndex(cs.fields[field].index), v)
	}
	return cp.Interface()
}

func (cs *Changeset) hasErrorOn(field string) bool {
	for _, e := range cs.Errors {
		if e.Field == field {
			return true
		}
	}
	return false
}

func (cs *Changeset) dataValue(field string) any {
	fi, ok := cs.fields[field]
	if !ok {
		panic(fmt.Sprintf("%T has no field %q", cs.Data, field))
	}
	fv := reflect.ValueOf(cs.Data).Elem().FieldByIndex(fi.index)
	if fv.Kind() == reflect.Pointer && fv.IsNil() {
		return nil
	}
	return fv.Interface()
}

// --- field metadata and casting ---

type fieldInfo struct {
	name  string
	index []int
	typ   reflect.Type
	enum  []string
}

var fieldCache sync.Map // reflect.Type -> map[string]fieldInfo

// fieldsOf maps db column names to struct fields. Fields tagged db:"-" or
// without a db tag are ignored. An `enum:"a,b,c"` tag restricts values.
func fieldsOf(t reflect.Type) map[string]fieldInfo {
	if m, ok := fieldCache.Load(t); ok {
		return m.(map[string]fieldInfo)
	}
	m := map[string]fieldInfo{}
	var walk func(t reflect.Type, prefix []int)
	walk = func(t reflect.Type, prefix []int) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			idx := append(append([]int{}, prefix...), i)
			if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get("db") == "" {
				walk(f.Type, idx)
				continue
			}
			name := strings.Split(f.Tag.Get("db"), ",")[0]
			if name == "" || name == "-" || isVirtual(f.Tag.Get("db")) {
				continue
			}
			fi := fieldInfo{name: name, index: idx, typ: f.Type}
			if e := f.Tag.Get("enum"); e != "" {
				fi.enum = strings.Split(e, ",")
			}
			m[name] = fi
		}
	}
	walk(t, nil)
	fieldCache.Store(t, m)
	return m
}

var (
	timeType        = reflect.TypeOf(time.Time{})
	utcType         = reflect.TypeOf(db.UTCDateTime{})
	dateType        = reflect.TypeOf(db.Date{})
	scannerIface    = reflect.TypeOf((*interface{ Scan(any) error })(nil)).Elem()
	errNotCastable  = errors.New("not castable")
	emptyStringNils = true
)

// castValue converts raw to a value assignable to the field (a pointer
// field gets a non-pointer value; nil means NULL/zero).
func castValue(fi fieldInfo, raw any) (any, error) {
	if raw == nil {
		return nil, nil
	}
	// Matches Ecto.Type.empty_trimmed_string?/1: a string is "empty" (and
	// cast to nil) when trimming its leading whitespace leaves nothing, not
	// only when it's the exact empty string.
	if s, ok := raw.(string); ok && emptyStringNils && strings.TrimLeft(s, " \t\n\v\f\r\u0085 ") == "" {
		return nil, nil
	}
	base := fi.typ
	if base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	rv := reflect.ValueOf(raw)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
		raw = rv.Interface()
	}

	var out any
	switch {
	case base == utcType || base == dateType:
		// Truncate even when raw is already the target type: Ecto's
		// :utc_datetime/:date casts always truncate, regardless of the
		// input's existing precision.
		t, err := toTime(raw)
		if err != nil {
			return nil, err
		}
		switch base {
		case utcType:
			out = db.UTCDateTime{Time: t.UTC().Truncate(time.Second)}
		default:
			out = db.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
		}
	case rv.Type() == base:
		out = raw
	case base == timeType:
		t, err := toTime(raw)
		if err != nil {
			return nil, err
		}
		out = t
	case base.Kind() == reflect.String:
		s, ok := raw.(string)
		if !ok {
			if rv.Kind() != reflect.String {
				return nil, errNotCastable
			}
			s = rv.String()
		}
		v := reflect.New(base).Elem()
		v.SetString(s)
		out = v.Interface()
	case base.Kind() == reflect.Bool:
		switch v := raw.(type) {
		case string:
			switch strings.ToLower(v) {
			case "true", "1", "on":
				out = true
			case "false", "0", "off":
				out = false
			default:
				return nil, errNotCastable
			}
		default:
			if rv.Kind() == reflect.Bool {
				out = rv.Bool()
			} else {
				return nil, errNotCastable
			}
		}
		out = reflect.ValueOf(out).Convert(base).Interface()
	case base.Kind() >= reflect.Int && base.Kind() <= reflect.Int64:
		var n int64
		switch {
		case rv.Kind() == reflect.String:
			i, err := strconv.ParseInt(strings.TrimSpace(rv.String()), 10, 64)
			if err != nil {
				return nil, err
			}
			n = i
		case rv.CanInt():
			n = rv.Int()
		case rv.CanUint():
			n = int64(rv.Uint())
		case rv.CanFloat() && rv.Float() == float64(int64(rv.Float())):
			n = int64(rv.Float())
		default:
			return nil, errNotCastable
		}
		out = reflect.ValueOf(n).Convert(base).Interface()
	case base.Kind() == reflect.Float32 || base.Kind() == reflect.Float64:
		f, ok := toFloat(raw)
		if !ok {
			return nil, errNotCastable
		}
		out = reflect.ValueOf(f).Convert(base).Interface()
	case reflect.PointerTo(base).Implements(scannerIface):
		// JSON-backed types etc.: accept the Go value or a JSON string.
		v := reflect.New(base)
		if s, ok := raw.(string); ok {
			if err := v.Interface().(interface{ Scan(any) error }).Scan(s); err != nil {
				return nil, err
			}
		} else if err := assignVia(v, raw); err != nil {
			return nil, err
		}
		out = v.Elem().Interface()
	case rv.Type().ConvertibleTo(base):
		out = rv.Convert(base).Interface()
	default:
		return nil, errNotCastable
	}

	if len(fi.enum) > 0 {
		s := fmt.Sprint(out)
		ok := false
		for _, e := range fi.enum {
			if e == s {
				ok = true
				break
			}
		}
		if !ok {
			return nil, errNotCastable
		}
	}
	return out, nil
}

// assignVia sets JSON-ish wrapper types (db.JSON[T] with a V field, or
// slice types like db.NestedStringArray) from a plain Go value.
func assignVia(dst reflect.Value, raw any) error {
	e := dst.Elem()
	rv := reflect.ValueOf(raw)
	if e.Kind() == reflect.Struct {
		if f := e.FieldByName("V"); f.IsValid() && rv.Type().AssignableTo(f.Type()) {
			f.Set(rv)
			return nil
		}
		if f := e.FieldByName("V"); f.IsValid() && rv.Type().ConvertibleTo(f.Type()) {
			f.Set(rv.Convert(f.Type()))
			return nil
		}
	}
	if rv.Type().ConvertibleTo(e.Type()) {
		e.Set(rv.Convert(e.Type()))
		return nil
	}
	return errNotCastable
}

func toTime(raw any) (time.Time, error) {
	switch v := raw.(type) {
	case time.Time:
		return v, nil
	case db.UTCDateTime:
		return v.Time, nil
	case db.UTCDateTimeUsec:
		return v.Time, nil
	case db.Date:
		return v.Time, nil
	case string:
		return db.ParseTime(v)
	}
	return time.Time{}, errNotCastable
}

func setField(f reflect.Value, v any) {
	if v == nil {
		f.Set(reflect.Zero(f.Type()))
		return
	}
	val := reflect.ValueOf(v)
	if f.Kind() == reflect.Pointer {
		p := reflect.New(f.Type().Elem())
		p.Elem().Set(val.Convert(f.Type().Elem()))
		f.Set(p)
		return
	}
	f.Set(val.Convert(f.Type()))
}

func equalValues(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if ra.Kind() == reflect.Pointer {
		if ra.IsNil() {
			return b == nil
		}
		a = ra.Elem().Interface()
	}
	if rb.Kind() == reflect.Pointer {
		if rb.IsNil() {
			return false
		}
		b = rb.Elem().Interface()
	}
	if ta, ok := a.(interface{ Equal(time.Time) bool }); ok {
		if tb, err := toTime(b); err == nil {
			return ta.Equal(tb)
		}
	}
	switch ta := a.(type) {
	case db.UTCDateTime:
		if tb, err := toTime(b); err == nil {
			return ta.Equal(tb)
		}
	case db.Date:
		if tb, err := toTime(b); err == nil {
			return ta.Equal(tb)
		}
	}
	return reflect.DeepEqual(a, b)
}

func isBlankValue(v any, trimStrings bool) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.String {
		s := rv.String()
		if trimStrings {
			s = strings.TrimSpace(s)
		}
		return s == ""
	}
	return false
}

func derefString(v any) (string, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "", false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.String {
		return "", false
	}
	return rv.String(), true
}

func toFloat(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0, false
		}
		rv = rv.Elem()
	}
	switch {
	case rv.CanInt():
		return float64(rv.Int()), true
	case rv.CanUint():
		return float64(rv.Uint()), true
	case rv.CanFloat():
		return rv.Float(), true
	case rv.Kind() == reflect.String:
		f, err := strconv.ParseFloat(strings.TrimSpace(rv.String()), 64)
		return f, err == nil
	}
	return 0, false
}

func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func toAttrs(v any) (Attrs, bool) {
	switch m := v.(type) {
	case Attrs:
		return m, true
	case map[string]any:
		return Attrs(m), true
	case map[string]string:
		out := Attrs{}
		for k, v := range m {
			out[k] = v
		}
		return out, true
	}
	return nil, false
}
