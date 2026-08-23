package inverted

// Intersect returns postings present in both inputs. Frequencies and positions
// are retained from the left side; query scoring looks up each source term.
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
