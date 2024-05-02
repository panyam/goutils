package utils

import (
	"strconv"
)

func PriceString(value float64) string {
	intval := int64(0.5 + value*100)
	return strconv.FormatFloat(float64(intval)/100.0, 'f', -1, 64)
}
