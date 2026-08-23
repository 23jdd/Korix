package analysis

// PipelineAnalyzer 把一个 Tokenizer 与零到多个 TokenFilter 串成分析链。
// Filters 严格按切片顺序执行，因此 Lowercase 通常应放在大小写敏感的 StopWord
// 之前。结构体只保存无状态组件，可作为字段 Analyzer 长期复用。
type PipelineAnalyzer struct {
	Tokenizer Tokenizer
	Filters   []TokenFilter
}

func (a PipelineAnalyzer) Analyze(text string) []Token {
	if a.Tokenizer == nil {
		return nil
	}
	tokens := a.Tokenizer.Tokenize(text)
	// 每个 filter 都消费上一步的完整输出。这样设计比把 filter 写死在 tokenizer
	// 中更容易组合，也允许索引字段使用不同的规范化策略。
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
