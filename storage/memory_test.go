package storage_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/23jdd/Koris/storage"
)

func TestMemoryStoreTransactionRollback(t *testing.T) {
	store := storage.NewMemoryStore()
	wantErr := errors.New("rollback")
	err := store.Transaction(func(tx storage.Tx) error {
		if err := tx.Put([]byte("key"), []byte("value")); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("transaction error = %v", err)
	}
	if _, err := store.Get([]byte("key")); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("rolled back value remained: %v", err)
	}
}

func TestMemoryStoreCopiesAndSorts(t *testing.T) {
	store := storage.NewMemoryStore()
	value := []byte("one")
	_ = store.Put([]byte("p/2"), value)
	value[0] = 'X'
	_ = store.Put([]byte("p/1"), []byte("two"))
	got, _ := store.Get([]byte("p/2"))
	if string(got) != "one" {
		t.Fatalf("stored slice was aliased: %q", got)
	}
	iterator := store.Scan([]byte("p/"))
	defer iterator.Close()
	var keys []string
	for iterator.Next() {
		keys = append(keys, string(iterator.Key()))
	}
	if !reflect.DeepEqual(keys, []string{"p/1", "p/2"}) {
		t.Fatalf("keys = %v", keys)
	}
}

func TestMemoryStoreClosed(t *testing.T) {
	store := storage.NewMemoryStore()
	_ = store.Close()
	if _, err := store.Get([]byte("x")); !errors.Is(err, storage.ErrClosed) {
		t.Fatalf("Get after close = %v", err)
	}
}
