// Package iterator provides sorted document-ID iterators used by query merges.
package iterator

import "sort"

type DocIDIterator interface {
	Next() bool
	DocID() uint64
	SkipTo(target uint64) bool
}

type Slice struct {
	ids   []uint64
	index int
}

func New(ids []uint64) *Slice {
	copyIDs := append([]uint64(nil), ids...)
	sort.Slice(copyIDs, func(i, j int) bool { return copyIDs[i] < copyIDs[j] })
	return &Slice{ids: copyIDs, index: -1}
}

func (i *Slice) Next() bool {
	if i.index+1 >= len(i.ids) {
		i.index = len(i.ids)
		return false
	}
	i.index++
	return true
}

func (i *Slice) DocID() uint64 {
	if i.index < 0 || i.index >= len(i.ids) {
		return 0
	}
	return i.ids[i.index]
}

func (i *Slice) SkipTo(target uint64) bool {
	start := i.index
	if start < 0 {
		start = 0
	}
	offset := sort.Search(len(i.ids)-start, func(n int) bool { return i.ids[start+n] >= target })
	i.index = start + offset
	return i.index < len(i.ids)
}
