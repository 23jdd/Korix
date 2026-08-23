// Package inverted 定义与存储后端无关的倒排索引基础结构和编码。
package inverted

import "sort"

// Posting 汇总一个 term 在一篇文档中的全部出现信息。
// Frequency 供 BM25 计算 TF；Positions 供 PhraseQuery 判断词序和距离。DocID 是
// Index 分配的内部数字 ID，而不是用户提供的字符串 ID。
type Posting struct {
	DocID     uint64   `json:"doc_id"`
	Frequency uint32   `json:"frequency"`
	Positions []uint32 `json:"positions,omitempty"`
}

// NewPosting 复制并排序 position，同时从 position 数量推导 Frequency。
// 复制可避免调用方随后修改输入切片，排序则是 delta 编码和短语合并的前提。
func NewPosting(docID uint64, positions []uint32) Posting {
	copyPositions := append([]uint32(nil), positions...)
	sort.Slice(copyPositions, func(i, j int) bool { return copyPositions[i] < copyPositions[j] })
	return Posting{DocID: docID, Frequency: uint32(len(copyPositions)), Positions: copyPositions}
}
