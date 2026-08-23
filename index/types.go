// Package index owns documents, metadata and inverted-index persistence.
package index

import (
	"errors"

	"github.com/23jdd/Koris/inverted"
)

var (
	// ErrInvalidDocument 表示外部 ID 为空、字段集合为空或字段名为空。
	ErrInvalidDocument = errors.New("index: document ID and fields are required")
	// ErrDocumentMissing 表示按内部或外部 ID 查找不到有效文档。
	ErrDocumentMissing = errors.New("index: document not found")
	// ErrCorruptIndex 表示持久化 JSON 或 schema 内容无法解码。
	ErrCorruptIndex = errors.New("index: corrupt stored value")
)

// DocumentMetadata 是对查询层公开的文档统计视图。
// Lengths 保存每个字段经过 Analyzer 后的 token 数，供 BM25 计算 DL。持久化内部
// 结构还包含 term vector，但不公开它，防止调用者依赖更新实现细节。
type DocumentMetadata struct {
	DocID      uint64            `json:"doc_id"`
	ExternalID string            `json:"external_id"`
	Lengths    map[string]uint32 `json:"lengths"`
}

// GlobalStats 保存整个索引的聚合统计。
// TotalFieldLength 是每个字段在所有文档中的 token 总数；除以 DocumentCount 得到
// BM25 avgDL。NextDocumentID 单调递增，删除文档不会回收旧 ID。
type GlobalStats struct {
	DocumentCount    uint64            `json:"document_count"`
	TotalFieldLength map[string]uint64 `json:"total_field_length"`
	NextDocumentID   uint64            `json:"next_document_id"`
}

type storedDocumentMetadata struct {
	// TermVectors[field][term] 是该 term 的全部 position。更新时直接用旧 vector
	// 回收 posting 和统计，即使 Analyzer 配置已变化也不会留下幽灵 term。
	DocumentMetadata
	TermVectors map[string]map[string][]uint32 `json:"term_vectors"`
}

// TermStats 把字段、term 与词典统计组合成便于诊断或导出的值。
type TermStats struct {
	Field string
	Term  string
	Info  inverted.TermInfo
}

// ConsistencyReport 汇总一次只读全量审计。Problems 为空表示已检查的不变量成立；
// Documents/Terms/Postings 可用于观察索引规模。
type ConsistencyReport struct {
	Documents uint64
	Terms     uint64
	Postings  uint64
	Problems  []string
}

// Valid 报告审计是否未发现问题。
func (r ConsistencyReport) Valid() bool { return len(r.Problems) == 0 }
