package index

import (
	"errors"
	"sort"
	"strings"

	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/inverted"
	"github.com/23jdd/Koris/storage"
)

func (i *Index) Document(docID uint64) (document.Document, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var doc document.Document
	if err := getJSON(i.store, documentKey(docID), &doc); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return document.Document{}, ErrDocumentMissing
		}
		return document.Document{}, err
	}
	return doc.Clone(), nil
}

func (i *Index) DocumentByID(externalID string) (document.Document, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	docID, found, err := lookupExternalID(i.store, externalID)
	if err != nil {
		return document.Document{}, err
	}
	if !found {
		return document.Document{}, ErrDocumentMissing
	}
	var doc document.Document
	if err := getJSON(i.store, documentKey(docID), &doc); err != nil {
		return document.Document{}, err
	}
	return doc.Clone(), nil
}

func (i *Index) Metadata(docID uint64) (DocumentMetadata, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var metadata storedDocumentMetadata
	if err := getJSON(i.store, documentMetaKey(docID), &metadata); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return DocumentMetadata{}, ErrDocumentMissing
		}
		return DocumentMetadata{}, err
	}
	return metadata.DocumentMetadata, nil
}

func (i *Index) Stats() (GlobalStats, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var stats GlobalStats
	err := getJSON(i.store, globalMetaKey, &stats)
	return stats, err
}

// QueryMetadata is the dependency-neutral metadata view consumed by queries.
func (i *Index) QueryMetadata(docID uint64) (string, map[string]uint32, error) {
	metadata, err := i.Metadata(docID)
	if err != nil {
		return "", nil, err
	}
	lengths := make(map[string]uint32, len(metadata.Lengths))
	for field, length := range metadata.Lengths {
		lengths[field] = length
	}
	return metadata.ExternalID, lengths, nil
}

// SearchStats is the dependency-neutral global view consumed by queries.
func (i *Index) SearchStats() (uint64, map[string]uint64, error) {
	stats, err := i.Stats()
	if err != nil {
		return 0, nil, err
	}
	totals := make(map[string]uint64, len(stats.TotalFieldLength))
	for field, total := range stats.TotalFieldLength {
		totals[field] = total
	}
	return stats.DocumentCount, totals, nil
}

func (i *Index) TermInfo(field, term string) (inverted.TermInfo, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var info inverted.TermInfo
	err := getJSON(i.store, termKey(field, term), &info)
	if errors.Is(err, storage.ErrNotFound) {
		return inverted.TermInfo{}, nil
	}
	return info, err
}

func (i *Index) Postings(field, term string) ([]inverted.Posting, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	iterator := i.store.Scan(termPostingPrefix(field, term))
	defer iterator.Close()
	postings := make([]inverted.Posting, 0)
	for iterator.Next() {
		posting, err := inverted.DecodePosting(iterator.Value())
		if err != nil {
			return nil, err
		}
		postings = append(postings, posting)
	}
	return postings, iterator.Error()
}

// Terms returns all terms in a field matching a raw term prefix.
func (i *Index) Terms(field, prefix string) ([]string, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	iterator := i.store.Scan(fieldTermPrefix(field))
	defer iterator.Close()
	terms := make([]string, 0)
	for iterator.Next() {
		term, err := decodeComponent(lastKeyComponent(iterator.Key()))
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(term, prefix) {
			terms = append(terms, term)
		}
	}
	sort.Strings(terms)
	return terms, iterator.Error()
}

func (i *Index) AllDocumentIDs() ([]uint64, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	iterator := i.store.Scan(docMetaPrefix)
	defer iterator.Close()
	ids := make([]uint64, 0)
	for iterator.Next() {
		docID, err := parseDocIDPart(lastKeyComponent(iterator.Key()))
		if err != nil {
			return nil, err
		}
		ids = append(ids, docID)
	}
	return ids, iterator.Error()
}
