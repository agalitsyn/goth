package httptools

import (
	"fmt"
	"net/http"
	"strconv"
)

func GetPathInt64(r *http.Request, key string) (int64, error) {
	raw := r.PathValue(key)
	if raw == "" {
		return 0, fmt.Errorf("path value '%s' is empty", key)
	}

	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse value to number: %w", err)
	}
	return parsed, nil
}

func GetQueryInt64(r *http.Request, key string) (int64, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse value to number: %w", err)
	}
	return parsed, nil
}
