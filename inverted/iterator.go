package inverted

import "sort"

// PostingIterator exposes sorted postings and supports skip-ahead for Boolean
// intersections. SkipTo performs a binary search over the immutable slice.
type PostingIterator struct {
	postings []Posting
	index    int
}

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
