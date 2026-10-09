package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeEncodeScript(t *testing.T) {
	for name, raw := range map[string][]byte{
		"utf8":   []byte("INT. CAFÉ - DAY\n\nJosé.\n"),
		"cp1252": []byte("INT. CAF\xc9 - DAY\n\nJos\xe9 \x93hi\x94.\n"),
		"bom":    append([]byte{0xEF, 0xBB, 0xBF}, "Title: X\nAuthor: Y\n\nINT. A - DAY\n"...),
		"crlf":   []byte("INT. A - DAY\r\n\r\nAction.\r\n"),
	} {
		text, f := decodeScript(raw)
		if strings.ContainsRune(text, '�') || strings.Contains(text, "\r") || strings.HasPrefix(text, string(utf8BOM)) {
			t.Errorf("%s: decoded %q", name, text)
		}
		if back, utf8 := encodeScript(text, f); !bytes.Equal(back, raw) || utf8 {
			t.Errorf("%s: saved %q, want %q", name, back, raw)
		}
	}
	// a character Windows-1252 has not: saved as UTF-8, and told so
	text, f := decodeScript([]byte("Jos\xe9\n"))
	if back, utf8 := encodeScript(text+"→\n", f); !utf8 || string(back) != "José\n→\n" {
		t.Errorf("unrepresentable: %q %v", back, utf8)
	}
}

// Opening and saving a Windows-1252 script with a BOM-less, CRLF body
// writes the same bytes back; a symlink stays a symlink.
func TestScriptFileKeepsItsFormat(t *testing.T) {
	dir := t.TempDir()
	raw := []byte("Title: CAF\xc9\r\n\r\nINT. CAF\xc9 - DAY\r\n\r\nJos\xe9 sits.\r\n")
	real := filepath.Join(dir, "real.fountain")
	link := filepath.Join(dir, "link.fountain")
	os.WriteFile(real, raw, 0o644)
	if err := os.Symlink(real, link); err != nil {
		t.Skip("no symlinks here")
	}
	w, _ := newDialogTestWindow(t)
	if err := w.LoadFile(link); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.textEditor.Text(), "CAFÉ") {
		t.Fatalf("decoded %q", w.textEditor.Text())
	}
	if err := w.saveToFile(link); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink became a file")
	}
	if got, _ := os.ReadFile(real); !bytes.Equal(got, raw) {
		t.Errorf("saved %q, want %q", got, raw)
	}
}
