package filter_test

import (
	"testing"

	"github.com/23jdd/Koris/analysis"
	filterpkg "github.com/23jdd/Koris/analysis/filter"
)

func TestFilters(t *testing.T) {
	tokens := []analysis.Token{{Term: "RUNNING"}, {Term: "is"}, {Term: "FAST"}}
	tokens = (filterpkg.LowercaseFilter{}).Filter(tokens)
	tokens = filterpkg.NewStopWordFilter([]string{"is"}).Filter(tokens)
	tokens = (filterpkg.StemmerFilter{}).Filter(tokens)
	if len(tokens) != 2 || tokens[0].Term != "run" || tokens[1].Term != "fast" {
		t.Fatalf("unexpected filter output: %#v", tokens)
	}
}

func TestFilterDoesNotMutateInput(t *testing.T) {
	input := []analysis.Token{{Term: "GO"}}
	_ = (filterpkg.LowercaseFilter{}).Filter(input)
	if input[0].Term != "GO" {
		t.Fatal("filter mutated caller input")
	}
}
