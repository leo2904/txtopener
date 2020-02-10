// Package txtopener provides helper functions to read files encoded as UTF-8, BOMBed UTF-8, BOMBed UTF-16-LE,
// BOMBed UTF-16-BE or any other known encoding as if they were UTF-8 files.
// For all BOMbed files the BOM is stripped out.
// All files without a BOM are treating with the reader provided by charset.NewReader() in order to get translated
// from the original character encoding to UTF-8

package txtopener

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/net/html/charset"
)

// MustOpenAndClose calls os.Open and returns a reader that converts the content to UTF-8 without BOM
// and a function to close the file that calls sync before close or panics otherwise
func MustOpenAndClose(name string) (io.Reader, func()) {
	file, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	return NewReader(file), func() {
		if err := file.Sync(); err != nil {
			panic(err)
		}
		if err := file.Close(); err != nil {
			panic(err)
		}
	}
}

// NewReader returns an io.Reader that converts the content of r to UTF-8 without BOM.
// It calls charset.DetermineEncoding() to find out what r's enconding is
func NewReader(r io.Reader) io.Reader {
	nr, err := charset.NewReader(r, "")
	if err != nil {
		if err == io.EOF {
			return r
		}
		panic(err)
	}

	// discarding the utf-8 BOM mark (EF BB BF)
	bom := make([]byte, 3)
	if n, err := io.ReadFull(nr, bom); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			panic(err)
		}
		if n < len(bom) {
			return bytes.NewReader(bom[:n])
		}
	}

	if bom[0] != 0xef || bom[1] != 0xbb || bom[2] != 0xbf {
		nr = io.MultiReader(bytes.NewReader(bom), nr)
	}
	return nr
}
