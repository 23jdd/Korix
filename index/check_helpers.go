package index

import (
	"encoding/json"
	"fmt"
	"strings"
)

func splitTermKey(key []byte) []string {
	// term key 的合法结构应恰好是 term/{field}/{term}，Check 会验证结果长度。
	value := strings.TrimPrefix(string(key), "term/")
	return strings.Split(value, "/")
}

func decodeJSONValue(data []byte, target any) error {
	// 与 getJSON 不同，此处 value 已来自 iterator，因此无需再次按 key 读取。
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("%w: %v", ErrCorruptIndex, err)
	}
	return nil
}
