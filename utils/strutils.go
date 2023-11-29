package utils

import (
	"math/rand"
	"net/url"
	"strings"
)

func EncodeURIComponent(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	return r
}

func AlphaReverseMap(alphabet string, out map[rune]uint) map[rune]uint {
	if out == nil {
		out = make(map[rune]uint)
	}

	for i, ch := range alphabet {
		out[ch] = uint(i)
	}
	return out
}

func ExcelEncode(value uint64, alphabet string) string {
	out := ""
	base := uint64(len(alphabet))
	for value > 0 {
		reminder := value % base
		out = string(alphabet[reminder]) + out
		value /= base
	}
	return out
}

func ExcelDecode(encoded string, alphabet string, revmap map[rune]uint) uint64 {
	base := uint64(len(alphabet))
	var n uint64 = 0
	for _, c := range encoded {
		n = n*base + uint64(revmap[c])
	}
	return n
}

func RandString(length int, chars string) string {
	if chars == "" {
		chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	}
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func PaddedWith(input string, padding byte) string {
	ch := string(padding)
	for len(input)%4 != 0 {
		input += ch
	}
	return input
}
