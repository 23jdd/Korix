package index

import (
	"fmt"

	"github.com/23jdd/Koris/inverted"
)

// Check 执行只读全量一致性审计，不修改任何索引数据。
//
// 当前检查两类关键不变量：全局 DocumentCount 与实际 docmeta 数一致；每个
// TermInfo.DocumentFrequency 与实际 posting 数一致。返回的 Problems 可直接用于
// 运维诊断；Store/解码错误则通过 error 返回，表示审计本身未能完成。
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
	// term 词典是审计入口：逐 term 扫描对应 posting 前缀并核对 DF。
	iterator := i.store.Scan(termPrefix)
	defer iterator.Close()
	for iterator.Next() {
		report.Terms++
		// key 结构和 component 解码也属于一致性检查，不能假设持久化数据可信。
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
