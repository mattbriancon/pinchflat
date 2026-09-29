package store

import (
	"errors"
	"reflect"
	"sort"
	"strings"
)

// ValidationErrors is a field name -> messages map returned (as an error) when
// typed params fail validation. Nested fields are keyed "assoc.field".
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

// paramSpec describes one castable input column of a params struct P: its
// name, base type (and allowed values for enums), and a pointer to the
// params field that holds it.
type paramSpec[P any] struct {
	name  string
	typ   reflect.Type
	enum  []string
	field func(*P) any // returns a pointer to the params' *T field
}

func typeOf[T any]() reflect.Type { return reflect.TypeOf((*T)(nil)).Elem() }

// parseParams casts each present attrs key into p, recording which fields were
// submitted in set and cast failures in errs.
func parseParams[P any](specs []paramSpec[P], p *P, attrs Attrs, set map[string]bool, errs map[string][]string) {
	for _, sp := range specs {
		raw, ok := attrs[sp.name]
		if !ok {
			continue
		}
		v, err := castValue(fieldInfo{name: sp.name, typ: sp.typ, enum: sp.enum}, raw)
		if err != nil {
			errs[sp.name] = append(errs[sp.name], "is invalid")
			continue
		}
		set[sp.name] = true
		setParam(sp.field(p), sp.typ, v)
	}
}

func setParam(dst any, typ reflect.Type, v any) {
	f := reflect.ValueOf(dst).Elem()
	if v == nil {
		f.Set(reflect.Zero(f.Type()))
		return
	}
	nv := reflect.New(typ)
	nv.Elem().Set(reflect.ValueOf(v))
	f.Set(nv)
}

// getParam returns the value a params field holds, or nil.
func getParam(dst any) any {
	f := reflect.ValueOf(dst).Elem()
	if f.IsNil() {
		return nil
	}
	return f.Elem().Interface()
}

// paramValue is the value held by the params field called name.
func paramValue[P any](specs []paramSpec[P], p *P, name string) any {
	for _, sp := range specs {
		if sp.name == name {
			return getParam(sp.field(p))
		}
	}
	return nil
}

// applyParams writes the submitted fields onto dst (a struct value).
func applyParams[P any](specs []paramSpec[P], p *P, set map[string]bool, dst reflect.Value) {
	fields := fieldsOf(dst.Type())
	for _, sp := range specs {
		if set[sp.name] {
			setField(dst.FieldByIndex(fields[sp.name].index), getParam(sp.field(p)))
		}
	}
}

// structValue reads a column of a record: nil pointers come back as untyped nil.
func structValue(rec any, column string) any {
	rv := reflect.ValueOf(rec).Elem()
	fi, ok := fieldsOf(rv.Type())[column]
	if !ok {
		return nil
	}
	fv := rv.FieldByIndex(fi.index)
	if fv.Kind() == reflect.Pointer && fv.IsNil() {
		return nil
	}
	return fv.Interface()
}

func copySet(set map[string]bool) map[string]bool {
	out := make(map[string]bool, len(set)+2)
	for k, v := range set {
		out[k] = v
	}
	return out
}

func copyErrs(errs map[string][]string) map[string][]string {
	out := make(map[string][]string, len(errs))
	for k, v := range errs {
		out[k] = append([]string(nil), v...)
	}
	return out
}

func addErr(errs map[string][]string, field, msg string) {
	errs[field] = append(errs[field], msg)
}
