// Use only after values already validated
// For example after form validation
package httptools

import (
	"strconv"
	"time"
)

func MustStrToInt64(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}

func MustStrToInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return v
}

func MustParseTime(s string, layout string) time.Time {
	v, err := time.Parse(layout, s)
	if err != nil {
		panic(err)
	}
	return v
}
