package query_test

import (
	"errors"
	"testing"

	"github.com/23jdd/Koris/query"
)

func TestParseQueryTypes(t *testing.T) {
	tests := []string{
		`title:golang`,
		`content:"distributed system"`,
		`title:go* AND NOT content:legacy`,
		`(go OR database) AND fast`,
	}
	for _, expression := range tests {
		if parsed, err := query.Parse(expression, "content"); err != nil || parsed == nil {
			t.Errorf("Parse(%q) = %#v, %v", expression, parsed, err)
		}
	}
}

func TestParseQueryErrors(t *testing.T) {
	for _, expression := range []string{`"unclosed`, `field:`, `(go OR db`} {
		if _, err := query.Parse(expression, "content"); !errors.Is(err, query.ErrInvalidQuery) {
			t.Errorf("Parse(%q) error = %v", expression, err)
		}
	}
}
