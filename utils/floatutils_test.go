package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPriceString(t *testing.T) {
	assert.Equal(t, PriceString(3), "3")
	assert.Equal(t, PriceString(3.2), "3.2")
	assert.Equal(t, PriceString(3.19), "3.19")
	assert.Equal(t, PriceString(3.19999), "3.2")
}
