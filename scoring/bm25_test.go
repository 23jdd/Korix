package scoring_test

import (
	"testing"

	"github.com/23jdd/Koris/scoring"
)

func TestBM25Properties(t *testing.T) {
	bm25 := scoring.DefaultBM25()
	lowTF := bm25.Score(1, 2, 100, 10, 10)
	highTF := bm25.Score(4, 2, 100, 10, 10)
	common := bm25.Score(1, 90, 100, 10, 10)
	if !(highTF > lowTF && lowTF > common && common > 0) {
		t.Fatalf("unexpected scores high=%f low=%f common=%f", highTF, lowTF, common)
	}
}
