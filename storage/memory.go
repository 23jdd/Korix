package storage

import (
	"bytes"
	"sort"
	"sync"
)

// MemoryStore is a transactional in-memory implementation. Write
// transactions use copy-on-write, providing rollback when fn returns an error.
type MemoryStore struct {
	mu     sync.RWMutex
	data   map[string][]byte
	closed bool
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{data: make(map[string][]byte)} }

func (s *MemoryStore) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrClosed
	}
	value, found := s.data[string(key)]
	if !found {
		return nil, ErrNotFound
	}
	return cloneBytes(value), nil
}

func (s *MemoryStore) Put(key, value []byte) error {
	return s.Transaction(func(tx Tx) error { return tx.Put(key, value) })
}

func (s *MemoryStore) Delete(key []byte) error {
	return s.Transaction(func(tx Tx) error { return tx.Delete(key) })
}

func (s *MemoryStore) Scan(prefix []byte) Iterator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return errorIterator(ErrClosed)
	}
	return scanMap(s.data, prefix)
}

func (s *MemoryStore) Transaction(fn func(Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	working := cloneMap(s.data)
	if err := fn(&memoryTx{data: working}); err != nil {
		return err
	}
	s.data = working
	return nil
}

func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

type memoryTx struct{ data map[string][]byte }

func (tx *memoryTx) Get(key []byte) ([]byte, error) {
	value, found := tx.data[string(key)]
	if !found {
		return nil, ErrNotFound
	}
	return cloneBytes(value), nil
}

func (tx *memoryTx) Put(key, value []byte) error {
	tx.data[string(key)] = cloneBytes(value)
	return nil
}

func (tx *memoryTx) Delete(key []byte) error {
	delete(tx.data, string(key))
	return nil
}

func (tx *memoryTx) Scan(prefix []byte) Iterator { return scanMap(tx.data, prefix) }

func scanMap(data map[string][]byte, prefix []byte) Iterator {
	entries := make([]entry, 0)
	for key, value := range data {
		if bytes.HasPrefix([]byte(key), prefix) {
			entries = append(entries, entry{key: []byte(key), value: cloneBytes(value)})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return bytes.Compare(entries[i].key, entries[j].key) < 0 })
	return newSliceIterator(entries)
}

func cloneMap(data map[string][]byte) map[string][]byte {
	clone := make(map[string][]byte, len(data))
	for key, value := range data {
		clone[key] = cloneBytes(value)
	}
	return clone
}
