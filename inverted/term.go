package inverted

// TermInfo 是词典中一个 field/term 的持久化统计。
// DocumentFrequency 表示包含该 term 的文档数，用于 BM25 IDF；TotalFrequency
// 是所有文档 TF 之和，可用于统计、优化和未来查询规划。
type TermInfo struct {
	DocumentFrequency uint64 `json:"document_frequency"`
	TotalFrequency    uint64 `json:"total_frequency"`
}
