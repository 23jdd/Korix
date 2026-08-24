package query

// PrefixQuery 把输入前缀扩展为字段词典中的所有匹配 term，再合并其 BM25 得分。
// Prefix 会先经过字段 Analyzer，以便大小写规则与索引保持一致。
type PrefixQuery struct {
	// Field 是要扩展词典的字段。
	Field string
	// Prefix 是不含星号的原始前缀。
	Prefix string
}

func (q PrefixQuery) Execute(searcher Searcher) ([]Hit, error) {
	tokens := searcher.Analyzer(q.Field).Analyze(q.Prefix)
	prefix := q.Prefix
	if len(tokens) > 0 {
		prefix = tokens[0].Term
	}
	// Terms 的具体实现可以是当前词典扫描，也可以未来替换为 Trie/FST。
	terms, err := searcher.Terms(q.Field, prefix)
	if err != nil {
		return nil, err
	}
	scores := make(map[string]float64)
	for _, term := range terms {
		postings, err := searcher.Postings(q.Field, term)
		if err != nil {
			return nil, err
		}
		termScores, err := scorePostings(searcher, q.Field, term, postings)
		if err != nil {
			return nil, err
		}
		for docID, score := range termScores {
			scores[docID] += score
		}
	}
	return hitsFromScores(scores)
}
