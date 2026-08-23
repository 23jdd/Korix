package storage

type entry struct {
	key   []byte
	value []byte
}

type sliceIterator struct {
	entries []entry
	index   int
	closed  bool
	err     error
}

func newSliceIterator(entries []entry) Iterator {
	// index=-1 表示尚未开始；第一次 Next 会移动到 entries[0]。
	return &sliceIterator{entries: entries, index: -1}
}

func errorIterator(err error) Iterator { return &sliceIterator{index: -1, err: err} }

func (i *sliceIterator) Next() bool {
	// 结束时把 index 放到 len(entries)，使 Valid 与 Key/Value 都稳定返回无效。
	if i.closed || i.err != nil || i.index+1 >= len(i.entries) {
		i.index = len(i.entries)
		return false
	}
	i.index++
	return true
}

func (i *sliceIterator) Valid() bool {
	return !i.closed && i.err == nil && i.index >= 0 && i.index < len(i.entries)
}

func (i *sliceIterator) Key() []byte {
	if !i.Valid() {
		return nil
	}
	// 禁止把内部快照暴露给调用者，否则外部修改会破坏后续读取结果。
	return cloneBytes(i.entries[i.index].key)
}

func (i *sliceIterator) Value() []byte {
	if !i.Valid() {
		return nil
	}
	return cloneBytes(i.entries[i.index].value)
}

func (i *sliceIterator) Error() error { return i.err }

func (i *sliceIterator) Close() error {
	i.closed = true
	i.entries = nil
	return nil
}

func cloneBytes(value []byte) []byte {
	// KV 存储的 []byte 所有权容易混淆，统一通过此函数执行 defensive copy。
	if value == nil {
		return nil
	}
	clone := make([]byte, len(value))
	copy(clone, value)
	return clone
}
