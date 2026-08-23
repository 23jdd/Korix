package storage

import (
	"bytes"
	"sort"
	"sync"
)

// MemoryStore 是并发安全的内存事务存储。
//
// 写事务在整张 map 的副本上执行：回调成功时一次替换原 map，失败时直接丢弃
// 副本，因此语义与 Bbolt 事务一致。该策略优先保证实现清晰和测试可靠性，代价是
// 写事务 O(n) 复制；它适合测试和中小型临时索引。
type MemoryStore struct {
	mu     sync.RWMutex
	data   map[string][]byte
	closed bool
}

// NewMemoryStore 创建空的可用 Store。
func NewMemoryStore() *MemoryStore { return &MemoryStore{data: make(map[string][]byte)} }

func (s *MemoryStore) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrClosed
	}
	value, found := s.data[string(key)]
	if !found {
		return nil, ErrNotFound
	}
	// 返回副本，防止调用方绕过锁修改 Store 内部状态。
	return cloneBytes(value), nil
}

func (s *MemoryStore) Put(key, value []byte) error {
	return s.Transaction(func(tx Tx) error { return tx.Put(key, value) })
}

func (s *MemoryStore) Delete(key []byte) error {
	return s.Transaction(func(tx Tx) error { return tx.Delete(key) })
}

func (s *MemoryStore) Scan(prefix []byte) Iterator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return errorIterator(ErrClosed)
	}
	// scanMap 在持有读锁时复制匹配项，释放锁后 Iterator 不再依赖原 map。
	return scanMap(s.data, prefix)
}

func (s *MemoryStore) Transaction(fn func(Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	// 写锁覆盖复制、回调与提交，确保两个事务不会基于同一旧版本分别提交。
	working := cloneMap(s.data)
	if err := fn(&memoryTx{data: working}); err != nil {
		return err
	}
	// 指针替换是提交点；在此之前的任何错误都不会触碰已提交数据。
	s.data = working
	return nil
}

func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

type memoryTx struct{ data map[string][]byte }

func (tx *memoryTx) Get(key []byte) ([]byte, error) {
	value, found := tx.data[string(key)]
	if !found {
		return nil, ErrNotFound
	}
	return cloneBytes(value), nil
}

func (tx *memoryTx) Put(key, value []byte) error {
	tx.data[string(key)] = cloneBytes(value)
	return nil
}

func (tx *memoryTx) Delete(key []byte) error {
	delete(tx.data, string(key))
	return nil
}

func (tx *memoryTx) Scan(prefix []byte) Iterator { return scanMap(tx.data, prefix) }

func scanMap(data map[string][]byte, prefix []byte) Iterator {
	entries := make([]entry, 0)
	for key, value := range data {
		if bytes.HasPrefix([]byte(key), prefix) {
			entries = append(entries, entry{key: []byte(key), value: cloneBytes(value)})
		}
	}
	// Go map 无迭代顺序；显式排序使 Memory 与 Bbolt 的 Scan 行为完全一致，
	// posting 合并也可依赖 DocID key 的有序性。
	sort.Slice(entries, func(i, j int) bool { return bytes.Compare(entries[i].key, entries[j].key) < 0 })
	return newSliceIterator(entries)
}

func cloneMap(data map[string][]byte) map[string][]byte {
	clone := make(map[string][]byte, len(data))
	for key, value := range data {
		clone[key] = cloneBytes(value)
	}
	return clone
}
