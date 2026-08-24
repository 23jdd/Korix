package index

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/inverted"
	"github.com/23jdd/Koris/storage"
)

// Add 写入一篇文档；ID 已存在时原子替换旧文档。返回值就是 doc.ID，Koris
// 不会额外分配内部数字 ID。
func (i *Index) Add(doc document.Document) (string, error) {
	ids, err := i.AddBatch([]document.Document{doc})
	if err != nil {
		return "", err
	}
	return ids[0], nil
}

// AddBatch 在一个事务中索引整个批次。
//
// 任一文档无效、Analyzer/编码失败或 Store 写入失败，都会回滚批次内已完成的所有
// 文档以及全局统计。返回 ID 的顺序与输入文档顺序一致。空批次是成功的空操作。
func (i *Index) AddBatch(documents []document.Document) ([]string, error) {
	if len(documents) == 0 {
		return nil, nil
	}
	// 在获取写锁和开启事务前完成纯输入校验，减少明显错误占用独占资源的时间。
	for _, doc := range documents {
		if err := validateDocument(doc); err != nil {
			return nil, err
		}
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	ids := make([]string, len(documents))
	err := i.store.Transaction(func(tx storage.Tx) error {
		// global 只在本地副本上累计，全部文档完成后才写回同一事务。
		var global GlobalStats
		if err := getJSON(tx, globalMetaKey, &global); err != nil {
			return err
		}
		ensureGlobalMaps(&global)
		for n, doc := range documents {
			id, err := i.addInTransaction(tx, &global, doc)
			if err != nil {
				return err
			}
			ids[n] = id
		}
		return putJSON(tx, globalMetaKey, global)
	})
	return ids, err
}

func (i *Index) addInTransaction(tx storage.Tx, global *GlobalStats, doc document.Document) (string, error) {
	var old storedDocumentMetadata
	err := getJSON(tx, documentMetaKey(doc.ID), &old)
	if err == nil {
		// 更新必须先基于“旧 term vector”撤销派生数据，不能用当前 Analyzer 重新
		// 分析旧原文，否则 Analyzer 迁移后会删错 term。
		if err := removeVectors(tx, &old, global); err != nil {
			return "", err
		}
	} else if errors.Is(err, storage.ErrNotFound) {
		global.DocumentCount++
	} else {
		return "", err
	}

	metadata := storedDocumentMetadata{
		DocumentMetadata: DocumentMetadata{ID: doc.ID, Lengths: make(map[string]uint32)},
		TermVectors:      make(map[string]map[string][]uint32),
	}
	// map 迭代无序；排序字段让写入顺序、测试结果和 Bbolt 页面变化尽量稳定。
	fields := sortedFields(doc.Fields)
	for _, field := range fields {
		tokens := i.Analyzer(field).Analyze(doc.Fields[field])
		metadata.Lengths[field] = uint32(len(tokens))
		global.TotalFieldLength[field] += uint64(len(tokens))
		// 先在内存按 term 聚合 position，同一文档/term 只写一条 Posting。
		vectors := make(map[string][]uint32)
		for _, token := range tokens {
			if token.Term != "" {
				vectors[token.Term] = append(vectors[token.Term], token.Position)
			}
		}
		metadata.TermVectors[field] = vectors
		for term, positions := range vectors {
			// Posting 与 TermInfo 必须在同一事务更新，保证 Reader 永远不会看到
			// 已增加 DF 但缺失 posting（或反之）的中间状态。
			posting := inverted.NewPosting(doc.ID, positions)
			if err := tx.Put(postingKey(field, term, doc.ID), inverted.EncodePosting(posting)); err != nil {
				return "", err
			}
			var info inverted.TermInfo
			err := getJSON(tx, termKey(field, term), &info)
			if err != nil && !errors.Is(err, storage.ErrNotFound) {
				return "", err
			}
			info.DocumentFrequency++
			info.TotalFrequency += uint64(len(positions))
			if err := putJSON(tx, termKey(field, term), info); err != nil {
				return "", err
			}
		}
	}
	// 原始 Document 是 Rebuild 的事实来源；docmeta 属于可重建的辅助数据。
	if err := putJSON(tx, documentKey(doc.ID), doc.Clone()); err != nil {
		return "", err
	}
	if err := putJSON(tx, documentMetaKey(doc.ID), metadata); err != nil {
		return "", err
	}
	return doc.ID, nil
}

// Delete 按字符串 ID 原子删除原文、metadata、全部 posting 与相关统计。
// 不存在的 ID 返回 ErrDocumentMissing。
func (i *Index) Delete(id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidDocument
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.store.Transaction(func(tx storage.Tx) error {
		var global GlobalStats
		if err := getJSON(tx, globalMetaKey, &global); err != nil {
			return err
		}
		var metadata storedDocumentMetadata
		if err := getJSON(tx, documentMetaKey(id), &metadata); errors.Is(err, storage.ErrNotFound) {
			return ErrDocumentMissing
		} else if err != nil {
			return err
		}
		if err := removeVectors(tx, &metadata, &global); err != nil {
			return err
		}
		if global.DocumentCount > 0 {
			global.DocumentCount--
		}
		if err := tx.Delete(documentKey(id)); err != nil {
			return err
		}
		if err := tx.Delete(documentMetaKey(id)); err != nil {
			return err
		}
		return putJSON(tx, globalMetaKey, global)
	})
}

func removeVectors(tx storage.Tx, metadata *storedDocumentMetadata, global *GlobalStats) error {
	// 本函数同时用于 Update 和 Delete：调用方负责决定是否减少 DocumentCount，
	// 这里仅回收字段长度、posting 以及 term 的 DF/总 TF。
	for field, terms := range metadata.TermVectors {
		length := uint64(metadata.Lengths[field])
		if global.TotalFieldLength[field] >= length {
			global.TotalFieldLength[field] -= length
		} else {
			global.TotalFieldLength[field] = 0
		}
		for term, positions := range terms {
			if err := tx.Delete(postingKey(field, term, metadata.ID)); err != nil {
				return err
			}
			var info inverted.TermInfo
			if err := getJSON(tx, termKey(field, term), &info); err != nil {
				return err
			}
			if info.DocumentFrequency <= 1 {
				// 最后一篇文档移除后直接删除词典项，Prefix/Fuzzy 不会枚举到空 term。
				if err := tx.Delete(termKey(field, term)); err != nil {
					return err
				}
			} else {
				info.DocumentFrequency--
				if info.TotalFrequency >= uint64(len(positions)) {
					info.TotalFrequency -= uint64(len(positions))
				}
				if err := putJSON(tx, termKey(field, term), info); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateDocument(doc document.Document) error {
	// 字段值允许为空：空字段仍是合法 stored field，只是不会生成 token。
	if strings.TrimSpace(doc.ID) == "" || len(doc.Fields) == 0 {
		return ErrInvalidDocument
	}
	for name := range doc.Fields {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%w: empty field name", ErrInvalidDocument)
		}
	}
	return nil
}

func ensureGlobalMaps(global *GlobalStats) {
	// 兼容早期或手工构造的 metadata：nil map 不能直接累加，因此在每个写事务
	// 开始时补齐默认值。
	if global.TotalFieldLength == nil {
		global.TotalFieldLength = make(map[string]uint64)
	}
}

func sortedFields(fields map[string]string) []string {
	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	sort.Strings(result)
	return result
}
