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
