// Package standard constructs the default general-purpose analyzer.
package standard

import (
	"github.com/23jdd/Koris/analysis"
	filterpkg "github.com/23jdd/Koris/analysis/filter"
	"github.com/23jdd/Koris/analysis/tokenizer"
)

// New returns a Unicode-aware tokenizer followed by lower-case and stop-word
// filters.
func New() analysis.Analyzer {
	return analysis.PipelineAnalyzer{
		Tokenizer: tokenizer.StandardTokenizer{},
		Filters: []analysis.TokenFilter{
			filterpkg.LowercaseFilter{},
			filterpkg.NewStopWordFilter(filterpkg.EnglishStopWords),
		},
	}
}
