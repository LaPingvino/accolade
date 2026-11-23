package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every hand-over from another goroutine to the UI goes through uiDo (or
// inBackground, after, longPressAfter): a plain fyne.Do runs, under the
// test driver, on the background goroutine beside the test (CI's race job,
// 8d93f6a). Only background.go calls fyne.Do, besides the two places
// below that are no hand-over. The _js.go files are read too, whatever
// the build.
func TestNoPlainFyneDo(t *testing.T) {
	allowed := map[string]string{
		"background.go": "uiDo itself",
		// FocusGained/FocusLost put the relayout after the focus event, on
		// the UI goroutine: no other goroutine's work
		"internal/editor/editor.go": "a relayout after a focus event, on the UI goroutine",
		// the default runner of a tip's delay; Accolade sets uiDo (background.go)
		"internal/tooltip/tooltip.go": "the package's default runner, replaced by uiDo",
	}
	plain := regexp.MustCompile(`\bfyne\.Do\b`)
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" || d.Name() == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || allowed[filepath.ToSlash(path)] != "" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(b), "\n") {
			code, _, _ := strings.Cut(line, "//")
			if plain.MatchString(code) {
				t.Errorf("%s:%d: plain fyne.Do — hand over with uiDo or inBackground (background.go): %s", path, i+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
