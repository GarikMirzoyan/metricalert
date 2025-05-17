package utils

import (
	"strconv"
	"strings"
)

func FormatNumber(num float64) string {
	rounded := strconv.FormatFloat(num, 'f', 3, 64)
	rounded = strings.TrimRight(rounded, "0")
	rounded = strings.TrimRight(rounded, ".")
	return rounded
}
