package analysis

// TokenStream supports incremental consumption and reuse for large inputs.
type TokenStream interface {
	Next() bool
	Token() Token
	Reset(text string)
}

// StreamAnalyzer optionally exposes streaming analysis.
type StreamAnalyzer interface {
	Analyzer
	Stream(text string) TokenStream
}

type sliceTokenStream struct {
	analyzer Analyzer
	tokens   []Token
	current  int
}

// NewTokenStream creates a reusable stream backed by an Analyzer. Tokenizers
// may internally stream later without changing callers.
func NewTokenStream(analyzer Analyzer, text string) TokenStream {
	s := &sliceTokenStream{analyzer: analyzer}
	s.Reset(text)
	return s
}

func (s *sliceTokenStream) Next() bool {
	if s.current >= len(s.tokens) {
		return false
	}
	s.current++
	return true
}

func (s *sliceTokenStream) Token() Token {
	if s.current == 0 || s.current > len(s.tokens) {
		return Token{}
	}
	return s.tokens[s.current-1]
}

func (s *sliceTokenStream) Reset(text string) {
	s.tokens = s.analyzer.Analyze(text)
	s.current = 0
}
