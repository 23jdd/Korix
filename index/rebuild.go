package index

import (
	"encoding/json"
	"fmt"

	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/storage"
)

// Rebuild 以 document/* 原始文档为唯一事实来源，在一个事务中重建全部派生数据。
//
// 适用场景包括 Analyzer 迁移、Check 报告不一致或升级索引 schema。重建始终
// 保留 Document.ID；任何文档 JSON 损坏或写入失败都会让整个事务回滚，旧索引
// 仍保持原状。
func (i *Index) Rebuild() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.store.Transaction(func(tx storage.Tx) error {
		// 必须先把原文完整复制到事务内存，再删除 document 前缀；Iterator 快照
		// 不应被当作删除后的长期数据源。
		documents := make([]document.Document, 0)
		iterator := tx.Scan(docPrefix)
		for iterator.Next() {
			var doc document.Document
			if err := json.Unmarshal(iterator.Value(), &doc); err != nil {
				iterator.Close()
				return fmt.Errorf("%w: document: %v", ErrCorruptIndex, err)
			}
			documents = append(documents, doc)
		}
		if err := iterator.Error(); err != nil {
			iterator.Close()
			return err
		}
		iterator.Close()
		// meta/global 最后直接覆盖；其余原文与派生键全部清空后重放 Document。
		for _, prefix := range [][]byte{docPrefix, docMetaPrefix, legacyIDPrefix, postingPrefix, termPrefix} {
			if err := deletePrefix(tx, prefix); err != nil {
				return err
			}
		}
		global := GlobalStats{TotalFieldLength: make(map[string]uint64)}
		for _, doc := range documents {
			if _, err := i.addInTransaction(tx, &global, doc); err != nil {
				return err
			}
		}
		return putJSON(tx, globalMetaKey, global)
	})
}

func deletePrefix(tx storage.Tx, prefix []byte) error {
	// 先收集 key 再删除，避免不同 Store 对“遍历期间修改”产生不一致行为。
	iterator := tx.Scan(prefix)
	keys := make([][]byte, 0)
	for iterator.Next() {
		keys = append(keys, iterator.Key())
	}
	if err := iterator.Error(); err != nil {
		iterator.Close()
		return err
	}
	iterator.Close()
	for _, key := range keys {
		if err := tx.Delete(key); err != nil {
			return err
		}
	}
	return nil
}
