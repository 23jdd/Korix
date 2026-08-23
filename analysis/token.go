// Package analysis contains the text analysis framework used by indexes and
// queries. It deliberately has no dependency on an index implementation.
package analysis

// Token describes a term and its location in the original UTF-8 text.
// Offsets are byte offsets, which makes slicing the original Go string safe.
type Token struct {
	Term        string `json:"term"`
	Position    uint32 `json:"position"`
	StartOffset uint32 `json:"start_offset"`
	EndOffset   uint32 `json:"end_offset"`
	Type        string `json:"type,omitempty"`
}

// Tokenizer is the first stage of an analysis pipeline.
type Tokenizer interface {
	Tokenize(text string) []Token
}

// TokenFilter transforms a complete token sequence.
type TokenFilter interface {
	Filter(tokens []Token) []Token
}

// Analyzer converts text into indexable tokens.
type Analyzer interface {
	Analyze(text string) []Token
}
