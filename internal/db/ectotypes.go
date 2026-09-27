package db

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Value types that read and write exactly what Ecto/ecto_sqlite3 stored.
// Never pass a bare time.Time to the driver: its default format differs from
// Ecto's, and the app's SQL compares timestamps as text and calls DATE().

// UTCDateTime is Ecto's :utc_datetime: TEXT "2006-01-02T15:04:05Z", second
// precision. Reads also accept the legacy forms found in real databases:
// "2006-01-02T15:04:05" (no Z, from the uploaded_at backfill), "2006-01-02"
// (the uploaded_at column default) and fractional seconds.
type UTCDateTime struct{ time.Time }

// UTCDateTimeUsec is Ecto's :utc_datetime_usec (used by Oban for
// scheduled_at, attempted_at, ...): "2006-01-02T15:04:05.000000Z".
type UTCDateTimeUsec struct{ time.Time }

// Date is Ecto's :date: TEXT "2006-01-02".
type Date struct{ time.Time }

// NaiveDateTime is how Ecto records schema_migrations.inserted_at:
// "2006-01-02T15:04:05".
type NaiveDateTime struct{ time.Time }

const (
	utcLayout     = "2006-01-02T15:04:05Z"
	utcUsecLayout = "2006-01-02T15:04:05.000000Z"
	naiveLayout   = "2006-01-02T15:04:05"
	dateLayout    = "2006-01-02"
)

// Now returns the current time truncated the way Ecto truncates
// :utc_datetime (to the second).
func Now() UTCDateTime { return UTCDateTime{time.Now().UTC().Truncate(time.Second)} }

// NowUsec returns the current time at microsecond precision.
func NowUsec() UTCDateTimeUsec { return UTCDateTimeUsec{time.Now().UTC().Truncate(time.Microsecond)} }

func (t UTCDateTime) Value() (driver.Value, error) {
	return t.UTC().Truncate(time.Second).Format(utcLayout), nil
}
func (t *UTCDateTime) Scan(src any) error { return scanTime(src, &t.Time) }
func (t UTCDateTime) String() string      { return t.UTC().Format(utcLayout) }

func (t UTCDateTimeUsec) Value() (driver.Value, error) {
	return t.UTC().Truncate(time.Microsecond).Format(utcUsecLayout), nil
}
func (t *UTCDateTimeUsec) Scan(src any) error { return scanTime(src, &t.Time) }

func (d Date) Value() (driver.Value, error) { return d.UTC().Format(dateLayout), nil }
func (d *Date) Scan(src any) error          { return scanTime(src, &d.Time) }
func (d Date) String() string               { return d.UTC().Format(dateLayout) }

func (t NaiveDateTime) Value() (driver.Value, error) {
	return t.UTC().Truncate(time.Second).Format(naiveLayout), nil
}
func (t *NaiveDateTime) Scan(src any) error { return scanTime(src, &t.Time) }

// ParseTime parses every timestamp form found in Pinchflat databases.
func ParseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		time.RFC3339Nano,             // 2006-01-02T15:04:05.999999Z / with offset
		"2006-01-02T15:04:05.999999", // naive, optional fraction
		"2006-01-02 15:04:05.999999", // SQLite CURRENT_TIMESTAMP (Oban inserted_at)
		"2006-01-02 15:04:05Z07:00",
		dateLayout,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised timestamp %q", s)
}

func scanTime(src any, dst *time.Time) error {
	switch v := src.(type) {
	case string:
		t, err := ParseTime(v)
		if err != nil {
			return err
		}
		*dst = t
	case []byte:
		return scanTime(string(v), dst)
	case time.Time:
		*dst = v.UTC()
	default:
		return fmt.Errorf("cannot scan %T into a timestamp", src)
	}
	return nil
}

// JSON stores T the way ecto_sqlite3 stores :map and {:array, _} columns:
// compact JSON text, with no HTML escaping (Jason's default).
type JSON[T any] struct{ V T }

func NewJSON[T any](v T) JSON[T] { return JSON[T]{V: v} }

func (j JSON[T]) Value() (driver.Value, error) { return EncodeJSON(j.V) }

func (j *JSON[T]) Scan(src any) error {
	var b []byte
	switch v := src.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	case nil:
		var zero T
		j.V = zero
		return nil
	default:
		return fmt.Errorf("cannot scan %T into JSON", src)
	}
	return json.Unmarshal(b, &j.V)
}

func (j JSON[T]) MarshalJSON() ([]byte, error)  { return json.Marshal(j.V) }
func (j *JSON[T]) UnmarshalJSON(b []byte) error { return json.Unmarshal(b, &j.V) }

// EncodeJSON encodes like Jason.encode!/1: compact, no HTML escaping.
func EncodeJSON(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// NestedStringArray is Ecto {:array, {:array, :string}} as ecto_sqlite3
// stores it: an outer JSON array whose elements are JSON-encoded *strings*
// of the inner arrays, e.g. ["[\"en\",\"/a.en.srt\"]"]. Used by
// media_items.subtitle_filepaths.
type NestedStringArray [][]string

func (a NestedStringArray) Value() (driver.Value, error) {
	outer := make([]string, 0, len(a))
	for _, inner := range a {
		s, err := EncodeJSON(inner)
		if err != nil {
			return nil, err
		}
		outer = append(outer, s)
	}
	return EncodeJSON(outer)
}

func (a *NestedStringArray) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case nil:
		*a = nil
		return nil
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("cannot scan %T into NestedStringArray", src)
	}
	var outer []json.RawMessage
	if err := json.Unmarshal([]byte(s), &outer); err != nil {
		return err
	}
	res := make([][]string, 0, len(outer))
	for _, raw := range outer {
		var inner []string
		// Elements are normally strings containing JSON; accept a bare
		// array too, in case anything ever wrote it that way.
		var str string
		if err := json.Unmarshal(raw, &str); err == nil {
			if err := json.Unmarshal([]byte(str), &inner); err != nil {
				return err
			}
		} else if err := json.Unmarshal(raw, &inner); err != nil {
			return err
		}
		res = append(res, inner)
	}
	*a = res
	return nil
}
