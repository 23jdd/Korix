// Package standard 构造 Koris 默认的通用文本分析器。
package standard

import (
	"github.com/23jdd/Koris/analysis"
	filterpkg "github.com/23jdd/Koris/analysis/filter"
	"github.com/23jdd/Koris/analysis/tokenizer"
)

// New 返回“StandardTokenizer → Lowercase → English StopWord”分析链。
// 写入与查询必须复用同一配置，否则查询生成的 term 可能与索引中的 term 不一致。
func New() analysis.Analyzer {
	return analysis.PipelineAnalyzer{
		Tokenizer: tokenizer.StandardTokenizer{},
		Filters: []analysis.TokenFilter{
			filterpkg.LowercaseFilter{},
			filterpkg.NewStopWordFilter(filterpkg.EnglishStopWords),
		},
	}
}
