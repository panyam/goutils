package http

import "fmt"

func ExampleJsonToQueryString() {
	input := map[string]any{"a": 1, "b": 2}
	queryStr := JsonToQueryString(input)
	fmt.Println(queryStr)

	// Output: a=1&b=2
}
