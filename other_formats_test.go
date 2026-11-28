// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fountain"
)

const otherScript = `Title: The Visit

INT. HALL - DAY

Anna waits.

ANNA
Is he coming?

BRAM ^
Soon.
`

// Fade In, Trelby and KIT Scenarist files open as imports, by
// Lexington's list of what it reads; Save As writes a copy in them, the
// window staying on its own file.
func TestOtherProgramsFormats(t *testing.T) {
	dir := t.TempDir()
	script := fountain.Parse(englishScenes, strings.NewReader(otherScript))
	for _, ext := range []string{".fadein", ".osf", ".trelby", ".kitsp"} {
		if _, ok := otherFormat("film" + ext); !ok {
			t.Errorf("%s is not another program's format", ext)
		}
		if name, kind := extFormat(ext); kind != kindScreenplay || strings.HasPrefix(name, strings.ToUpper(ext[1:])) {
			t.Errorf("%s: %q, %q", ext, name, kind)
		}
		p := filepath.Join(dir, "film"+ext)
		if err := cli.Write(context.Background(), script, cli.Options{To: ext[1:], Output: p}); err != nil {
			t.Fatal(err)
		}
		w := newTestWindow(t, "")
		if err := w.LoadFile(p); err != nil {
			t.Fatalf("%s: %v", ext, err)
		}
		text := w.textEditor.Text()
		if !strings.Contains(text, "ANNA") || !strings.Contains(text, "INT. HALL - DAY") || !strings.Contains(text, "Soon.") {
			t.Errorf("%s opened as\n%s", ext, text)
		}
		if w.currentFile == p {
			t.Errorf("%s: the import is the window's file", ext)
		}
	}
	for _, ext := range []string{".fadein", ".trelby", ".kitsp"} {
		found := false
		for _, e := range saveExtensions() {
			found = found || e == ext
		}
		if !found {
			t.Errorf("Save As has no %s", ext)
		}
	}
	w := newTestWindow(t, otherScript)
	own := filepath.Join(dir, "own.fountain")
	w.currentFile = own
	copyTo := filepath.Join(dir, "copy.trelby")
	w.saveAsWith(copyTo, nil)
	if data, err := os.ReadFile(copyTo); err != nil || !strings.Contains(string(data), "#Start-Script") {
		t.Fatalf("no Trelby copy: %v %.60q", err, data)
	}
	if w.currentFile != own {
		t.Errorf("Save As a copy made %s the window's file", w.currentFile)
	}
}
