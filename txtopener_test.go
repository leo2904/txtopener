package txtopener

import (
	"io/ioutil"
	"strings"
	"testing"
)

var utf8bom = []byte{0xef, 0xbb, 0xbf}
var utf16lebom = []byte{0xff, 0xfe}
var utf16bebom = []byte{0xfe, 0xff}

func TestNewReader(t *testing.T) {
	var tests = []struct {
		feed     string
		expected string
	}{
		{"", ""},
		{"a", "a"},
		{"ab", "ab"},
		{"abc", "abc"},
		{"abcd", "abcd"},
		{"Leonardo", "Leonardo"},
		{"paral·lel", "paral·lel"},
		{"façade", "façade"},
		{`pingüino`, "pingüino"},
		{string(utf8bom), ""},
		{string(utf8bom) + "pingüino", "pingüino"},
		{string(utf16lebom), ""},
		{string(utf16bebom), ""},
	}

	for i, tt := range tests {
		got, err := ioutil.ReadAll(NewReader(strings.NewReader(tt.feed)))
		if err != nil {
			t.Errorf("error en ReadAll: %v", err)
		}
		if string(got) != tt.expected {
			t.Errorf("%d. feeded: %s -> got: %s - expected: %s", i, tt.feed, got, tt.expected)
		}
	}
}
