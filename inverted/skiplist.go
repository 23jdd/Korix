package inverted

import "math"

type skipPoint struct {
	docID uint64
	index int
}

// SkipListIterator 在 posting list 上每隔约 sqrt(N) 个元素建立一个跳点。
//
// 与需要随机访问完整切片的二分相比，显式跳点更接近磁盘分块 posting 的访问
// 模式：先跨块跳跃，再在目标块内顺序扫描。传入 postings 必须已按 DocID 排序。
type SkipListIterator struct {
	postings []Posting
	skips    []skipPoint
	index    int
}

// NewSkipListIterator 复制 posting slice 并构建稀疏跳点；position 数据仍按
// Posting 的只读约定使用。
func NewSkipListIterator(postings []Posting) *SkipListIterator {
	iterator := &SkipListIterator{postings: append([]Posting(nil), postings...), index: -1}
	// sqrt(N) 在跳点空间和块内线性扫描长度之间取得平衡。
	step := int(math.Sqrt(float64(len(postings))))
	if step < 2 {
		step = 2
	}
	for index := step; index < len(postings); index += step {
		iterator.skips = append(iterator.skips, skipPoint{docID: postings[index].DocID, index: index})
	}
	return iterator
}

func (i *SkipListIterator) Next() bool {
	if i.index+1 >= len(i.postings) {
		i.index = len(i.postings)
		return false
	}
	i.index++
	return true
}

func (i *SkipListIterator) Valid() bool { return i.index >= 0 && i.index < len(i.postings) }

func (i *SkipListIterator) Posting() Posting {
	if !i.Valid() {
		return Posting{}
	}
	return i.postings[i.index]
}

func (i *SkipListIterator) SkipTo(target uint64) bool {
	// 只采用 docID <= target 的最远跳点，避免跨过第一个满足条件的 posting。
	for _, point := range i.skips {
		if point.index > i.index && point.docID <= target {
			i.index = point.index
		}
	}
	if i.index < 0 {
		i.index = 0
	}
	for i.index < len(i.postings) && i.postings[i.index].DocID < target {
		i.index++
	}
	return i.Valid()
}
