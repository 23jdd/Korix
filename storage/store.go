// Package storage 定义索引与持久化实现之间的最小边界。
//
// 上层只使用有序 KV、前缀扫描和原子事务，因此 MemoryStore 与 BboltStore 对搜索
// 逻辑完全等价。新的存储实现只要遵守复制、排序和事务回滚约定即可接入。
package storage

import "errors"

var (
	// ErrNotFound 表示键不存在。调用方应使用 errors.Is 判断。
	ErrNotFound = errors.New("storage: key not found")
	// ErrClosed 表示 Store 已关闭，之后的读写不再有效。
	ErrClosed = errors.New("storage: store closed")
)

// Iterator 遍历一个不可变、按 key 字节字典序排列的扫描快照。
//
// 调用顺序为 Next → Key/Value；Next 返回 false 后必须检查 Error。Key 与 Value
// 返回独立副本，调用者可在下一次 Next 或 Close 后继续持有。使用结束应调用 Close。
type Iterator interface {
	Next() bool
	Valid() bool
	Key() []byte
	Value() []byte
	Error() error
	Close() error
}

// Reader 抽取 Store 与 Tx 共有的只读操作。Scan(prefix) 只返回以 prefix 开头的键。
type Reader interface {
	Get(key []byte) ([]byte, error)
	Scan(prefix []byte) Iterator
}

// Tx 是原子读写视图。Transaction 回调返回 nil 时所有修改一起提交，返回错误时
// 必须全部回滚。Tx 仅在回调执行期间有效，不应被保存到回调外部。
type Tx interface {
	Reader
	Put(key, value []byte) error
	Delete(key []byte) error
}

// Store 隔离持久化细节。Put/Delete 是单操作事务的便捷形式；需要同时维护文档、
// posting、词典和统计时，索引 Writer 必须使用 Transaction 保证一致性。
type Store interface {
	Reader
	Put(key, value []byte) error
	Delete(key []byte) error
	Transaction(fn func(Tx) error) error
	Close() error
}
