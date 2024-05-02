package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParseAndFormatDate(t *testing.T) {
	testtime := time.Date(2022, 12, 20, 01, 02, 03, 04, time.UTC)
	formatted := FormatDate(testtime)
	assert.Equal(t, formatted, "2022_12_20", "Formatted data doesnt match")
	parsedtime := ParseDate(formatted)
	assert.Equal(t, parsedtime, time.Date(2022, 12, 20, 0, 0, 0, 0, time.UTC), "Parsed date doesnt match")
}

func TestParseAndFormatTime(t *testing.T) {
	testtime := time.Date(2022, 12, 20, 01, 02, 03, 04, time.UTC)
	formatted := FormatTime(testtime)
	assert.Equal(t, formatted, "2022_12_20 01-02-03", "Formatted data doesnt match")
	parsedtime := ParseTime(formatted)
	assert.Equal(t, parsedtime, time.Date(2022, 12, 20, 01, 02, 03, 0, time.UTC), "Parsed date doesnt match")
}
