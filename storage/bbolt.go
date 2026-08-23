package storage

import (
	"bytes"
	"errors"
	"os"
	"time"

	bbolt "go.etcd.io/bbolt"
)

var bucketName = []byte("koris")

// BboltStore persists all keys in one ordered bucket. Values and iterators are
// copied before the bbolt read transaction closes.
type BboltStore struct{ db *bbolt.DB }

func OpenBbolt(path string, mode uint32) (*BboltStore, error) {
	db, err := bbolt.Open(path, fileMode(mode), &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	store := &BboltStore{db: db}
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, createErr := tx.CreateBucketIfNotExists(bucketName)
		return createErr
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func fileMode(mode uint32) os.FileMode {
	if mode == 0 {
		mode = 0o600
	}
	return os.FileMode(mode)
}

func (s *BboltStore) Get(key []byte) ([]byte, error) {
	if s == nil || s.db == nil {
		return nil, ErrClosed
	}
	var result []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		value := tx.Bucket(bucketName).Get(key)
		if value == nil {
			return ErrNotFound
		}
		result = cloneBytes(value)
		return nil
	})
	return result, normalizeBboltError(err)
}

func (s *BboltStore) Put(key, value []byte) error {
	return s.Transaction(func(tx Tx) error { return tx.Put(key, value) })
}

func (s *BboltStore) Delete(key []byte) error {
	return s.Transaction(func(tx Tx) error { return tx.Delete(key) })
}

func (s *BboltStore) Scan(prefix []byte) Iterator {
	if s == nil || s.db == nil {
		return errorIterator(ErrClosed)
	}
	var entries []entry
	err := s.db.View(func(tx *bbolt.Tx) error {
		entries = scanBucket(tx.Bucket(bucketName), prefix)
		return nil
	})
	if err != nil {
		return errorIterator(normalizeBboltError(err))
	}
	return newSliceIterator(entries)
}

func (s *BboltStore) Transaction(fn func(Tx) error) error {
	if s == nil || s.db == nil {
		return ErrClosed
	}
	return normalizeBboltError(s.db.Update(func(tx *bbolt.Tx) error {
		return fn(&bboltTx{bucket: tx.Bucket(bucketName)})
	}))
}

func (s *BboltStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

type bboltTx struct{ bucket *bbolt.Bucket }

func (tx *bboltTx) Get(key []byte) ([]byte, error) {
	value := tx.bucket.Get(key)
	if value == nil {
		return nil, ErrNotFound
	}
	return cloneBytes(value), nil
}

func (tx *bboltTx) Put(key, value []byte) error { return tx.bucket.Put(key, value) }
func (tx *bboltTx) Delete(key []byte) error     { return tx.bucket.Delete(key) }
func (tx *bboltTx) Scan(prefix []byte) Iterator {
	return newSliceIterator(scanBucket(tx.bucket, prefix))
}

func scanBucket(bucket *bbolt.Bucket, prefix []byte) []entry {
	entries := make([]entry, 0)
	cursor := bucket.Cursor()
	for key, value := cursor.Seek(prefix); key != nil && bytes.HasPrefix(key, prefix); key, value = cursor.Next() {
		entries = append(entries, entry{key: cloneBytes(key), value: cloneBytes(value)})
	}
	return entries
}

func normalizeBboltError(err error) error {
	if errors.Is(err, bbolt.ErrDatabaseNotOpen) {
		return ErrClosed
	}
	return err
}
