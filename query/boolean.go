package query

// BooleanQuery 组合任意子 Query。
//
// Must 取交集并累加分数；Should 在没有 Must 时形成并集，有 Must 时仅给已有候选
// 加分；MustNot 删除候选。只有 MustNot 时会以全部有效文档为初始集合，这与常见
// 搜索引擎的纯否定查询语义一致。
type BooleanQuery struct {
	// Must 中的每个子查询都必须命中。
	Must []Query
	// Should 是可选加分子句；没有 Must 时至少需要命中一个 Should。
	Should []Query
	// MustNot 中任一子查询命中都会排除文档。
	MustNot []Query
}

func (q BooleanQuery) Execute(searcher Searcher) ([]Hit, error) {
	var candidates map[string]float64
	for clauseIndex, clause := range q.Must {
		hits, err := clause.Execute(searcher)
		if err != nil {
			return nil, err
		}
		clauseScores := hitScoreMap(hits)
		if clauseIndex == 0 {
			// 第一条 Must 直接建立候选集，后续 Must 原地求交，避免额外 map。
			candidates = clauseScores
			continue
		}
		for docID := range candidates {
			if score, found := clauseScores[docID]; found {
				candidates[docID] += score
			} else {
				delete(candidates, docID)
			}
		}
	}
	for _, clause := range q.Should {
		hits, err := clause.Execute(searcher)
		if err != nil {
			return nil, err
		}
		if candidates == nil {
			candidates = make(map[string]float64)
		}
		for _, hit := range hits {
			if len(q.Must) == 0 {
				candidates[hit.ID] += hit.Score
			} else if _, found := candidates[hit.ID]; found {
				candidates[hit.ID] += hit.Score
			}
		}
	}
	if candidates == nil {
		candidates = make(map[string]float64)
		if len(q.MustNot) > 0 {
			// 纯否定查询必须先取全集，否则从空集合删除仍然永远为空。
			ids, err := searcher.AllDocumentIDs()
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				candidates[id] = 0
			}
		}
	}
	for _, clause := range q.MustNot {
		hits, err := clause.Execute(searcher)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			delete(candidates, hit.ID)
		}
	}
	return hitsFromScores(candidates)
}

func hitScoreMap(hits []Hit) map[string]float64 {
	result := make(map[string]float64, len(hits))
	for _, hit := range hits {
		result[hit.ID] = hit.Score
	}
	return result
}
