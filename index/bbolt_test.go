package index_test

import (
	"testing"

	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/index"
	"github.com/23jdd/Koris/query"
	"github.com/23jdd/Koris/storage"
)

func TestPersistentIndexReopen(t *testing.T) {
	path := t.TempDir() + "/index.db"
	store, err := storage.OpenBbolt(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := index.New(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Add(document.Document{ID: "1", Fields: map[string]string{"content": "persistent search"}}); err != nil {
		t.Fatal(err)
	}
	if err := idx.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = storage.OpenBbolt(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	idx, err = index.New(store)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	hits, err := idx.Search(query.TermQuery{Field: "content", Term: "persistent"}, 0)
	if err != nil || len(hits) != 1 || hits[0].ID != "1" {
		t.Fatalf("hits=%#v err=%v", hits, err)
	}
}
