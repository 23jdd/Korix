// Package Koris provides the compact public facade for the embedded search
// engine. Advanced callers can use the focused subpackages directly.
package Koris

import (
	"github.com/23jdd/Koris/document"
	indexpkg "github.com/23jdd/Koris/index"
	"github.com/23jdd/Koris/storage"
)

// New creates a transient in-memory index.
func New(options ...indexpkg.Option) (*indexpkg.Index, error) {
	return indexpkg.New(storage.NewMemoryStore(), options...)
}

// Open creates or opens a persistent bbolt-backed index.
func Open(path string, options ...indexpkg.Option) (*indexpkg.Index, error) {
	store, err := storage.OpenBbolt(path, 0o600)
	if err != nil {
		return nil, err
	}
	idx, err := indexpkg.New(store, options...)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return idx, nil
}

// Add is a convenience helper for simple field maps.
func Add(idx *indexpkg.Index, id string, fields map[string]string) (uint64, error) {
	return idx.Add(document.Document{ID: id, Fields: fields})
}
