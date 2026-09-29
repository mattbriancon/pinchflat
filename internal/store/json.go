package store

// Hand-written W0 infrastructure.

import (
	"bytes"
	"encoding/json"
)

// DecodeJSON decodes like Jason.decode!/1 into generic values, keeping
// numbers as json.Number so integers stay integers.
func DecodeJSON(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return dec.Decode(v)
}
