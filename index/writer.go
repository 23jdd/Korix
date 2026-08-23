package index

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/inverted"
	"github.com/23jdd/Koris/storage"
)

// Add inserts or atomically replaces a document with the same external ID.
func (i *Index) Add(doc document.Document) (uint64, error) {
	ids, err := i.AddBatch([]document.Document{doc})
	if err != nil {
		return 0, err
	}
	return ids[0], nil
}

// AddBatch indexes all documents in one transaction. Any error rolls back the
// entire batch and its global statistics.
func (i *Index) AddBatch(documents []document.Document) ([]uint64, error) {
	if len(documents) == 0 {
		return nil, nil
	}
	for _, doc := range documents {
		if err := validateDocument(doc); err != nil {
			return nil, err
		}
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	ids := make([]uint64, len(documents))
	err := i.store.Transaction(func(tx storage.Tx) error {
		var global GlobalStats
		if err := getJSON(tx, globalMetaKey, &global); err != nil {
			return err
		}
		ensureGlobalMaps(&global)
		for n, doc := range documents {
			docID, err := i.addInTransaction(tx, &global, doc)
			if err != nil {
				return err
			}
			ids[n] = docID
		}
		return putJSON(tx, globalMetaKey, global)
	})
	return ids, err
}

func (i *Index) addInTransaction(tx storage.Tx, global *GlobalStats, doc document.Document) (uint64, error) {
	docID, exists, err := lookupExternalID(tx, doc.ID)
	if err != nil {
		return 0, err
	}
	if exists {
		var old storedDocumentMetadata
		if err := getJSON(tx, documentMetaKey(docID), &old); err != nil {
			return 0, err
		}
		if err := removeVectors(tx, &old, global); err != nil {
			return 0, err
		}
	} else {
		docID = global.NextDocumentID
		if docID == 0 {
			docID = 1
		}
		global.NextDocumentID = docID + 1
		global.DocumentCount++
	}

	metadata := storedDocumentMetadata{
		DocumentMetadata: DocumentMetadata{DocID: docID, ExternalID: doc.ID, Lengths: make(map[string]uint32)},
		TermVectors:      make(map[string]map[string][]uint32),
	}
	fields := sortedFields(doc.Fields)
	for _, field := range fields {
		tokens := i.Analyzer(field).Analyze(doc.Fields[field])
		metadata.Lengths[field] = uint32(len(tokens))
		global.TotalFieldLength[field] += uint64(len(tokens))
		vectors := make(map[string][]uint32)
		for _, token := range tokens {
			if token.Term != "" {
				vectors[token.Term] = append(vectors[token.Term], token.Position)
			}
		}
		metadata.TermVectors[field] = vectors
		for term, positions := range vectors {
			posting := inverted.NewPosting(docID, positions)
			if err := tx.Put(postingKey(field, term, docID), inverted.EncodePosting(posting)); err != nil {
				return 0, err
			}
			var info inverted.TermInfo
			err := getJSON(tx, termKey(field, term), &info)
			if err != nil && !errors.Is(err, storage.ErrNotFound) {
				return 0, err
			}
			info.DocumentFrequency++
			info.TotalFrequency += uint64(len(positions))
			if err := putJSON(tx, termKey(field, term), info); err != nil {
				return 0, err
			}
		}
	}
	if err := putJSON(tx, documentKey(docID), doc.Clone()); err != nil {
		return 0, err
	}
	if err := putJSON(tx, documentMetaKey(docID), metadata); err != nil {
		return 0, err
	}
	if err := tx.Put(externalIDKey(doc.ID), encodeUint64(docID)); err != nil {
		return 0, err
	}
	return docID, nil
}

func (i *Index) Delete(externalID string) error {
	if strings.TrimSpace(externalID) == "" {
		return ErrInvalidDocument
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.store.Transaction(func(tx storage.Tx) error {
		docID, exists, err := lookupExternalID(tx, externalID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrDocumentMissing
		}
		var global GlobalStats
		if err := getJSON(tx, globalMetaKey, &global); err != nil {
			return err
		}
		var metadata storedDocumentMetadata
		if err := getJSON(tx, documentMetaKey(docID), &metadata); err != nil {
			return err
		}
		if err := removeVectors(tx, &metadata, &global); err != nil {
			return err
		}
		if global.DocumentCount > 0 {
			global.DocumentCount--
		}
		if err := tx.Delete(documentKey(docID)); err != nil {
			return err
		}
		if err := tx.Delete(documentMetaKey(docID)); err != nil {
			return err
		}
		if err := tx.Delete(externalIDKey(externalID)); err != nil {
			return err
		}
		return putJSON(tx, globalMetaKey, global)
	})
}

func removeVectors(tx storage.Tx, metadata *storedDocumentMetadata, global *GlobalStats) error {
	for field, terms := range metadata.TermVectors {
		length := uint64(metadata.Lengths[field])
		if global.TotalFieldLength[field] >= length {
			global.TotalFieldLength[field] -= length
		} else {
			global.TotalFieldLength[field] = 0
		}
		for term, positions := range terms {
			if err := tx.Delete(postingKey(field, term, metadata.DocID)); err != nil {
				return err
			}
			var info inverted.TermInfo
			if err := getJSON(tx, termKey(field, term), &info); err != nil {
				return err
			}
			if info.DocumentFrequency <= 1 {
				if err := tx.Delete(termKey(field, term)); err != nil {
					return err
				}
			} else {
				info.DocumentFrequency--
				if info.TotalFrequency >= uint64(len(positions)) {
					info.TotalFrequency -= uint64(len(positions))
				}
				if err := putJSON(tx, termKey(field, term), info); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateDocument(doc document.Document) error {
	if strings.TrimSpace(doc.ID) == "" || len(doc.Fields) == 0 {
		return ErrInvalidDocument
	}
	for name := range doc.Fields {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%w: empty field name", ErrInvalidDocument)
		}
	}
	return nil
}

func lookupExternalID(reader storage.Reader, externalID string) (uint64, bool, error) {
	data, err := reader.Get(externalIDKey(externalID))
	if errors.Is(err, storage.ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	docID, err := decodeUint64(data)
	return docID, true, err
}

func ensureGlobalMaps(global *GlobalStats) {
	if global.TotalFieldLength == nil {
		global.TotalFieldLength = make(map[string]uint64)
	}
	if global.NextDocumentID == 0 {
		global.NextDocumentID = 1
	}
}

func sortedFields(fields map[string]string) []string {
	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	sort.Strings(result)
	return result
}
