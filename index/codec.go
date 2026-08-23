package index

import (
	"encoding/json"
	"fmt"

	"github.com/23jdd/Koris/storage"
)

func putJSON(tx storage.Tx, key []byte, value any) error {
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
		return fmt.Errorf("%w: %s: %v", ErrCorruptIndex, key, err)
	}
	return nil
}
