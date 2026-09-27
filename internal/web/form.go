package web

// A small port of Phoenix.HTML.Form over core.Changeset, so templates can do
// what `to_form(changeset)` + `<.input field={f[:name]} />` did.

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db"
)

// Form wraps a changeset under a param name ("source", "media_profile").
type Form struct {
	As        string
	Changeset *core.Changeset
	// Params, when set, are the raw submitted params; Phoenix shows
	// those back to the user after a failed submit.
	Params core.Attrs
}

// FormFor is to_form(changeset, as: as).
func FormFor(cs *core.Changeset, as string) *Form { return &Form{As: as, Changeset: cs} }

// FormField is Phoenix.HTML.FormField.
type FormField struct {
	ID     string // "source_custom_name"
	Name   string // "source[custom_name]"
	Value  any    // current value (change or data)
	Errors []string
}

// Field is f[:name]. Errors are only shown once the changeset has an action
// (i.e. after a failed insert/update), like Phoenix.
func (f *Form) Field(name string) FormField {
	ff := FormField{ID: f.As + "_" + name, Name: f.As + "[" + name + "]"}
	if f.Changeset != nil {
		if raw, ok := f.Params[name]; ok {
			ff.Value = raw
		} else {
			ff.Value = f.Changeset.GetField(name)
		}
		if f.Changeset.Action != "" {
			ff.Errors = f.Changeset.ErrorsOn(name)
		}
	}
	return ff
}

// HasErrors reports whether the error banner should show (@changeset.action).
func (f *Form) HasErrors() bool {
	return f.Changeset != nil && f.Changeset.Action != "" && !f.Changeset.Valid()
}

// InputValue renders a value like Phoenix.HTML.Form.normalize_value/2.
func InputValue(v any) string {
	if v == nil {
		return ""
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		v = rv.Elem().Interface()
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case db.Date:
		return x.String()
	case db.UTCDateTime:
		return x.UTC().Format("2006-01-02T15:04:05Z")
	case time.Time:
		return x.UTC().Format(time.RFC3339)
	case db.JSON[[]string]:
		return strings.Join(x.V, ",")
	}
	return fmt.Sprint(v)
}

// Checked reports whether a checkbox/toggle value is on.
func Checked(v any) bool { s := InputValue(v); return s == "true" || s == "on" || s == "1" }

// ParseForm is Phoenix's %{"source" => params}: it returns the params nested
// under `as` as Attrs. Nested maps (source[metadata][x]) become Attrs too;
// repeated keys (x[]) become []string.
func ParseForm(r *http.Request, as string) core.Attrs {
	_ = r.ParseForm()
	out := core.Attrs{}
	prefix := as + "["
	keys := make([]string, 0, len(r.PostForm))
	for k := range r.PostForm {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		for k := range r.Form {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		vals := r.Form[k]
		path := parseBracketPath(k[len(as):])
		setNested(out, path, vals)
	}
	return out
}

// parseBracketPath turns "[a][b][]" into ["a","b",""].
func parseBracketPath(s string) []string {
	var out []string
	for strings.HasPrefix(s, "[") {
		end := strings.IndexByte(s, ']')
		if end < 0 {
			break
		}
		out = append(out, s[1:end])
		s = s[end+1:]
	}
	return out
}

func setNested(m core.Attrs, path []string, vals []string) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 || (len(path) == 2 && path[1] == "") {
		if len(path) == 2 { // x[] -> list
			m[path[0]] = vals
		} else {
			m[path[0]] = vals[len(vals)-1] // last wins, like Plug
		}
		return
	}
	child, ok := m[path[0]].(core.Attrs)
	if !ok {
		child = core.Attrs{}
		m[path[0]] = child
	}
	setNested(child, path[1:], vals)
}
