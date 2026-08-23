package main

import (
	"fmt"

	koris "github.com/23jdd/Koris"
	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/highlight"
	"github.com/23jdd/Koris/query"
)

func main() {
	idx, err := koris.New()
	if err != nil {
		panic(err)
	}
	defer idx.Close()
      
	_, err = idx.AddBatch([]document.Document{
		{ID: "go", Fields: map[string]string{"title": "Learning Go", "content": "Go is fast", "category": "book"}},
		{ID: "systems", Fields: map[string]string{"title": "Systems", "content": "A distributed system written in Golang", "category": "article"}},
	})
	if err != nil {
		panic(err)
	}

	hits, err := idx.Search(query.MatchQuery{Field: "content", Text: "distributed golang", Operator: query.AND}, 10)
	if err != nil {
		panic(err)
	}
	for _, hit := range hits {
		fragments, err := idx.Highlight(hit.DocID, "content", []string{"distributed", "golang"}, highlight.DefaultOptions())
		if err != nil {
			panic(err)
		}
		fmt.Printf("id=%s score=%.3f snippet=%s\n", hit.ID, hit.Score, fragments[0].Text)
	}
}
