package utils

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func Test1(t *testing.T) {
	a, err := SetMapFields(nil,
		"a/b", 3,
		"a/c/d/e", "hello world",
		"a/c/f", []string{"hello", "universe"},
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, a, map[string]interface{}{
		"a": map[string]interface{}{
			"b": 3,
			"c": map[string]interface{}{
				"d": map[string]interface{}{
					"e": "hello world",
				},
				"f": []string{
					"hello",
					"universe",
				},
			},
		},
	})

	val, err := GetMapField(a, "a/b")
	assert.Equal(t, err, nil)
	assert.Equal(t, val, 3)

	val, err = GetMapField(a, strings.Split("a/b", "/"))
	assert.Equal(t, err, nil)
	assert.Equal(t, val, 3)

	val, err = GetMapField(a, strings.Split("a/c/d/e", "/"))
	assert.Equal(t, err, nil)
	assert.Equal(t, val, "hello world")

	val, err = GetMapField(a, strings.Split("a/b/c", "/"))
	assert.Equal(t, err.Error(), "field_path (a/b/c) at index 1 is not a map")
	assert.Equal(t, val, nil)

	val, err = GetMapField(a, strings.Split("a/c/d/x", "/"))
	assert.Equal(t, err, nil)
	assert.Equal(t, val, nil)
}

func TestCopyFields(t *testing.T) {
	a, err := SetMapFields(nil,
		"a/b", 3,
		"a/c/d/e", "hello world",
		"a/c/d", []string{"hello", "universe"},
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, a, map[string]interface{}{
		"a": map[string]interface{}{
			"b": 3,
			"c": map[string]interface{}{
				"d": []string{
					"hello",
					"universe",
				},
			},
		},
	})

	// now copy to b
	b, err := CopyMapFields(a, nil, "a/c/d", "x/y", "a/b", "l/m/n/o")
	assert.Equal(t, err, nil)
	assert.Equal(t, b, map[string]interface{}{
		"l": map[string]interface{}{
			"m": map[string]interface{}{
				"n": map[string]interface{}{
					"o": 3,
				},
			},
		},
		"x": map[string]interface{}{
			"y": []string{
				"hello",
				"universe",
			},
		},
	})
}

func TestReplace(t *testing.T) {
	a, err := SetMapFields(nil,
		"a/b", 3,
		"a/c/d/e", "hello world",
		"a/c/d", []string{"hello", "universe"},
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, a, map[string]interface{}{
		"a": map[string]interface{}{
			"b": 3,
			"c": map[string]interface{}{
				"d": []string{
					"hello",
					"universe",
				},
			},
		},
	})
}

func TestFailure(t *testing.T) {
	a, err := SetMapFields(nil,
		"a/b", 3,
		"a/b/c", "hello world",
	)

	assert.Equal(t, a, map[string]interface{}{
		"a": map[string]interface{}{
			"b": 3,
		},
	})
	assert.Equal(t, err.Error(), "field_path (a/b/c) at index 1 is not a map")
}
