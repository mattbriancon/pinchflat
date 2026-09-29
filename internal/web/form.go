package web

// A small port of Phoenix.HTML.Form, so templates can do what
// `to_form(...)` + `<.input field={f[:name]} />` did.

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/mattbriancon/pinchflat/internal/db"
)

// Form is a form under a param name ("source", "media_profile"): the values
// to display and the validation errors, by field name.
type Form struct {
	As     string
	Values map[string]any
	Errors map[string][]string
}

// NewForm builds a form from plain values and validation errors.
func NewForm(as string, values map[string]any, errs map[string][]string) *Form {
	return &Form{As: as, Values: values, Errors: errs}
}

// FormField is Phoenix.HTML.FormField.
type FormField struct {
	ID     string // "source_custom_name"
	Name   string // "source[custom_name]"
	Value  any    // current value (change or data)
	Errors []string
}

// Field is f[:name].
func (f *Form) Field(name string) FormField {
	return FormField{
		ID:     f.As + "_" + name,
		Name:   f.As + "[" + name + "]",
		Value:  f.Values[name],
		Errors: f.Errors[name],
	}
}

// HasErrors reports whether the error banner should show.
func (f *Form) HasErrors() bool { return len(f.Errors) > 0 }

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
