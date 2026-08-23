package index

import "github.com/23jdd/Koris/highlight"

func (i *Index) Highlight(docID uint64, field string, terms []string, options highlight.Options) ([]highlight.Fragment, error) {
	doc, err := i.Document(docID)
	if err != nil {
		return nil, err
	}
	return highlight.Extract(i.Analyzer(field), doc.Fields[field], terms, options), nil
}
