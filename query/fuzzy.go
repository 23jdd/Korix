package query

// FuzzyQuery 把输入扩展为 Unicode Levenshtein 距离不超过 MaxEdits 的词典 term。
// MaxEdits 被限制为 1..3，避免配置错误导致大量低质量扩展；距离越远，BM25 分数
// 按 1/(distance+1) 衰减。
type FuzzyQuery struct {
	// Field 是要枚举词典的字段。
	Field string
	// Term 是未分析的原始查询词；仅使用 Analyzer 输出的第一个 token。
	Term string
	// MaxEdits 是最大 Unicode 编辑距离，非正数回退 1，大于 3 截断为 3。
	MaxEdits int
}

func (q FuzzyQuery) Execute(searcher Searcher) ([]Hit, error) {
	tokens := searcher.Analyzer(q.Field).Analyze(q.Term)
	if len(tokens) == 0 {
		return nil, nil
	}
	term := tokens[0].Term
	maxEdits := q.MaxEdits
	if maxEdits <= 0 {
		maxEdits = 1
	}
	if maxEdits > 3 {
		maxEdits = 3
	}
	// 空前缀枚举字段完整词典。大词典实现可在 Searcher.Terms 后接 BK-tree/FST。
	allTerms, err := searcher.Terms(q.Field, "")
	if err != nil {
		return nil, err
	}
	scores := make(map[uint64]float64)
	for _, candidate := range allTerms {
		distance := levenshtein(term, candidate, maxEdits)
		if distance > maxEdits {
			continue
		}
		postings, err := searcher.Postings(q.Field, candidate)
		if err != nil {
			return nil, err
		}
		termScores, err := scorePostings(searcher, q.Field, candidate, postings)
		if err != nil {
			return nil, err
		}
		boost := 1 / float64(distance+1)
		for docID, score := range termScores {
			scores[docID] += score * boost
		}
	}
	return hitsFromScores(searcher, scores)
}

func levenshtein(left, right string, cutoff int) int {
	// 在 rune 层计算，中文或 emoji 各算一个编辑单元，而不是按 UTF-8 字节计数。
	a, b := []rune(left), []rune(right)
	// 长度差已经超过 cutoff 时，无需分配 DP 行。
	if len(a)-len(b) > cutoff || len(b)-len(a) > cutoff {
		return cutoff + 1
	}
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	// 只保留上一行和当前行，把标准 O(mn) 空间降为 O(n)。
	for i, ar := range a {
		current[0] = i + 1
		rowMin := current[0]
		for j, br := range b {
			cost := 0
			if ar != br {
				cost = 1
			}
			current[j+1] = min3(current[j]+1, previous[j+1]+1, previous[j]+cost)
			if current[j+1] < rowMin {
				rowMin = current[j+1]
			}
		}
		if rowMin > cutoff {
			// 该行最优值也已越界，提前停止词典候选比较。
			return cutoff + 1
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
