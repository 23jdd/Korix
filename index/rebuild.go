package index

import (
	"encoding/json"
	"fmt"

	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/storage"
)

// Rebuild reconstructs all derived index data from stored documents in one
// transaction. It is the recovery path for analyzer migrations or a failed
// consistency audit. Internal document IDs may change.
func (i *Index) Rebuild() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.store.Transaction(func(tx storage.Tx) error {
		documents := make([]document.Document, 0)
		iterator := tx.Scan(docPrefix)
		for iterator.Next() {
			var doc document.Document
			if err := json.Unmarshal(iterator.Value(), &doc); err != nil {
				iterator.Close()
				return fmt.Errorf("%w: document: %v", ErrCorruptIndex, err)
			}
			documents = append(documents, doc)
		}
		if err := iterator.Error(); err != nil {
			iterator.Close()
			return err
		}
		iterator.Close()
		for _, prefix := range [][]byte{docPrefix, docMetaPrefix, idPrefix, postingPrefix, termPrefix} {
			if err := deletePrefix(tx, prefix); err != nil {
				return err
			}
		}
		global := GlobalStats{TotalFieldLength: make(map[string]uint64), NextDocumentID: 1}
		for _, doc := range documents {
			if _, err := i.addInTransaction(tx, &global, doc); err != nil {
				return err
			}
		}
		return putJSON(tx, globalMetaKey, global)
	})
}

func deletePrefix(tx storage.Tx, prefix []byte) error {
	iterator := tx.Scan(prefix)
	keys := make([][]byte, 0)
	for iterator.Next() {
		keys = append(keys, iterator.Key())
	}
	if err := iterator.Error(); err != nil {
		iterator.Close()
		return err
	}
	iterator.Close()
	for _, key := range keys {
		if err := tx.Delete(key); err != nil {
			return err
		}
	}
	return nil
}
