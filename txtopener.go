// Package txtopener provides helper functions to read files encoded as UTF-8, BOMBed UTF-8, BOMBed UTF-16-LE,
// BOMBed UTF-16-BE or any other known encoding as if they were UTF-8 files.
// For all BOMbed files the BOM is stripped out.
// All files without a BOM are treating with the reader provided by charset.NewReader() in order to get translated
// from the original character encoding to UTF-8

package txtopener

import (
	"bytes"
	"io"
	"mime"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// MustOpenAndClose calls os.Open and returns a reader that converts the content to UTF-8 without BOM
// and a function to close the file who panics if there is an error
func MustOpenAndClose(name string) (io.Reader, func()) {
	file, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	return NewReader(file), func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}
}

// NewReader returns an io.Reader that converts the content of r to UTF-8 without BOM.
// It calls charset.DetermineEncoding() to find out what r's enconding is
func NewReader(r io.Reader) io.Reader {
	nr, err := newReader(r, "")
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

// newReader returns an io.Reader that converts the content of r to UTF-8.
// It calls DetermineEncoding to find out what r's encoding is.
func newReader(r io.Reader, contentType string) (io.Reader, error) {
	preview := make([]byte, 10240)
	n, err := io.ReadFull(r, preview)
	switch {
	case err == io.ErrUnexpectedEOF:
		preview = preview[:n]
	case err != nil:
		return nil, err
	}
	r = io.MultiReader(bytes.NewReader(preview), r)

	if e, _, _ := determineEncoding(preview, contentType); e != encoding.Nop {
		r = transform.NewReader(r, e.NewDecoder())
	}
	return r, nil
}

// determineEncoding determines the encoding of an HTML document by examining
// up to the first 10240 bytes of content and the declared Content-Type.
//
// See http://www.whatwg.org/specs/web-apps/current-work/multipage/parsing.html#determining-the-character-encoding
func determineEncoding(content []byte, contentType string) (e encoding.Encoding, name string, certain bool) {
	if len(content) > 10240 {
		content = content[:10240]
	}

	for _, b := range boms {
		if bytes.HasPrefix(content, b.bom) {
			e, name = charset.Lookup(b.enc)
			return e, name, true
		}
	}

	if _, params, err := mime.ParseMediaType(contentType); err == nil {
		if cs, ok := params["charset"]; ok {
			if e, name = charset.Lookup(cs); e != nil {
				return e, name, true
			}
		}
	}

	if len(content) > 0 {
		e, name = prescan(content)
		if e != nil {
			return e, name, false
		}
	}

	// Try to detect UTF-8.
	// First eliminate any partial rune at the end.
	for i := len(content) - 1; i >= 0 && i > len(content)-4; i-- {
		b := content[i]
		if b < 0x80 {
			break
		}
		if utf8.RuneStart(b) {
			content = content[:i]
			break
		}
	}
	hasHighBit := false
	for _, c := range content {
		if c >= 0x80 {
			hasHighBit = true
			break
		}
	}
	if hasHighBit && utf8.Valid(content) {
		return encoding.Nop, "utf-8", false
	}

	// TODO: change default depending on user's locale?
	// return charmap.Windows1252, "windows-1252", false
	return charmap.ISO8859_1, "ISO 8859-1", false
}

func prescan(content []byte) (e encoding.Encoding, name string) {
	z := html.NewTokenizer(bytes.NewReader(content))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return nil, ""

		case html.StartTagToken, html.SelfClosingTagToken:
			tagName, hasAttr := z.TagName()
			if !bytes.Equal(tagName, []byte("meta")) {
				continue
			}
			attrList := make(map[string]bool)
			gotPragma := false

			const (
				dontKnow = iota
				doNeedPragma
				doNotNeedPragma
			)
			needPragma := dontKnow

			name = ""
			e = nil
			for hasAttr {
				var key, val []byte
				key, val, hasAttr = z.TagAttr()
				ks := string(key)
				if attrList[ks] {
					continue
				}
				attrList[ks] = true
				for i, c := range val {
					if 'A' <= c && c <= 'Z' {
						val[i] = c + 0x20
					}
				}

				switch ks {
				case "http-equiv":
					if bytes.Equal(val, []byte("content-type")) {
						gotPragma = true
					}

				case "content":
					if e == nil {
						name = fromMetaElement(string(val))
						if name != "" {
							e, name = charset.Lookup(name)
							if e != nil {
								needPragma = doNeedPragma
							}
						}
					}

				case "charset":
					e, name = charset.Lookup(string(val))
					needPragma = doNotNeedPragma
				}
			}

			if needPragma == dontKnow || needPragma == doNeedPragma && !gotPragma {
				continue
			}

			if strings.HasPrefix(name, "utf-16") {
				name = "utf-8"
				e = encoding.Nop
			}

			if e != nil {
				return e, name
			}
		}
	}
}

func fromMetaElement(s string) string {
	for s != "" {
		csLoc := strings.Index(s, "charset")
		if csLoc == -1 {
			return ""
		}
		s = s[csLoc+len("charset"):]
		s = strings.TrimLeft(s, " \t\n\f\r")
		if !strings.HasPrefix(s, "=") {
			continue
		}
		s = s[1:]
		s = strings.TrimLeft(s, " \t\n\f\r")
		if s == "" {
			return ""
		}
		if q := s[0]; q == '"' || q == '\'' {
			s = s[1:]
			closeQuote := strings.IndexRune(s, rune(q))
			if closeQuote == -1 {
				return ""
			}
			return s[:closeQuote]
		}

		end := strings.IndexAny(s, "; \t\n\f\r")
		if end == -1 {
			end = len(s)
		}
		return s[:end]
	}
	return ""
}

var boms = []struct {
	bom []byte
	enc string
}{
	{[]byte{0xfe, 0xff}, "utf-16be"},
	{[]byte{0xff, 0xfe}, "utf-16le"},
	{[]byte{0xef, 0xbb, 0xbf}, "utf-8"},
}
