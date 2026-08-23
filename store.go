package Koris

import "github.com/23jdd/Koris/storage"

type (
	// Store 是事务 KV 存储接口的根包别名。
	Store = storage.Store
	// Tx 是仅在 Store.Transaction 回调期间有效的事务视图。
	Tx = storage.Tx
	// Iterator 是按 key 排序的前缀扫描游标。
	Iterator = storage.Iterator
)

// NewMemoryStore 创建内存事务 Store。
var NewMemoryStore = storage.NewMemoryStore

// OpenBboltStore 打开持久化 Bbolt Store。
var OpenBboltStore = storage.OpenBbolt
