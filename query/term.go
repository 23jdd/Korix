package query

// TermQuery 执行一个字段内的精确 term 查询。
// Term 不会再次经过 Analyzer，适合调用者已经持有规范化 term 的场景；原始用户
// 文本应使用 MatchQuery，否则大小写、停用词和 stemming 可能与索引不一致。
type TermQuery struct {
	// Field 是目标字段。
	Field string
	// Term 必须是与索引 Analyzer 输出一致的规范化词项。
	Term string
}

func (q TermQuery) Execute(searcher Searcher) ([]Hit, error) {
	if q.Field == "" || q.Term == "" {
		return nil, nil
	}
	postings, err := searcher.Postings(q.Field, q.Term)
	if err != nil {
		return nil, err
	}
	scores, err := scorePostings(searcher, q.Field, q.Term, postings)
	if err != nil {
		return nil, err
	}
	hits, err := hitsFromScores(scores)
	if err != nil {
		return nil, err
	}
	// Explanation 需要 TF；排序后的 Hit 已不再直接携带 Posting，所以建立一次
	// ID→Frequency 映射补充轻量解释。
	byID := make(map[string]uint32, len(postings))
	for _, posting := range postings {
		byID[posting.DocID] = posting.Frequency
	}
	for i := range hits {
		hits[i].Explanation = explanation(q.Field, q.Term, byID[hits[i].ID])
	}
	return hits, nil
}
