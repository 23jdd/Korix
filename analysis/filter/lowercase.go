// Package filter provides reusable token filters.
package filter

import (
	"strings"

	"github.com/23jdd/Koris/analysis"
)

// LowercaseFilter applies Unicode lower casing.
type LowercaseFilter struct{}

func (LowercaseFilter) Filter(tokens []analysis.Token) []analysis.Token {
	result := make([]analysis.Token, len(tokens))
	copy(result, tokens)
	for i := range result {
		result[i].Term = strings.ToLower(result[i].Term)
	}
	return result
}
