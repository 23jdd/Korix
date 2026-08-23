package index

import (
	"errors"
	"sort"
	"strings"

	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/inverted"
	"github.com/23jdd/Koris/storage"
)

// Document 按内部 DocID 返回原始文档副本。Store 的 NotFound 被转换成稳定的
// ErrDocumentMissing，调用方无需依赖后端错误。
func (i *Index) Document(docID uint64) (document.Document, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var doc document.Document
	if err := getJSON(i.store, documentKey(docID), &doc); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return document.Document{}, ErrDocumentMissing
		}
		return document.Document{}, err
	}
	return doc.Clone(), nil
}

// DocumentByID 先解析外部字符串 ID，再读取原始文档。返回深拷贝，调用方修改
// Fields 不会改变索引内容。
func (i *Index) DocumentByID(externalID string) (document.Document, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	docID, found, err := lookupExternalID(i.store, externalID)
	if err != nil {
		return document.Document{}, err
	}
	if !found {
		return document.Document{}, ErrDocumentMissing
	}
	var doc document.Document
	if err := getJSON(i.store, documentKey(docID), &doc); err != nil {
		return document.Document{}, err
	}
	return doc.Clone(), nil
}

// Metadata 返回 BM25 所需的公开文档统计，不暴露内部 term vector。
func (i *Index) Metadata(docID uint64) (DocumentMetadata, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var metadata storedDocumentMetadata
	if err := getJSON(i.store, documentMetaKey(docID), &metadata); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return DocumentMetadata{}, ErrDocumentMissing
		}
		return DocumentMetadata{}, err
	}
	return metadata.DocumentMetadata, nil
}

// Stats 返回文档总数、字段总长度和下一 DocID。值由 JSON 解码得到，map 不与
// Store 内存共享，调用方可安全读取或复制。
func (i *Index) Stats() (GlobalStats, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var stats GlobalStats
	err := getJSON(i.store, globalMetaKey, &stats)
	return stats, err
}

// QueryMetadata 是 Query 层使用的无包循环 metadata 视图。它复制 Lengths map，
// 防止查询实现意外修改 Index 返回的统计。
func (i *Index) QueryMetadata(docID uint64) (string, map[string]uint32, error) {
	metadata, err := i.Metadata(docID)
	if err != nil {
		return "", nil, err
	}
	lengths := make(map[string]uint32, len(metadata.Lengths))
	for field, length := range metadata.Lengths {
		lengths[field] = length
	}
	return metadata.ExternalID, lengths, nil
}

// SearchStats 是 Query 层使用的无包循环全局统计视图，并复制字段长度 map。
func (i *Index) SearchStats() (uint64, map[string]uint64, error) {
	stats, err := i.Stats()
	if err != nil {
		return 0, nil, err
	}
	totals := make(map[string]uint64, len(stats.TotalFieldLength))
	for field, total := range stats.TotalFieldLength {
		totals[field] = total
	}
	return stats.DocumentCount, totals, nil
}

// TermInfo 返回字段词典统计。不存在的 term 返回零值而不是错误，使普通的
// “无命中”查询不需要特殊处理 storage.ErrNotFound。
func (i *Index) TermInfo(field, term string) (inverted.TermInfo, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var info inverted.TermInfo
	err := getJSON(i.store, termKey(field, term), &info)
	if errors.Is(err, storage.ErrNotFound) {
		return inverted.TermInfo{}, nil
	}
	return info, err
}

// Postings 通过前缀扫描返回一个 field/term 的全部 posting。固定宽度 DocID key
// 保证结果按 DocID 升序，可直接用于 Boolean/Phrase merge。
func (i *Index) Postings(field, term string) ([]inverted.Posting, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	iterator := i.store.Scan(termPostingPrefix(field, term))
	defer iterator.Close()
	postings := make([]inverted.Posting, 0)
	for iterator.Next() {
		// 每条 value 独立校验；任一损坏都会终止查询并返回 codec 错误。
		posting, err := inverted.DecodePosting(iterator.Value())
		if err != nil {
			return nil, err
		}
		postings = append(postings, posting)
	}
	return postings, iterator.Error()
}

// Terms 返回字段中以原始字符串 prefix 开头的全部 term，并按 term 排序。
//
// 因 key component 使用 Base64，编码后的字节前缀不等于原 term 前缀，所以需要
// 扫描整个字段词典、解码后过滤。未来可在该接口后替换为 Trie/FST。
func (i *Index) Terms(field, prefix string) ([]string, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	iterator := i.store.Scan(fieldTermPrefix(field))
	defer iterator.Close()
	terms := make([]string, 0)
	for iterator.Next() {
		term, err := decodeComponent(lastKeyComponent(iterator.Key()))
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(term, prefix) {
			terms = append(terms, term)
		}
	}
	sort.Strings(terms)
	return terms, iterator.Error()
}

// AllDocumentIDs 返回全部有效内部 DocID，顺序由固定宽度 docmeta key 保证升序。
// BooleanQuery 仅含 MustNot 子句时用它构造初始全集。
func (i *Index) AllDocumentIDs() ([]uint64, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	iterator := i.store.Scan(docMetaPrefix)
	defer iterator.Close()
	ids := make([]uint64, 0)
	for iterator.Next() {
		docID, err := parseDocIDPart(lastKeyComponent(iterator.Key()))
		if err != nil {
			return nil, err
		}
		ids = append(ids, docID)
	}
	return ids, iterator.Error()
}
