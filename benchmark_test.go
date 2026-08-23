package Koris_test

import (
	"fmt"
	"testing"

	Koris "github.com/23jdd/Koris"
	"github.com/23jdd/Koris/analysis/standard"
	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/inverted"
	"github.com/23jdd/Koris/query"
)

func BenchmarkStandardAnalyzer(b *testing.B) {
	analyzer := standard.New()
	text := "Go is a fast programming language for reliable distributed database systems."
	b.ReportAllocs()
	for b.Loop() {
		_ = analyzer.Analyze(text)
	}
}

func BenchmarkPostingCodec(b *testing.B) {
	posting := inverted.NewPosting(123456, []uint32{1, 4, 9, 15, 30, 100})
	b.ReportAllocs()
	for b.Loop() {
		encoded := inverted.EncodePosting(posting)
		if _, err := inverted.DecodePosting(encoded); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIndexAdd(b *testing.B) {
	idx, err := Koris.New()
	if err != nil {
		b.Fatal(err)
	}
	defer idx.Close()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		_, err := idx.Add(document.Document{ID: fmt.Sprintf("doc-%d", i), Fields: map[string]string{
			"content": "Go provides fast reliable embedded full text search",
		}})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTermSearch1000Documents(b *testing.B) {
	idx, err := Koris.New()
	if err != nil {
		b.Fatal(err)
	}
	defer idx.Close()
	docs := make([]document.Document, 1000)
	for i := range docs {
		docs[i] = document.Document{ID: fmt.Sprintf("doc-%d", i), Fields: map[string]string{
			"content": fmt.Sprintf("embedded search engine document number %d", i),
		}}
	}
	if _, err := idx.AddBatch(docs); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := idx.Search(query.TermQuery{Field: "content", Term: "search"}, 10); err != nil {
			b.Fatal(err)
		}
	}
}
