package inverted

// Intersect 对两个已按 DocID 排序的 posting list 求交集。
// 命中时保留左侧 Posting；Query 会分别从原始 term list 计算分数，因此这里不应
// 合并不属于同一个 term 的 Frequency 或 Positions。复杂度为 O(len(left)+len(right))。
func Intersect(left, right []Posting) []Posting {
	result := make([]Posting, 0)
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		switch {
		case left[i].DocID < right[j].DocID:
			i++
		case left[i].DocID > right[j].DocID:
			j++
		default:
			result = append(result, left[i])
			i++
			j++
		}
	}
	return result
}

// Union 对两个已排序 posting list 求去重并集。相同 DocID 保留左侧 Posting，
// 结果仍按 DocID 排序，可继续参与后续 merge。
func Union(left, right []Posting) []Posting {
	result := make([]Posting, 0, len(left)+len(right))
	i, j := 0, 0
	for i < len(left) || j < len(right) {
		switch {
		case j >= len(right) || (i < len(left) && left[i].DocID < right[j].DocID):
			result = append(result, left[i])
			i++
		case i >= len(left) || right[j].DocID < left[i].DocID:
			result = append(result, right[j])
			j++
		default:
			result = append(result, left[i])
			i++
			j++
		}
	}
	return result
}
