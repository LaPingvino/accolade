// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// spdxOwn is the license of Accolade's own code (LICENSING.md).
const spdxOwn = "MIT OR GPL-3.0-or-later"

// spdxOther are the files that hold others' work too, with their whole
// license expression; LICENSING.md lists them.
var spdxOther = map[string]string{
	"icons.go": "(MIT OR GPL-3.0-or-later) AND Apache-2.0", // Material Design icons
}

// TestSPDXHeaders: every Go file, and every script, starts with its SPDX
// line (after a script's #! line), and the license is Accolade's own
// unless LICENSING.md lists the file as holding others' work.
func TestSPDXHeaders(t *testing.T) {
	comment := map[string]string{".go": "//", ".js": "//", ".swift": "//", ".sh": "#", ".ps1": "#"}
	seen := map[string]bool{}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(d.Name(), ".") && d.Name() != ".github" || d.Name() == "dist") {
				return filepath.SkipDir
			}
			return nil
		}
		c, ok := comment[filepath.Ext(path)]
		if !ok {
			return nil
		}
		path = filepath.ToSlash(path)
		want, other := spdxOther[path]
		if other {
			seen[path] = true
		} else {
			want = spdxOwn
		}
		line, err := spdxLine(path)
		if err != nil {
			return err
		}
		if line != c+" SPDX-License-Identifier: "+want {
			t.Errorf("%s: first line %q, want %q", path, line, c+" SPDX-License-Identifier: "+want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for path := range spdxOther {
		if !seen[path] {
			t.Errorf("spdxOther lists %s, which is gone", path)
		}
	}
}

// spdxLine is a file's first line, or its second after a #! line.
func spdxLine(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSuffix(s.Text(), "\r")
		if !strings.HasPrefix(line, "#!") {
			return line, nil
		}
	}
	return "", s.Err()
}
