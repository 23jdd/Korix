package index_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/query"
)

func TestConcurrentReadAndWrite(t *testing.T) {
	idx := newTestIndex(t)
	if _, err := idx.Add(document.Document{ID: "seed", Fields: map[string]string{"content": "search seed"}}); err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	for writer := 0; writer < 2; writer++ {
		wait.Add(1)
		go func(writer int) {
			defer wait.Done()
			for n := 0; n < 25; n++ {
				_, err := idx.Add(document.Document{ID: fmt.Sprintf("%d-%d", writer, n), Fields: map[string]string{"content": "concurrent search"}})
				if err != nil {
					t.Errorf("Add: %v", err)
					return
				}
			}
		}(writer)
	}
	for reader := 0; reader < 4; reader++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for n := 0; n < 25; n++ {
				if _, err := idx.Search(query.TermQuery{Field: "content", Term: "search"}, 10); err != nil {
					t.Errorf("Search: %v", err)
					return
				}
			}
		}()
	}
	wait.Wait()
	report, err := idx.Check()
	if err != nil || !report.Valid() || report.Documents != 51 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
}
