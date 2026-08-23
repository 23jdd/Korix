// Package filter 提供可组合、与索引无关的 token 过滤器。
package filter

import (
	"strings"

	"github.com/23jdd/Koris/analysis"
)

// LowercaseFilter 使用 Unicode 规则统一小写，使 GO、Go 与 go 命中同一词项。
// Filter 会复制输入 slice，避免修改上游或调用者仍在使用的 Token。
type LowercaseFilter struct{}

func (LowercaseFilter) Filter(tokens []analysis.Token) []analysis.Token {
	result := make([]analysis.Token, len(tokens))
	copy(result, tokens)
	for i := range result {
		result[i].Term = strings.ToLower(result[i].Term)
	}
	return result
}
