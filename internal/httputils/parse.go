package httputils

import (
	"strconv"
)

func ParsePositiveInt(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
