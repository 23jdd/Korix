// Package Koris 提供嵌入式全文搜索引擎的精简入口。
//
// 简单场景可直接使用 New/Open、Document 和 Query 别名；需要自定义 Analyzer、
// Store、codec 或查询执行时，可导入对应子包。搜索核心未调用第三方搜索引擎库，
// bbolt 仅作为可选 KV 持久化后端。
package Koris

import (
	"github.com/23jdd/Koris/document"
	indexpkg "github.com/23jdd/Koris/index"
	"github.com/23jdd/Koris/storage"
)

// New 创建临时内存索引。它适合测试、短生命周期任务和中小数据集；进程退出后
// 数据消失。options 与持久化 Open 使用相同配置方式。
func New(options ...indexpkg.Option) (*indexpkg.Index, error) {
	return indexpkg.New(storage.NewMemoryStore(), options...)
}

// Open 创建或打开 Bbolt 持久化索引。若后续 Index 初始化失败，会自动关闭已经
// 打开的 Store，避免文件句柄泄漏。调用者成功获得 Index 后负责 Close。
func Open(path string, options ...indexpkg.Option) (*indexpkg.Index, error) {
	store, err := storage.OpenBbolt(path, 0o600)
	if err != nil {
		return nil, err
	}
	idx, err := indexpkg.New(store, options...)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return idx, nil
}

// Add 是构造 Document 并调用 Index.Add 的便捷函数；复杂场景可直接使用
// idx.Add/idx.AddBatch 以获得更明确的数据模型。
func Add(idx *indexpkg.Index, id string, fields map[string]string) (string, error) {
	return idx.Add(document.Document{ID: id, Fields: fields})
}
