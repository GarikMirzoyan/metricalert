package utils

import (
	"strconv"
	"strings"
)

// FormatNumber formats float with up to three decimals, trimming trailing zeros.
func FormatNumber(num float64) string {
	rounded := strconv.FormatFloat(num, 'f', 3, 64)
	rounded = strings.TrimRight(rounded, "0")
	rounded = strings.TrimRight(rounded, ".")
	return rounded
}
