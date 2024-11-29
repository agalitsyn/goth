package pagination

import (
	"errors"
	"fmt"
	"strings"
)

type Order string

const (
	OrderAsc  Order = "ASC"
	OrderDesc Order = "DESC"
)

func OrderFromString(s string) (Order, error) {
	if s == "" {
		return "", nil
	}
	switch strings.ToUpper(s) {
	case "ASC":
		return OrderAsc, nil
	case "DESC":
		return OrderDesc, nil
	default:
		return "", errors.New("sort direction can be 'ASC' or 'DESC' only")
	}
}

type Sort struct {
	Order Order
	By    string
}

const defaultMaxLimit = 1000

type Pagination struct {
	Offset uint64
	Limit  uint64
	Sort   []Sort
}

// Validate returns error on invalid pagination
// sortByMapping is mapping between client and storage for dynamic column name in ORDER BY statement
// "field name from client request" => "field name in database"
func (f *Pagination) Validate(sortByMapping map[string]string) error {
	for _, sort := range f.Sort {
		if sort.By != "" {
			allowedSortFields := make([]string, 0, len(sortByMapping))
			for k := range sortByMapping {
				allowedSortFields = append(allowedSortFields, k)
			}

			got := strings.ToLower(sort.By)
			column, ok := sortByMapping[got]
			if !ok {
				return fmt.Errorf(
					"allowed sort fields: %s, got: %s",
					strings.Join(allowedSortFields, ", "),
					sort.By,
				)
			}
			sort.By = column
		}
	}

	if f.Limit > defaultMaxLimit {
		return fmt.Errorf("limit should be lower than %d", defaultMaxLimit)
	}

	return nil
}
