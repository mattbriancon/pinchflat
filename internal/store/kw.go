package store

// KV is one key/value entry. Only SetSetting still takes a KW; it goes away
// with that function's typed-params conversion.
type KV struct {
	Key   string
	Value any
}

// KW is a list of key/value entries.
type KW []KV

// Opt builds a key/value entry.
func Opt(key string, value any) KV { return KV{Key: key, Value: value} }
