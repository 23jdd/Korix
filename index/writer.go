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

// Add 写入一篇文档；外部 ID 已存在时原子替换旧文档。
// 返回的内部 DocID 在更新时保持不变，便于调用者缓存搜索结果引用。
func (i *Index) Add(doc document.Document) (uint64, error) {
	ids, err := i.AddBatch([]document.Document{doc})
	if err != nil {
		return 0, err
	}
	return ids[0], nil
}

// AddBatch 在一个事务中索引整个批次。
//
// 任一文档无效、Analyzer/编码失败或 Store 写入失败，都会回滚批次内已完成的所有
// 文档以及全局统计。返回 ID 的顺序与输入文档顺序一致。空批次是成功的空操作。
func (i *Index) AddBatch(documents []document.Document) ([]uint64, error) {
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
	ids := make([]uint64, len(documents))
	err := i.store.Transaction(func(tx storage.Tx) error {
		// global 只在本地副本上累计，全部文档完成后才写回同一事务。
		var global GlobalStats
		if err := getJSON(tx, globalMetaKey, &global); err != nil {
			return err
		}
		ensureGlobalMaps(&global)
		for n, doc := range documents {
			docID, err := i.addInTransaction(tx, &global, doc)
			if err != nil {
				return err
			}
			ids[n] = docID
		}
		return putJSON(tx, globalMetaKey, global)
	})
	return ids, err
}

func (i *Index) addInTransaction(tx storage.Tx, global *GlobalStats, doc document.Document) (uint64, error) {
	docID, exists, err := lookupExternalID(tx, doc.ID)
	if err != nil {
		return 0, err
	}
	if exists {
		// 更新必须先基于“旧 term vector”撤销派生数据，不能用当前 Analyzer 重新
		// 分析旧原文，否则 Analyzer 迁移后会删错 term。
		var old storedDocumentMetadata
		if err := getJSON(tx, documentMetaKey(docID), &old); err != nil {
			return 0, err
		}
		if err := removeVectors(tx, &old, global); err != nil {
			return 0, err
		}
	} else {
		// DocID 单调分配且不回收，避免删除后的 ID 被新文档复用造成悬空引用误命中。
		docID = global.NextDocumentID
		if docID == 0 {
			docID = 1
		}
		global.NextDocumentID = docID + 1
		global.DocumentCount++
	}

	metadata := storedDocumentMetadata{
		DocumentMetadata: DocumentMetadata{DocID: docID, ExternalID: doc.ID, Lengths: make(map[string]uint32)},
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
			posting := inverted.NewPosting(docID, positions)
			if err := tx.Put(postingKey(field, term, docID), inverted.EncodePosting(posting)); err != nil {
				return 0, err
			}
			var info inverted.TermInfo
			err := getJSON(tx, termKey(field, term), &info)
			if err != nil && !errors.Is(err, storage.ErrNotFound) {
				return 0, err
			}
			info.DocumentFrequency++
			info.TotalFrequency += uint64(len(positions))
			if err := putJSON(tx, termKey(field, term), info); err != nil {
				return 0, err
			}
		}
	}
	// 原始 Document 是 Rebuild 的事实来源；docmeta 与 id 映射均属于可重建/辅助数据。
	if err := putJSON(tx, documentKey(docID), doc.Clone()); err != nil {
		return 0, err
	}
	if err := putJSON(tx, documentMetaKey(docID), metadata); err != nil {
		return 0, err
	}
	if err := tx.Put(externalIDKey(doc.ID), encodeUint64(docID)); err != nil {
		return 0, err
	}
	return docID, nil
}

// Delete 按外部 ID 原子删除原文、映射、metadata、全部 posting 与相关统计。
// DocID 不会放回分配池；不存在的 ID 返回 ErrDocumentMissing。
func (i *Index) Delete(externalID string) error {
	if strings.TrimSpace(externalID) == "" {
		return ErrInvalidDocument
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.store.Transaction(func(tx storage.Tx) error {
		docID, exists, err := lookupExternalID(tx, externalID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrDocumentMissing
		}
		var global GlobalStats
		if err := getJSON(tx, globalMetaKey, &global); err != nil {
			return err
		}
		var metadata storedDocumentMetadata
		if err := getJSON(tx, documentMetaKey(docID), &metadata); err != nil {
			return err
		}
		if err := removeVectors(tx, &metadata, &global); err != nil {
			return err
		}
		if global.DocumentCount > 0 {
			global.DocumentCount--
		}
		if err := tx.Delete(documentKey(docID)); err != nil {
			return err
		}
		if err := tx.Delete(documentMetaKey(docID)); err != nil {
			return err
		}
		if err := tx.Delete(externalIDKey(externalID)); err != nil {
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
			if err := tx.Delete(postingKey(field, term, metadata.DocID)); err != nil {
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

func lookupExternalID(reader storage.Reader, externalID string) (uint64, bool, error) {
	// “未找到”是正常分支，通过 found=false 返回；其他 Store 错误必须向上传播。
	data, err := reader.Get(externalIDKey(externalID))
	if errors.Is(err, storage.ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	docID, err := decodeUint64(data)
	return docID, true, err
}

func ensureGlobalMaps(global *GlobalStats) {
	// 兼容早期或手工构造的 metadata：nil map 不能直接累加，next=0 也不是合法
	// 分配值，因此在每个写事务开始时补齐默认值。
	if global.TotalFieldLength == nil {
		global.TotalFieldLength = make(map[string]uint64)
	}
	if global.NextDocumentID == 0 {
		global.NextDocumentID = 1
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
