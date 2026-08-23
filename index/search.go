package index

import "github.com/23jdd/Koris/query"

// Search 执行 Query，并在相关性排序完成后截取前 limit 条结果。
// limit <= 0 表示返回全部命中；nil Query 返回空结果。limit 在排序后应用，避免
// 各子查询提前截断导致 Boolean 合并漏掉高分文档。
func (i *Index) Search(q query.Query, limit int) ([]query.Hit, error) {
	if q == nil {
		return nil, nil
	}
	hits, err := q.Execute(i)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}
