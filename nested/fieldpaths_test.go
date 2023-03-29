package reflect

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test1(t *testing.T) {
	a := make(map[string]interface{})
	err := SetMapFields(a,
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
}

func TestReplace(t *testing.T) {
	a := make(map[string]interface{})
	err := SetMapFields(a,
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
	a := make(map[string]interface{})
	err := SetMapFields(a,
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
