package iterator_test

import (
	"reflect"
	"testing"

	iteratorpkg "github.com/23jdd/Koris/iterator"
)

func TestStringDocumentIDIterator(t *testing.T) {
	iterator := iteratorpkg.New([]string{"文档-2", "doc/1", "alpha"})
	if !iterator.SkipTo("doc") || iterator.DocID() != "doc/1" {
		t.Fatalf("SkipTo returned %q", iterator.DocID())
	}
	var remaining []string
	for iterator.Next() {
		remaining = append(remaining, iterator.DocID())
	}
	if !reflect.DeepEqual(remaining, []string{"文档-2"}) {
		t.Fatalf("remaining IDs = %#v", remaining)
	}
}
