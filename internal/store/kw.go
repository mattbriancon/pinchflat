package store

// Keyword lists: a small port of Elixir's keyword lists, used for yt-dlp
// option lists and for `opts \\ []` function options. Hand-written W0
// infrastructure; not a manifest row.

import "fmt"

// KV is one keyword-list entry. A bare atom in the list (e.g. :no_warnings)
// is a KV with Flag set and no Value.
type KV struct {
	Key   string
	Value any
	Flag  bool
}

// KW ports an Elixir keyword list, keeping order and duplicates. Use it for
// yt-dlp option lists and for `opts \\ []` function options.
type KW []KV

// Flag builds a bare-atom entry: [:no_warnings] -> KW{Flag("no_warnings")}.
func Flag(key string) KV { return KV{Key: key, Flag: true} }

// Opt builds a key/value entry: [output: "x"] -> KW{Opt("output", "x")}.
func Opt(key string, value any) KV { return KV{Key: key, Value: value} }

// Get returns the first value for key (Keyword.get/3 without default).
func (kw KW) Get(key string) (any, bool) {
	for _, kv := range kw {
		if kv.Key == key && !kv.Flag {
			return kv.Value, true
		}
	}
	return nil, false
}

// GetOr is Keyword.get/3 with a default.
func (kw KW) GetOr(key string, def any) any {
	if v, ok := kw.Get(key); ok {
		return v
	}
	return def
}

// Bool returns a boolean option, false if absent.
func (kw KW) Bool(key string) bool {
	v, ok := kw.Get(key)
	b, _ := v.(bool)
	return ok && b
}

// String returns a string option, "" if absent.
func (kw KW) String(key string) string {
	v, _ := kw.Get(key)
	s, _ := v.(string)
	return s
}

// HasFlag reports whether the bare atom key is present (`:foo in opts`).
func (kw KW) HasFlag(key string) bool {
	for _, kv := range kw {
		if kv.Key == key && kv.Flag {
			return true
		}
	}
	return false
}

// Has reports whether key is present as a flag or key/value.
func (kw KW) Has(key string) bool {
	for _, kv := range kw {
		if kv.Key == key {
			return true
		}
	}
	return false
}

// Contains reports whether the exact entry is present (`{:k, v} in opts`).
func (kw KW) Contains(e KV) bool {
	for _, kv := range kw {
		if kv.Key == e.Key && kv.Flag == e.Flag && fmt.Sprint(kv.Value) == fmt.Sprint(e.Value) {
			return true
		}
	}
	return false
}

// Put replaces all entries for key with one value (Keyword.put/3).
func (kw KW) Put(key string, value any) KW {
	out := kw.Delete(key)
	return append(out, Opt(key, value))
}

// Delete removes all entries for key (Keyword.delete/2).
func (kw KW) Delete(key string) KW {
	out := make(KW, 0, len(kw))
	for _, kv := range kw {
		if kv.Key != key {
			out = append(out, kv)
		}
	}
	return out
}
