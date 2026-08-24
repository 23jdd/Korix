package query

import "github.com/23jdd/Koris/inverted"

// PhraseQuery 要求分析后的 term 按顺序、按词位距离出现在同一字段中。
// Slop 允许实际间距比查询间距额外多出的总词位数；0 表示精确短语。
type PhraseQuery struct {
	// Field 是要检查 position 的字段。
	Field string
	// Text 会先通过字段 Analyzer，停用词产生的 position gap 会被保留。
	Text string
	// Slop 是允许超出查询词位间隔的总距离。
	Slop uint32
}

func (q PhraseQuery) Execute(searcher Searcher) ([]Hit, error) {
	tokens := searcher.Analyzer(q.Field).Analyze(q.Text)
	if len(tokens) == 0 {
		return nil, nil
	}
	// 每个 term 建立 DocID→Posting 表，随后以第一个 term 的文档为候选探测其他
	// term，避免对所有文档做笛卡尔遍历。
	postingSets := make([]map[string]inverted.Posting, len(tokens))
	termScores := make([]map[string]float64, len(tokens))
	for i, token := range tokens {
		postings, err := searcher.Postings(q.Field, token.Term)
		if err != nil {
			return nil, err
		}
		postingSets[i] = make(map[string]inverted.Posting, len(postings))
		for _, posting := range postings {
			postingSets[i][posting.DocID] = posting
		}
		termScores[i], err = scorePostings(searcher, q.Field, token.Term, postings)
		if err != nil {
			return nil, err
		}
	}
	scores := make(map[string]float64)
	for docID, first := range postingSets[0] {
		positions := make([][]uint32, len(tokens))
		positions[0] = first.Positions
		matched := true
		for i := 1; i < len(tokens); i++ {
			posting, found := postingSets[i][docID]
			if !found {
				matched = false
				break
			}
			positions[i] = posting.Positions
		}
		// Analyzer 可能删除停用词但保留 Position，因此计算查询 token 相对词位。
		relative := make([]uint32, len(tokens))
		for i := range tokens {
			relative[i] = tokens[i].Position - tokens[0].Position
		}
		if matched && phrasePositionsMatch(positions, relative, q.Slop) {
			for i := range tokens {
				scores[docID] += termScores[i][docID]
			}
			// 短语比独立 term 同现更具体，使用固定 boost 提升排序；基础分仍来自 BM25。
			scores[docID] *= 1.5
		}
	}
	return hitsFromScores(scores)
}

func phrasePositionsMatch(positions [][]uint32, relative []uint32, slop uint32) bool {
	if len(positions) == 0 {
		return false
	}
	// 尝试第一个 term 的每个出现位置；后续 term 总是选择不小于期望位置的最早
	// 出现，从而得到该起点下最小可能 slop。
	for _, start := range positions[0] {
		previous := start
		usedSlop := uint32(0)
		matched := true
		for i := 1; i < len(positions); i++ {
			expectedGap := uint32(1)
			if i < len(relative) && relative[i] > relative[i-1] {
				expectedGap = relative[i] - relative[i-1]
			}
			next, found := nextPosition(positions[i], previous+expectedGap)
			if !found || next < previous+expectedGap {
				matched = false
				break
			}
			usedSlop += next - (previous + expectedGap)
			if usedSlop > slop {
				matched = false
				break
			}
			previous = next
		}
		if matched {
			return true
		}
	}
	return false
}

func nextPosition(positions []uint32, minimum uint32) (uint32, bool) {
	// Positions 已由 Posting codec 保证有序，首次 >= minimum 即是最优候选。
	for _, position := range positions {
		if position >= minimum {
			return position, true
		}
	}
	return 0, false
}
