package main

import (
	"embed"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// A script format is a file like Lexington's presets (name.toml): its
// name, the format it starts from (Base) and the elements that differ,
// and Accolade's part, which Lexington skips: the buttons the toolbar
// shows for it, inserting what that kind of script uses often (pages and
// panels in a comic, acts in a series, blackouts in a play):
//
//	[[Accolade.Buttons]]
//	Label = "Panel"
//	Insert = "Panel {n}. "
//	Restart = ".PAGE"
//
// {n} numbers it ({N} in words: ONE, TWO): one more than the lines up to
// the cursor that start as Insert does (up to the number) with a number,
// counting again after a line starting with Restart. A page break (===)
// starting Insert is left out for number 1. Line = true puts it on the
// next line, not in a paragraph of its own.
//
// Accolade's own formats (formats/) are written to builtinFormatDir; one's
// own, and changed copies of those, are in userFormatDir, over them.

//go:embed formats/*.toml
var builtinFormats embed.FS

// builtinFormatDir is where Accolade's formats are, as files for Lexington
// (written there if they are not yet).
func builtinFormatDir() string {
	dir := filepath.Join(getConfigDir(), "formats-builtin")
	if !fileExists(dir) {
		writeBuiltinFormats(dir)
	}
	return dir
}

// userFormatDir is where one's own formats are.
func userFormatDir() string { return filepath.Join(getConfigDir(), "formats") }

// formatDirs are the format folders as Lexington takes them (a path list).
func formatDirs() string {
	return builtinFormatDir() + string(os.PathListSeparator) + userFormatDir()
}

// writeBuiltinFormats puts Accolade's formats where Lexington reads them
// (again each start: they change with Accolade).
func writeBuiltinFormats(dir string) {
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	files, _ := builtinFormats.ReadDir("formats")
	for _, f := range files {
		if b, err := builtinFormats.ReadFile("formats/" + f.Name()); err == nil {
			_ = os.WriteFile(filepath.Join(dir, f.Name()), b, 0o644)
		}
	}
}

type formatButton struct {
	Label   string
	Insert  string
	Restart string `toml:",omitempty"`
	// Line: on the line after the cursor's, not as a paragraph of its own
	// (lyrics go under the singer's name, line after line)
	Line bool `toml:",omitempty"`
}

// formatFile is the file of a format: one's own, else Accolade's ("" if
// neither has one).
func formatFile(key string) string {
	for _, dir := range []string{userFormatDir(), builtinFormatDir()} {
		if p := filepath.Join(dir, key+".toml"); fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// formatButtons are the buttons of the format with key, from its file.
func formatButtons(key string) []formatButton {
	var f struct {
		Accolade struct{ Buttons []formatButton }
	}
	if p := formatFile(key); p != "" {
		_, _ = toml.DecodeFile(p, &f)
	}
	return f.Accolade.Buttons
}

var numberWords = strings.Fields("ONE TWO THREE FOUR FIVE SIX SEVEN EIGHT NINE TEN ELEVEN TWELVE THIRTEEN FOURTEEN FIFTEEN SIXTEEN SEVENTEEN EIGHTEEN NINETEEN TWENTY")

// numberWord is n in words, as acts and pages are numbered (ONE, TWO),
// in figures past twenty.
func numberWord(n int) string {
	if n >= 1 && n <= len(numberWords) {
		return numberWords[n-1]
	}
	return strconv.Itoa(n)
}

// numbered reports whether rest (a line after a button's prefix) starts
// with a number, in figures or words: "> 3. WHO" is a song, "> BLACKOUT."
// is not.
func numbered(rest string) bool {
	word, _, _ := strings.Cut(strings.TrimSpace(rest), " ")
	word = strings.TrimRight(word, ".:<")
	if word == "" {
		return false
	}
	if word[0] >= '0' && word[0] <= '9' {
		return true
	}
	for _, w := range numberWords {
		if strings.EqualFold(word, w) {
			return true
		}
	}
	return false
}

// insertButton inserts a button's text as a paragraph of its own at the
// cursor's line, numbered, with the cursor after it.
func insertButton(b *buffer.Buffer, fb formatButton) {
	line, _ := b.Position(b.Cursor())
	n := 1
	if i := strings.IndexAny(fb.Insert, "{"); i >= 0 && (strings.HasPrefix(fb.Insert[i:], "{n}") || strings.HasPrefix(fb.Insert[i:], "{N}")) {
		prefix := strings.TrimSpace(strings.TrimPrefix(fb.Insert[:i], "===\n\n"))
		for l := 0; l <= line; l++ {
			t := strings.TrimSpace(b.Line(l))
			switch {
			case fb.Restart != "" && strings.HasPrefix(t, fb.Restart):
				n = 1
			case prefix != "" && strings.HasPrefix(t, prefix) && numbered(t[len(prefix):]):
				n++
			}
		}
	}
	text := strings.ReplaceAll(fb.Insert, "{n}", strconv.Itoa(n))
	text = strings.ReplaceAll(text, "{N}", numberWord(n))
	if n == 1 {
		text = strings.TrimPrefix(text, "===\n\n")
	}
	at := b.LineEnd(line)
	switch {
	case fb.Line && strings.TrimSpace(b.Line(line)) != "":
		text = "\n" + text
	case strings.TrimSpace(b.Line(line)) == "":
		at = b.LineStart(line)
		if line > 0 && strings.TrimSpace(b.Line(line-1)) != "" {
			text = "\n" + text
		}
	default:
		text = "\n\n" + text
	}
	b.Replace(at, at, text)
	b.SetCursor(at+len([]rune(text)), false)
}
