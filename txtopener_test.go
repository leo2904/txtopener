package txtopener

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	utf8bom    = []byte{0xef, 0xbb, 0xbf}
	utf16lebom = []byte{0xff, 0xfe}
	utf16bebom = []byte{0xfe, 0xff}
)

type test struct {
	feed     []byte
	expected []byte
}

type tests []test

func (t *tests) AddString(feed, expected string) {
	t.AddByte([]byte(feed), []byte(expected))
}

func (t *tests) AddByte(feed, expected []byte) {
	tt := test{feed, expected}
	*t = append(*t, tt)
}

func TestNewReader(t *testing.T) {
	compareText := "callejón sin salida - ślepy zaułek - återvändsgränd - ћорсокак - αδιέξοδο - 死胡同 - 行き止まり"

	tests := make(tests, 0)
	tests.AddString("", "")
	tests.AddString("a", "a")
	tests.AddString("ab", "ab")
	tests.AddString("abc", "abc")
	tests.AddString("abcd", "abcd")
	tests.AddString(compareText, compareText)
	tests.AddByte(utf8bom, nil)
	tests.AddByte(append(utf8bom, compareText...), []byte(compareText))
	// tests.AddByte(append(utf16lebom, compareText...), []byte(compareText))

	for i, tt := range tests {
		got, err := ioutil.ReadAll(NewReader(bytes.NewReader(tt.feed)))
		if err != nil {
			t.Errorf("error en ReadAll: %v", err)
		}

		if !equalSlice(got, tt.expected) {
			t.Errorf("\n%d. feeded: % x\ngot     : % x\nexpected: % x\n", i, tt.feed, got, tt.expected)
		}
	}
}

func TestOpener(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.txt"))
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".txt")
		newFile := filepath.Join("testdata", "converted", name+"_UTF8.txt")
		in, closeIn := MustOpenAndClose(file)
		defer closeIn()

		out, err := os.Create(newFile)
		if err != nil {
			t.Fatal(err)
		}
		defer out.Close()

		scan := bufio.NewScanner(in)
		for scan.Scan() {
			_, err := fmt.Fprint(out, scan.Text())
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func equalSlice(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
