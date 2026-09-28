package store

// Ports of the custom Jason.Encoder impls for Source and MediaProfile. Their
// JSON is the payload handed to user lifecycle scripts, so field names and
// value formats must match what Elixir produced
// (testdata/elixir/golden/user_script_*.json). The MediaItem encoder lives
// in media__media_item.go.
//
// Hand-written; not a manifest row.

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/db"
)

// schemaJSONMap renders a schema struct's columns the way Jason renders an
// Ecto struct: every field by name, NULL as null, :utc_datetime as
// "2006-01-02T15:04:05Z", :date as "2006-01-02", arrays as arrays.
func schemaJSONMap(v any, exclude ...string) map[string]any {
	skip := map[string]bool{}
	for _, e := range exclude {
		skip[e] = true
	}
	out := map[string]any{}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		name := strings.Split(f.Tag.Get("db"), ",")[0]
		if name == "" || name == "-" || skip[name] || isVirtual(f.Tag.Get("db")) {
			continue
		}
		out[name] = jsonValue(rv.Field(i))
	}
	return out
}

func jsonValue(fv reflect.Value) any {
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			return nil
		}
		fv = fv.Elem()
	}
	switch x := fv.Interface().(type) {
	case db.UTCDateTime:
		return x.String()
	case db.UTCDateTimeUsec:
		return x.UTC().Format("2006-01-02T15:04:05.000000Z")
	case db.Date:
		return x.String()
	case db.NestedStringArray:
		if x == nil {
			return [][]string{}
		}
		return [][]string(x)
	}
	if fv.Kind() == reflect.Struct {
		if f := fv.FieldByName("V"); f.IsValid() {
			return f.Interface() // db.JSON[T]
		}
	}
	if fv.Kind() == reflect.String {
		return fv.String() // enum types
	}
	return fv.Interface()
}

// MarshalJSON: Source's encoder preloads media_profile and drops metadata,
// tasks and media_items. Callers must preload MediaProfile first
// (PreloadSourceMediaProfile); Go can't hit the DB from MarshalJSON.
func (s Source) MarshalJSON() ([]byte, error) {
	m := schemaJSONMap(&s)
	if s.MediaProfile != nil {
		m["media_profile"] = schemaJSONMap(s.MediaProfile)
	} else {
		m["media_profile"] = nil
	}
	return json.Marshal(m)
}

// MarshalJSON: MediaProfile's encoder drops sources.
func (p MediaProfile) MarshalJSON() ([]byte, error) {
	return json.Marshal(schemaJSONMap(&p))
}
