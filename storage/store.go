// Package storage defines the persistence boundary used by the index.
package storage

import "errors"

var (
	ErrNotFound = errors.New("storage: key not found")
	ErrClosed   = errors.New("storage: store closed")
)

// Iterator scans an immutable, lexicographically ordered key snapshot.
type Iterator interface {
	Next() bool
	Valid() bool
	Key() []byte
	Value() []byte
	Error() error
	Close() error
}

// Reader is shared by Store and transaction views.
type Reader interface {
	Get(key []byte) ([]byte, error)
	Scan(prefix []byte) Iterator
}

// Tx is a read/write atomic transaction.
type Tx interface {
	Reader
	Put(key, value []byte) error
	Delete(key []byte) error
}

// Store isolates persistence details from all search logic.
type Store interface {
	Reader
	Put(key, value []byte) error
	Delete(key []byte) error
	Transaction(fn func(Tx) error) error
	Close() error
}
