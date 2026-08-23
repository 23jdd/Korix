package index

import (
	"encoding/json"
	"fmt"
	"strings"
)

func splitTermKey(key []byte) []string {
	value := strings.TrimPrefix(string(key), "term/")
	return strings.Split(value, "/")
}

func decodeJSONValue(data []byte, target any) error {
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("%w: %v", ErrCorruptIndex, err)
	}
	return nil
}
