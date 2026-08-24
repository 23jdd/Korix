package index_test

import (
	"strings"
	"testing"

	"github.com/23jdd/Koris/highlight"
	"github.com/23jdd/Koris/index"
	"github.com/23jdd/Koris/query"
	"github.com/23jdd/Koris/storage"
)

func TestHighlightAndFacet(t *testing.T) {
	idx := newTestIndex(t)
	addFixture(t, idx)
	hits, err := idx.Search(query.TermQuery{Field: "content", Term: "distributed"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	fragments, err := idx.Highlight(hits[0].ID, "content", []string{"distributed"}, highlight.DefaultOptions())
	if err != nil || len(fragments) != 1 || !strings.Contains(fragments[0].Text, "<em>distributed</em>") {
		t.Fatalf("fragments=%#v err=%v", fragments, err)
	}
	buckets, err := idx.Facet(query.MatchQuery{Field: "content", Text: "distributed"}, "category", 10)
	if err != nil || len(buckets) != 2 || buckets[0].Count != 1 || buckets[1].Count != 1 {
		t.Fatalf("buckets=%#v err=%v", buckets, err)
	}
}

func TestRebuild(t *testing.T) {
	idx := newTestIndex(t)
	addFixture(t, idx)
	if err := idx.Rebuild(); err != nil {
		t.Fatal(err)
	}
	report, err := idx.Check()
	if err != nil || !report.Valid() || report.Documents != 3 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	hits, err := idx.Search(query.TermQuery{Field: "content", Term: "golang"}, 0)
	if err != nil || len(hits) != 1 || hits[0].ID != "web" {
		t.Fatalf("hits=%#v err=%v", hits, err)
	}
}

func TestRebuildMigratesLegacyNumericIDSchema(t *testing.T) {
	store := storage.NewMemoryStore()
	if err := store.Put([]byte("meta/global"), []byte(`{"document_count":1,"total_field_length":{"content":2},"next_document_id":2}`)); err != nil {
		t.Fatal(err)
	}
	legacyDocument := []byte(`{"id":"legacy/文档","fields":{"content":"legacy data"}}`)
	if err := store.Put([]byte("document/0000000000000001"), legacyDocument); err != nil {
		t.Fatal(err)
	}
	if err := store.Put([]byte("id/bGVnYWN5L-aWh-ahow"), []byte("1")); err != nil {
		t.Fatal(err)
	}
	idx, err := index.New(store)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	if err := idx.Rebuild(); err != nil {
		t.Fatal(err)
	}
	doc, err := idx.Document("legacy/文档")
	if err != nil || doc.ID != "legacy/文档" {
		t.Fatalf("document=%#v err=%v", doc, err)
	}
	legacyIDs := store.Scan([]byte("id/"))
	defer legacyIDs.Close()
	if legacyIDs.Next() {
		t.Fatalf("legacy ID mapping was not removed: %q", legacyIDs.Key())
	}
}
