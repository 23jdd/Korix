package storage_test

import (
	"testing"

	"github.com/23jdd/Koris/storage"
)

func TestBboltPersistence(t *testing.T) {
	path := t.TempDir() + "/koris.db"
	store, err := storage.OpenBbolt(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put([]byte("hello"), []byte("world")); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = storage.OpenBbolt(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	value, err := store.Get([]byte("hello"))
	if err != nil || string(value) != "world" {
		t.Fatalf("reopened value=%q err=%v", value, err)
	}
}
