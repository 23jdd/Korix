package query

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var ErrInvalidQuery = errors.New("query: invalid expression")

// Parse supports field:value, quoted phrases, trailing prefix wildcards,
// parentheses and the AND/OR/NOT Boolean operators. Adjacent clauses imply OR.
func Parse(expression, defaultField string) (Query, error) {
	lexer, err := lex(expression)
	if err != nil {
		return nil, err
	}
	if len(lexer) == 0 {
		return nil, nil
	}
	parser := expressionParser{tokens: lexer, defaultField: defaultField}
	result, err := parser.parseOr()
	if err != nil {
		return nil, err
	}
	if parser.index != len(parser.tokens) {
		return nil, fmt.Errorf("%w near %q", ErrInvalidQuery, parser.tokens[parser.index].text)
	}
	return result, nil
}

type lexToken struct {
	kind tokenKind
	text string
}

type tokenKind uint8

const (
	atomToken tokenKind = iota
	andToken
	orToken
	notToken
	leftParenToken
	rightParenToken
)

func lex(expression string) ([]lexToken, error) {
	runes := []rune(expression)
	result := make([]lexToken, 0)
	for i := 0; i < len(runes); {
		if unicode.IsSpace(runes[i]) {
			i++
			continue
		}
		if runes[i] == '(' || runes[i] == ')' {
			kind := leftParenToken
			if runes[i] == ')' {
				kind = rightParenToken
			}
			result = append(result, lexToken{kind: kind, text: string(runes[i])})
			i++
			continue
		}
		start := i
		inQuote := false
		for i < len(runes) {
			if runes[i] == '"' {
				inQuote = !inQuote
				i++
				continue
			}
			if !inQuote && (unicode.IsSpace(runes[i]) || runes[i] == '(' || runes[i] == ')') {
				break
			}
			i++
		}
		if inQuote {
			return nil, fmt.Errorf("%w: unclosed quote", ErrInvalidQuery)
		}
		value := string(runes[start:i])
		kind := atomToken
		switch strings.ToUpper(value) {
		case "AND":
			kind = andToken
		case "OR":
			kind = orToken
		case "NOT":
			kind = notToken
		}
		result = append(result, lexToken{kind: kind, text: value})
	}
	return result, nil
}

type expressionParser struct {
	tokens       []lexToken
	index        int
	defaultField string
}

func (p *expressionParser) parseOr() (Query, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.index < len(p.tokens) {
		kind := p.tokens[p.index].kind
		implicit := kind == atomToken || kind == leftParenToken || kind == notToken
		if kind != orToken && !implicit {
			break
		}
		if !implicit {
			p.index++
		}
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = BooleanQuery{Should: []Query{left, right}}
	}
	return left, nil
}

func (p *expressionParser) parseAnd() (Query, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.index < len(p.tokens) && p.tokens[p.index].kind == andToken {
		p.index++
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = BooleanQuery{Must: []Query{left, right}}
	}
	return left, nil
}

func (p *expressionParser) parseUnary() (Query, error) {
	if p.index < len(p.tokens) && p.tokens[p.index].kind == notToken {
		p.index++
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return BooleanQuery{MustNot: []Query{child}}, nil
	}
	return p.parsePrimary()
}

func (p *expressionParser) parsePrimary() (Query, error) {
	if p.index >= len(p.tokens) {
		return nil, fmt.Errorf("%w: expected clause", ErrInvalidQuery)
	}
	token := p.tokens[p.index]
	if token.kind == leftParenToken {
		p.index++
		child, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.index >= len(p.tokens) || p.tokens[p.index].kind != rightParenToken {
			return nil, fmt.Errorf("%w: expected ')'", ErrInvalidQuery)
		}
		p.index++
		return child, nil
	}
	if token.kind != atomToken {
		return nil, fmt.Errorf("%w near %q", ErrInvalidQuery, token.text)
	}
	p.index++
	return atomQuery(token.text, p.defaultField)
}

func atomQuery(value, defaultField string) (Query, error) {
	field, text := defaultField, value
	if colon := strings.IndexByte(value, ':'); colon >= 0 {
		field, text = value[:colon], value[colon+1:]
	}
	if field == "" || text == "" {
		return nil, fmt.Errorf("%w: empty field or value", ErrInvalidQuery)
	}
	if strings.HasPrefix(text, "\"") {
		if len(text) < 2 || !strings.HasSuffix(text, "\"") {
			return nil, fmt.Errorf("%w: malformed phrase", ErrInvalidQuery)
		}
		return PhraseQuery{Field: field, Text: strings.Trim(text, "\"")}, nil
	}
	if strings.HasSuffix(text, "*") && len(text) > 1 {
		return PrefixQuery{Field: field, Prefix: strings.TrimSuffix(text, "*")}, nil
	}
	return MatchQuery{Field: field, Text: text}, nil
}
