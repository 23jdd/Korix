package analysis

// PipelineAnalyzer composes one tokenizer and zero or more filters.
type PipelineAnalyzer struct {
	Tokenizer Tokenizer
	Filters   []TokenFilter
}

func (a PipelineAnalyzer) Analyze(text string) []Token {
	if a.Tokenizer == nil {
		return nil
	}
	tokens := a.Tokenizer.Tokenize(text)
	for _, filter := range a.Filters {
		if filter != nil {
			tokens = filter.Filter(tokens)
		}
	}
	return tokens
}

func (a PipelineAnalyzer) Stream(text string) TokenStream {
	return NewTokenStream(a, text)
}
