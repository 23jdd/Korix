// Package tokenizer 提供内置的 Unicode 感知分词器。
package tokenizer

import (
	"unicode"
	"unicode/utf8"

	"github.com/23jdd/Koris/analysis"
)

// WhitespaceTokenizer 按 Unicode 空白字符切分，标点仍作为 term 的一部分保留。
// 它适合日志、预分词文本等“空白就是唯一边界”的数据；自然语言通常应使用
// StandardTokenizer。
type WhitespaceTokenizer struct{}

func (WhitespaceTokenizer) Tokenize(text string) []analysis.Token {
	var tokens []analysis.Token
	start := -1
	position := uint32(0)
	// range 返回 UTF-8 字节偏移，因此无需二次换算即可填写 Token offset。
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
	// 所有 tokenizer 共用这个出口，集中保证 offset 有序且切片是合法 UTF-8。
	if start < 0 || end <= start || !utf8.ValidString(text[start:end]) {
		return tokens
	}
	return append(tokens, analysis.Token{
		Term: text[start:end], Position: position, StartOffset: uint32(start), EndOffset: uint32(end), Type: kind,
	})
}
