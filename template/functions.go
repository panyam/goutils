package template

import (
	"html/template"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/panyam/goutils/utils"
)

func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"Now":           time.Now,
		"HTML":          func(s string) template.HTML { return template.HTML(s) },
		"JS":            func(s string) template.JS { return template.JS(s) },
		"BytesToString": func(input []byte) string { return string(input) },
		"URL":           func(s string) template.URL { return template.URL(s) },
		"RangeN":        func(n int) []struct{} { return make([]struct{}, n) },
		"TypeOf":        reflect.TypeOf,
		"ExpandAttrs":   ExpandAttrs,
		"Slice":         SliceArray,
		"ToJson":        utils.ToJson,
		"AddInts":       AddNums[int],
		"MultInts":      MultNums[int],
		"SubInt":        SubNums[int],
		"DivInt":        IntDiv[int, int],
		"String":        ToString,
		"Int":           ToInt,
		"Float":         ToFloat,
		"IntList":       NumList[int],
		"Add": func(vals ...any) (out float64) {
			for _, v := range vals {
				out += ToFloat(v)
			}
			return
		},
		"Multiply": func(vals ...any) (out float64) {
			for _, v := range vals {
				out *= ToFloat(v)
			}
			return
		},
		"Sub": func(a any, b any) (out float64) { return ToFloat(a) - ToFloat(b) },
		"Div": func(a any, b any) float64 { return ToFloat(a) / ToFloat(b) },
		"Floor": func(val float64) int64 {
			return int64(val)
		},
		"Ceil": func(val float64) int {
			return int(val + 0.5)
		},

		"JoinA": func(delim string, parts []string) string {
			return strings.Join(parts, delim)
		},
		"Join": func(delim string, parts ...string) string {
			return strings.Join(parts, delim)
		},
		"Split":     strings.Split,
		"HasPrefix": strings.HasPrefix,
		"HasSuffix": strings.HasSuffix,
		"Replace":   strings.Replace,
		"Slugify":   Slugify,
		"dict":      ValuesToDict,
		"Ago":       TimeSinceString,
		"DBG": func(values ...any) string {
			log.Println(values)
			return ""
		},
	}
}
