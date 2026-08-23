package index

import (
	"fmt"

	"github.com/23jdd/Koris/inverted"
)

// Check performs a complete index consistency audit without modifying data.
func (i *Index) Check() (ConsistencyReport, error) {
	ids, err := i.AllDocumentIDs()
	if err != nil {
		return ConsistencyReport{}, err
	}
	stats, err := i.Stats()
	if err != nil {
		return ConsistencyReport{}, err
	}
	report := ConsistencyReport{Documents: uint64(len(ids))}
	if report.Documents != stats.DocumentCount {
		report.Problems = append(report.Problems, fmt.Sprintf("document count: metadata=%d actual=%d", stats.DocumentCount, report.Documents))
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	iterator := i.store.Scan(termPrefix)
	defer iterator.Close()
	for iterator.Next() {
		report.Terms++
		parts := splitTermKey(iterator.Key())
		if len(parts) != 2 {
			report.Problems = append(report.Problems, "invalid term key: "+string(iterator.Key()))
			continue
		}
		field, err1 := decodeComponent(parts[0])
		term, err2 := decodeComponent(parts[1])
		if err1 != nil || err2 != nil {
			report.Problems = append(report.Problems, "invalid term encoding: "+string(iterator.Key()))
			continue
		}
		var info inverted.TermInfo
		if err := decodeJSONValue(iterator.Value(), &info); err != nil {
			return report, err
		}
		postings := i.store.Scan(termPostingPrefix(field, term))
		var count uint64
		for postings.Next() {
			count++
			report.Postings++
		}
		if err := postings.Error(); err != nil {
			postings.Close()
			return report, err
		}
		postings.Close()
		if count != info.DocumentFrequency {
			report.Problems = append(report.Problems, fmt.Sprintf("term %s:%s df=%d postings=%d", field, term, info.DocumentFrequency, count))
		}
	}
	return report, iterator.Error()
}
