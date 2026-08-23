package index_test

import (
	"strings"
	"testing"

	"github.com/23jdd/Koris/highlight"
	"github.com/23jdd/Koris/query"
)

func TestHighlightAndFacet(t *testing.T) {
	idx := newTestIndex(t)
	addFixture(t, idx)
	hits, err := idx.Search(query.TermQuery{Field: "content", Term: "distributed"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	fragments, err := idx.Highlight(hits[0].DocID, "content", []string{"distributed"}, highlight.DefaultOptions())
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
