package inverted_test

import (
	"reflect"
	"testing"

	"github.com/23jdd/Koris/inverted"
)

func TestPostingCodecRoundTrip(t *testing.T) {
	want := inverted.NewPosting(42, []uint32{1000, 1, 8, 9})
	encoded := inverted.EncodePosting(want)
	got, err := inverted.DecodePosting(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if len(encoded) >= 8+4+len(want.Positions)*4 {
		t.Fatalf("posting not compact: %d bytes", len(encoded))
	}
}

func TestPostingCodecRejectsCorruption(t *testing.T) {
	if _, err := inverted.DecodePosting([]byte{0x80}); err == nil {
		t.Fatal("corrupt posting accepted")
	}
}

func TestPostingIteratorSkipTo(t *testing.T) {
	iterator := inverted.NewPostingIterator([]inverted.Posting{{DocID: 9}, {DocID: 1}, {DocID: 5}})
	if !iterator.SkipTo(5) || iterator.Posting().DocID != 5 {
		t.Fatalf("SkipTo = %#v", iterator.Posting())
	}
	if !iterator.Next() || iterator.Posting().DocID != 9 {
		t.Fatal("Next after SkipTo failed")
	}
}

func TestSkipListIterator(t *testing.T) {
	postings := make([]inverted.Posting, 100)
	for i := range postings {
		postings[i].DocID = uint64(i * 2)
	}
	iterator := inverted.NewSkipListIterator(postings)
	if !iterator.SkipTo(73) || iterator.Posting().DocID != 74 {
		t.Fatalf("SkipTo(73) = %#v", iterator.Posting())
	}
}
