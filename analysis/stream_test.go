package analysis_test

import (
	"testing"

	"github.com/23jdd/Koris/analysis"
	"github.com/23jdd/Koris/analysis/standard"
)

func TestTokenStreamReset(t *testing.T) {
	stream := analysis.NewTokenStream(standard.New(), "Go is fast")
	var terms []string
	for stream.Next() {
		terms = append(terms, stream.Token().Term)
	}
	if len(terms) != 2 || terms[0] != "go" || terms[1] != "fast" {
		t.Fatalf("terms = %v", terms)
	}
	stream.Reset("Simple")
	if !stream.Next() || stream.Token().Term != "simple" || stream.Next() {
		t.Fatal("reset did not replace stream")
	}
}
