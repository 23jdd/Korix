// Package tokenizer provides built-in Unicode-aware tokenizers.
package tokenizer

import (
	"unicode"
	"unicode/utf8"

	"github.com/23jdd/Koris/analysis"
)

// WhitespaceTokenizer splits on Unicode whitespace and preserves punctuation.
type WhitespaceTokenizer struct{}

func (WhitespaceTokenizer) Tokenize(text string) []analysis.Token {
	var tokens []analysis.Token
	start := -1
	position := uint32(0)
	for offset, r := range text {
		if unicode.IsSpace(r) {
			if start >= 0 {
				tokens = appendToken(tokens, text, start, offset, position, "word")
				position++
				start = -1
			}
		} else if start < 0 {
			start = offset
		}
	}
	if start >= 0 {
		tokens = appendToken(tokens, text, start, len(text), position, "word")
	}
	return tokens
}

func appendToken(tokens []analysis.Token, text string, start, end int, position uint32, kind string) []analysis.Token {
	if start < 0 || end <= start || !utf8.ValidString(text[start:end]) {
		return tokens
	}
	return append(tokens, analysis.Token{
		Term: text[start:end], Position: position, StartOffset: uint32(start), EndOffset: uint32(end), Type: kind,
	})
}
