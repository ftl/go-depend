package gen

import "encoding/json"

// Marshal pretends to be generated code.
func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
