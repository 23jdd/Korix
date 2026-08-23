package inverted

import "math"

type skipPoint struct {
	docID uint64
	index int
}

// SkipListIterator maintains sqrt(N) skip points over a posting list. It is a
// compact alternative to binary search when postings are streamed from blocks.
type SkipListIterator struct {
	postings []Posting
	skips    []skipPoint
	index    int
}

func NewSkipListIterator(postings []Posting) *SkipListIterator {
	iterator := &SkipListIterator{postings: append([]Posting(nil), postings...), index: -1}
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
