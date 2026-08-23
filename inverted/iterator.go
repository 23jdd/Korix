package inverted

import "sort"

// PostingIterator 遍历按 DocID 排序的 posting，并支持向前跳转。
// SkipTo 使用二分查找，适合 Boolean AND 在长短 posting list 间快速对齐 DocID。
type PostingIterator struct {
	postings []Posting
	index    int
}

// NewPostingIterator 复制并按 DocID 排序输入，防止调用方修改或传入无序数据破坏
// SkipTo 的二分前提。
func NewPostingIterator(postings []Posting) *PostingIterator {
	copyPostings := append([]Posting(nil), postings...)
	sort.Slice(copyPostings, func(i, j int) bool { return copyPostings[i].DocID < copyPostings[j].DocID })
	return &PostingIterator{postings: copyPostings, index: -1}
}

func (i *PostingIterator) Next() bool {
	if i.index+1 >= len(i.postings) {
		i.index = len(i.postings)
		return false
	}
	i.index++
	return true
}

func (i *PostingIterator) Valid() bool { return i.index >= 0 && i.index < len(i.postings) }

func (i *PostingIterator) Posting() Posting {
	if !i.Valid() {
		return Posting{}
	}
	return i.postings[i.index]
}

func (i *PostingIterator) SkipTo(docID uint64) bool {
	// 只在当前游标之后查找，保证 Iterator 单调向前，绝不重新返回旧 DocID。
	start := i.index
	if start < 0 {
		start = 0
	}
	offset := sort.Search(len(i.postings)-start, func(j int) bool {
		return i.postings[start+j].DocID >= docID
	})
	i.index = start + offset
	return i.Valid()
}
