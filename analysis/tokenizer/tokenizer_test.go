package tokenizer_test

import (
	"testing"

	"github.com/23jdd/Koris/analysis/tokenizer"
)

func TestWhitespaceTokenizerOffsets(t *testing.T) {
	tokens := (tokenizer.WhitespaceTokenizer{}).Tokenize("Go\t世界 fast")
	want := []string{"Go", "世界", "fast"}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(want))
	}
	for i := range want {
		if tokens[i].Term != want[i] || tokens[i].Position != uint32(i) {
			t.Fatalf("token %d = %#v", i, tokens[i])
		}
	}
	if tokens[1].StartOffset != 3 || tokens[1].EndOffset != 9 {
		t.Fatalf("Unicode byte offsets = %d:%d", tokens[1].StartOffset, tokens[1].EndOffset)
	}
}

func TestSimpleTokenizer(t *testing.T) {
	tokens := (tokenizer.SimpleTokenizer{}).Tokenize("Hello, Go123!")
	if len(tokens) != 2 || tokens[0].Term != "Hello" || tokens[1].Term != "Go123" {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
}

func TestStandardTokenizerTypes(t *testing.T) {
	tokens := (tokenizer.StandardTokenizer{}).Tokenize("mail me@example.com at https://go.dev, price 12.5!")
	types := make(map[string]string)
	for _, token := range tokens {
		types[token.Term] = token.Type
	}
	if types["me@example.com"] != "email" {
		t.Fatalf("email not recognized: %#v", tokens)
	}
	if types["https://go.dev"] != "url" {
		t.Fatalf("URL not recognized: %#v", tokens)
	}
	if types["12.5"] != "number" {
		t.Fatalf("number not recognized: %#v", tokens)
	}
}

func TestChineseTokenizerDictionary(t *testing.T) {
	tokenizer := tokenizer.NewChineseTokenizer([]string{"中华人民共和国", "人民", "共和国"}).WithMode(tokenizer.BidirectionalMaximumMatching)
	tokens := tokenizer.Tokenize("中华人民共和国")
	if len(tokens) != 1 || tokens[0].Term != "中华人民共和国" {
		t.Fatalf("unexpected Chinese segmentation: %#v", tokens)
	}
}

func TestChineseTokenizerUnknownFallback(t *testing.T) {
	tokens := tokenizer.NewChineseTokenizer(nil).Tokenize("搜索AI")
	terms := []string{"搜", "索", "AI"}
	if len(tokens) != len(terms) {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
	for i := range terms {
		if tokens[i].Term != terms[i] {
			t.Fatalf("token %d = %q", i, tokens[i].Term)
		}
	}
}

func TestHMMTokenizerViterbi(t *testing.T) {
	model := tokenizer.DefaultHMMModel()
	tokens := tokenizer.NewHMMTokenizer(model).Tokenize("中文分词 AI")
	if len(tokens) < 2 {
		t.Fatalf("unexpected HMM tokens: %#v", tokens)
	}
	joined := ""
	for _, token := range tokens[:len(tokens)-1] {
		joined += token.Term
	}
	if joined != "中文分词" || tokens[len(tokens)-1].Term != "AI" {
		t.Fatalf("HMM lost input: %#v", tokens)
	}
}
