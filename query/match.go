package query

// Operator 控制 MatchQuery 如何组合分析后的多个 term。
type Operator uint8

const (
	// OR 返回命中任一 term 的文档，并累加所有命中 term 的 BM25 分数。
	OR Operator = iota
	// AND 只保留命中全部不同 term 的文档。
	AND
)

// MatchQuery 先使用字段 Analyzer 处理原始 Text，再执行多个 Term 查询。
// 重复 term 会去重，避免用户重复输入同一个词时人为重复累计分数。
type MatchQuery struct {
	// Field 是要搜索的字段名。
	Field string
	// Text 是尚未分析的用户输入。
	Text string
	// Operator 决定不同 term 使用 AND 还是 OR；零值为 OR。
	Operator Operator
}

func (q MatchQuery) Execute(searcher Searcher) ([]Hit, error) {
	tokens := searcher.Analyzer(q.Field).Analyze(q.Text)
	if len(tokens) == 0 {
		return nil, nil
	}
	terms := make([]string, 0, len(tokens))
	seen := make(map[string]struct{})
	for _, token := range tokens {
		if _, found := seen[token.Term]; !found {
			seen[token.Term] = struct{}{}
			terms = append(terms, token.Term)
		}
	}
	// scores 累加相关性；matched 记录每篇文档命中的不同 term 数，供 AND 过滤。
	scores := make(map[string]float64)
	matched := make(map[string]int)
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
			matched[docID]++
		}
	}
	if q.Operator == AND {
		// 所有 term 扫描结束后一次删除不完整命中，比每轮反复求交更容易同时累分。
		for docID := range scores {
			if matched[docID] != len(terms) {
				delete(scores, docID)
			}
		}
	}
	return hitsFromScores(scores)
}
