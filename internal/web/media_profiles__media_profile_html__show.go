package web

// Plain-Go half of media_profile_html/show.html.heex: list_items_from_map/1
// (core_components.ex) applied to Map.from_struct(@media_profile). The
// shared core_components.templ has no such component (see the report), so
// it's reimplemented here, scoped to MediaProfile's fields.

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db"
)

// mpAttr is one {k, v} pair list_items_from_map/1 renders.
type mpAttr struct {
	Key   string
	Value string
	IsURL bool
}

// mediaProfilesRawAttributes is Map.from_struct(@media_profile) followed by
// list_items_from_map's own filter/format pass: struct fields other than
// Date/DateTime are dropped (there are none besides the timestamps here,
// since Sources is tagged db:"-"), lists are joined with ", ", and anything
// that looks like an http(s) URL is flagged so the template can link it.
func mediaProfilesRawAttributes(p *core.MediaProfile) []mpAttr {
	var out []mpAttr
	v := reflect.ValueOf(p).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		tag, _, _ = strings.Cut(tag, ",")
		out = append(out, mpAttr{Key: tag, Value: mediaProfilesAttrValue(v.Field(i).Interface())})
	}
	for i := range out {
		out[i].IsURL = strings.HasPrefix(out[i].Value, "http")
	}
	return out
}

func mediaProfilesAttrValue(iface any) string {
	switch x := iface.(type) {
	case bool:
		if x {
			return "true"
		}
		return "false"
	case string:
		return x
	case int, int64:
		return fmt.Sprint(x)
	case db.JSON[[]string]:
		return strings.Join(x.V, ", ")
	case db.UTCDateTime:
		return x.UTC().Format("2006-01-02T15:04:05Z")
	}
	rv := reflect.ValueOf(iface)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		return mediaProfilesAttrValue(rv.Elem().Interface())
	}
	return fmt.Sprint(iface)
}
