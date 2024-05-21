package http

import "fmt"

func ExampleJsonToQueryString() {
	input := map[string]any{"a": 1, "b": 2}
	queryStr := JsonToQueryString(input)
	fmt.Println(queryStr)

	// Output: a=1&b=2
}

func ExampleNormalizeWsUrl() {
	fmt.Println(NormalizeWsUrl("http://google.com"))
	fmt.Println(NormalizeWsUrl("https://github.com"))

	// Output:
	// ws://google.com
	// wss://github.com
}
