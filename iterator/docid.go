// Package iterator 提供查询合并使用的有序文档 ID 迭代器。
package iterator

import "sort"

// DocIDIterator 是单调向前的 DocID 游标。SkipTo 定位首个 >= target 的 ID，
// 返回 false 表示到达末尾。
type DocIDIterator interface {
	Next() bool
	DocID() uint64
	SkipTo(target uint64) bool
}

// Slice 是基于内存切片的 DocIDIterator 实现。
type Slice struct {
	ids   []uint64
	index int
}

// New 复制并排序输入 ID，调用方随后修改原切片不会影响迭代器。
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
	// 从当前游标起二分查找，保持严格单调遍历。
	start := i.index
	if start < 0 {
		start = 0
	}
	offset := sort.Search(len(i.ids)-start, func(n int) bool { return i.ids[start+n] >= target })
	i.index = start + offset
	return i.index < len(i.ids)
}
