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
	return &sliceIterator{entries: entries, index: -1}
}

func errorIterator(err error) Iterator { return &sliceIterator{index: -1, err: err} }

func (i *sliceIterator) Next() bool {
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
	if value == nil {
		return nil
	}
	clone := make([]byte, len(value))
	copy(clone, value)
	return clone
}
