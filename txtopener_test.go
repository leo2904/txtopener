package txtopener

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
