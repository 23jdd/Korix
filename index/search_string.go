package index

import "github.com/23jdd/Koris/query"

func (i *Index) SearchString(expression, defaultField string, limit int) ([]query.Hit, error) {
	parsed, err := query.Parse(expression, defaultField)
	if err != nil {
		return nil, err
	}
	return i.Search(parsed, limit)
}
