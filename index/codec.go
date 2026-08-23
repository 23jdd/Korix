package index

import (
	"encoding/json"
	"fmt"

	"github.com/23jdd/Koris/storage"
)

func putJSON(tx storage.Tx, key []byte, value any) error {
	// JSON 用于低频 metadata，优先考虑可检查与 schema 演进；高频 posting 使用
	// inverted 二进制 codec 以降低空间和解码成本。
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return tx.Put(key, data)
}

func getJSON(reader storage.Reader, key []byte, target any) error {
	data, err := reader.Get(key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		// 包装 ErrCorruptIndex，同时保留 key 与原始 JSON 错误供恢复诊断。
		return fmt.Errorf("%w: %s: %v", ErrCorruptIndex, key, err)
	}
	return nil
}
