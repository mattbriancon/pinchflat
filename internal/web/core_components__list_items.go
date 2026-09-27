package web

// Go half of core_components.ex's list_items_from_map/1, rendered as
// Map.from_struct(record): every schema field except associations (structs
// other than Date/DateTime, lists of records), lists joined with ", ".

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/mattbriancon/pinchflat/internal/db"
)

type listItem struct{ Key, Value string }

// listItemsFromStruct returns the rows for a schema struct (or pointer to
// one). Keys come from `db` tags. Elixir iterates the map in term order,
// which for these atom keys is alphabetical, so rows are sorted by key.
func listItemsFromStruct(record any) []listItem {
	rv := reflect.ValueOf(record)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	rt := rv.Type()
	var items []listItem
	for i := 0; i < rt.NumField(); i++ {
		name := strings.Split(rt.Field(i).Tag.Get("db"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		v, ok := listItemValue(rv.Field(i))
		if !ok {
			continue
		}
		items = append(items, listItem{Key: name, Value: v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	return items
}

// listItemValue renders a field like HEEx's {v} (Phoenix.HTML.Safe);
// ok is false for fields list_items_from_map filters out.
func listItemValue(fv reflect.Value) (string, bool) {
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			return "", true
		}
		fv = fv.Elem()
	}
	switch x := fv.Interface().(type) {
	case db.UTCDateTime:
		return x.UTC().Format("2006-01-02 15:04:05Z"), true // DateTime.to_string
	case db.UTCDateTimeUsec:
		return x.UTC().Format("2006-01-02 15:04:05.000000Z"), true
	case db.Date:
		return x.String(), true
	case time.Time:
		return x.UTC().Format("2006-01-02 15:04:05Z"), true
	case db.NestedStringArray:
		parts := make([]string, len(x))
		for i, inner := range x {
			parts[i] = strings.Join(inner, "") // to_string(list) of chardata
		}
		return strings.Join(parts, ", "), true
	}
	switch fv.Kind() {
	case reflect.Struct:
		if f := fv.FieldByName("V"); f.IsValid() { // db.JSON[T]
			return listItemValue(f)
		}
		return "", false // an association
	case reflect.Slice:
		if fv.Type().Elem().Kind() == reflect.Pointer || fv.Type().Elem().Kind() == reflect.Struct {
			return "", false // has_many
		}
		parts := make([]string, fv.Len())
		for i := range parts {
			parts[i] = fmt.Sprint(fv.Index(i).Interface())
		}
		return strings.Join(parts, ", "), true
	}
	return fmt.Sprint(fv.Interface()), true
}

// listItemIsHTTPURL is `URI.parse(v).scheme =~ "http"`.
func listItemIsHTTPURL(v string) bool {
	u, err := url.Parse(v)
	return err == nil && u.Scheme != "" && strings.Contains(u.Scheme, "http")
}
