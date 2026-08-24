package index_test

import (
	"errors"
	"testing"

	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/index"
	"github.com/23jdd/Koris/query"
	"github.com/23jdd/Koris/storage"
)

func newTestIndex(t *testing.T) *index.Index {
	t.Helper()
	idx, err := index.New(storage.NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	return idx
}

func addFixture(t *testing.T, idx *index.Index) {
	t.Helper()
	docs := []document.Document{
		{ID: "go", Fields: map[string]string{"title": "Learning Go", "content": "Go is a fast programming language", "category": "book"}},
		{ID: "db", Fields: map[string]string{"title": "Distributed Database", "content": "A distributed database system is reliable", "category": "book"}},
		{ID: "web", Fields: map[string]string{"title": "Golang Web", "content": "Build a distributed system with golang", "category": "article"}},
	}
	if _, err := idx.AddBatch(docs); err != nil {
		t.Fatal(err)
	}
}

func TestAddLookupAndStats(t *testing.T) {
	idx := newTestIndex(t)
	docID, err := idx.Add(document.Document{ID: "1", Fields: map[string]string{"content": "go good go fast"}})
	if err != nil {
		t.Fatal(err)
	}
	posting, err := idx.Postings("content", "go")
	if err != nil || len(posting) != 1 {
		t.Fatalf("postings=%#v err=%v", posting, err)
	}
	if posting[0].DocID != docID || posting[0].Frequency != 2 || len(posting[0].Positions) != 2 || posting[0].Positions[1] != 2 {
		t.Fatalf("posting=%#v", posting[0])
	}
	stats, err := idx.Stats()
	if err != nil || stats.DocumentCount != 1 || stats.TotalFieldLength["content"] != 4 {
		t.Fatalf("stats=%#v err=%v", stats, err)
	}
	doc, err := idx.DocumentByID("1")
	if err != nil || doc.Fields["content"] != "go good go fast" {
		t.Fatalf("doc=%#v err=%v", doc, err)
	}
}

func TestStringIDIsUsedEndToEnd(t *testing.T) {
	idx := newTestIndex(t)
	id := "tenant/用户:42"
	addedID, err := idx.Add(document.Document{ID: id, Fields: map[string]string{"content": "string identifier"}})
	if err != nil {
		t.Fatal(err)
	}
	if addedID != id {
		t.Fatalf("Add returned %q, want %q", addedID, id)
	}
	metadata, err := idx.Metadata(id)
	if err != nil || metadata.ID != id {
		t.Fatalf("metadata=%#v err=%v", metadata, err)
	}
	postings, err := idx.Postings("content", "identifier")
	if err != nil || len(postings) != 1 || postings[0].DocID != id {
		t.Fatalf("postings=%#v err=%v", postings, err)
	}
	hits, err := idx.Search(query.TermQuery{Field: "content", Term: "identifier"}, 0)
	if err != nil || len(hits) != 1 || hits[0].ID != id {
		t.Fatalf("hits=%#v err=%v", hits, err)
	}
	ids, err := idx.AllDocumentIDs()
	if err != nil || len(ids) != 1 || ids[0] != id {
		t.Fatalf("ids=%#v err=%v", ids, err)
	}
}

func TestUpdateReplacesPostingsAndStats(t *testing.T) {
	idx := newTestIndex(t)
	firstID, _ := idx.Add(document.Document{ID: "1", Fields: map[string]string{"content": "old old term"}})
	secondID, err := idx.Add(document.Document{ID: "1", Fields: map[string]string{"content": "new term"}})
	if err != nil {
		t.Fatal(err)
	}
	if firstID != secondID {
		t.Fatalf("update changed document ID %q -> %q", firstID, secondID)
	}
	oldPostings, _ := idx.Postings("content", "old")
	newPostings, _ := idx.Postings("content", "new")
	if len(oldPostings) != 0 || len(newPostings) != 1 {
		t.Fatalf("old=%#v new=%#v", oldPostings, newPostings)
	}
	stats, _ := idx.Stats()
	if stats.DocumentCount != 1 || stats.TotalFieldLength["content"] != 2 {
		t.Fatalf("stats after update = %#v", stats)
	}
}

func TestDeleteAndConsistency(t *testing.T) {
	idx := newTestIndex(t)
	addFixture(t, idx)
	if err := idx.Delete("db"); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.DocumentByID("db"); !errors.Is(err, index.ErrDocumentMissing) {
		t.Fatalf("deleted lookup error = %v", err)
	}
	report, err := idx.Check()
	if err != nil || !report.Valid() || report.Documents != 2 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if err := idx.Delete("missing"); !errors.Is(err, index.ErrDocumentMissing) {
		t.Fatalf("missing delete error = %v", err)
	}
}

func TestBatchValidationIsAtomic(t *testing.T) {
	idx := newTestIndex(t)
	_, err := idx.AddBatch([]document.Document{
		{ID: "valid", Fields: map[string]string{"content": "one"}},
		{ID: "", Fields: map[string]string{"content": "bad"}},
	})
	if !errors.Is(err, index.ErrInvalidDocument) {
		t.Fatalf("batch error = %v", err)
	}
	stats, _ := idx.Stats()
	if stats.DocumentCount != 0 {
		t.Fatalf("partial batch committed: %#v", stats)
	}
}

func TestTermAndMatchQueries(t *testing.T) {
	idx := newTestIndex(t)
	addFixture(t, idx)
	hits, err := idx.Search(query.TermQuery{Field: "content", Term: "distributed"}, 10)
	if err != nil || len(hits) != 2 {
		t.Fatalf("term hits=%#v err=%v", hits, err)
	}
	hits, err = idx.Search(query.MatchQuery{Field: "content", Text: "distributed golang", Operator: query.AND}, 10)
	if err != nil || len(hits) != 1 || hits[0].ID != "web" {
		t.Fatalf("AND hits=%#v err=%v", hits, err)
	}
}

func TestBooleanPhrasePrefixAndFuzzyQueries(t *testing.T) {
	idx := newTestIndex(t)
	addFixture(t, idx)

	phrase, err := idx.Search(query.PhraseQuery{Field: "content", Text: "distributed system"}, 0)
	if err != nil || len(phrase) != 1 || phrase[0].ID != "web" {
		t.Fatalf("phrase=%#v err=%v", phrase, err)
	}
	prefix, err := idx.Search(query.PrefixQuery{Field: "title", Prefix: "go"}, 0)
	if err != nil || len(prefix) != 2 {
		t.Fatalf("prefix=%#v err=%v", prefix, err)
	}
	fuzzy, err := idx.Search(query.FuzzyQuery{Field: "content", Term: "gloang", MaxEdits: 2}, 0)
	if err != nil || len(fuzzy) != 1 || fuzzy[0].ID != "web" {
		t.Fatalf("fuzzy=%#v err=%v", fuzzy, err)
	}
	boolean, err := idx.Search(query.BooleanQuery{
		Must:    []query.Query{query.TermQuery{Field: "content", Term: "distributed"}},
		MustNot: []query.Query{query.TermQuery{Field: "content", Term: "database"}},
	}, 0)
	if err != nil || len(boolean) != 1 || boolean[0].ID != "web" {
		t.Fatalf("boolean=%#v err=%v", boolean, err)
	}
}

func TestPhrasePreservesStopwordPositionGap(t *testing.T) {
	idx := newTestIndex(t)
	_, _ = idx.Add(document.Document{ID: "gap", Fields: map[string]string{"content": "go is fast"}})
	_, _ = idx.Add(document.Document{ID: "adjacent", Fields: map[string]string{"content": "go fast"}})
	hits, err := idx.Search(query.PhraseQuery{Field: "content", Text: "go is fast"}, 0)
	if err != nil || len(hits) != 1 || hits[0].ID != "gap" {
		t.Fatalf("gap phrase hits=%#v err=%v", hits, err)
	}
}

func TestSearchString(t *testing.T) {
	idx := newTestIndex(t)
	addFixture(t, idx)
	hits, err := idx.SearchString(`content:"distributed system" OR title:database`, "content", 10)
	if err != nil || len(hits) != 2 {
		t.Fatalf("hits=%#v err=%v", hits, err)
	}
	hits, err = idx.SearchString(`content:distributed AND NOT content:database`, "content", 10)
	if err != nil || len(hits) != 1 || hits[0].ID != "web" {
		t.Fatalf("NOT hits=%#v err=%v", hits, err)
	}
}
