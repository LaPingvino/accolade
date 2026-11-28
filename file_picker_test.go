// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"github.com/LaPingvino/lexington/lex"
)

// TestPickFiltersComplete: every extension a dialog offers is in a group
// of its kind, and the groups come in their order.
func TestPickFiltersComplete(t *testing.T) {
	lists := map[string][]string{
		"open": openExtensions(), "import": importExtensions, "pandoc": pandocExtensions(),
		"merge": mergeExtensions(), "save": saveExtensions(), "pictures": pictureExtensions,
	}
	for name, exts := range lists {
		filters, kinds, _ := pickFilters(exts, "")
		for _, e := range exts {
			found := slices.ContainsFunc(filters, func(f dialog.NamedFilter) bool {
				return strings.HasPrefix(f.Name, " ") && slices.Contains(f.Filter.(*storage.ExtensionFileFilter).Extensions, strings.ToLower(e)) ||
					len(filters) == 1 && slices.Contains(f.Filter.(*storage.ExtensionFileFilter).Extensions, strings.ToLower(e))
			})
			if !found {
				t.Errorf("%s: %s is in no format", name, e)
			}
		}
		order := map[string]int{kindScreenplay: 0, kindNovel: 1, kindOutline: 2, "": 3}
		for i := 1; i < len(filters); i++ {
			if order[filters[i].Kind] < order[filters[i-1].Kind] {
				t.Errorf("%s: %q after %q", name, filters[i].Name, filters[i-1].Name)
			}
		}
		if name == "pictures" && len(kinds) != 0 {
			t.Errorf("pictures have kinds: %v", kinds)
		}
	}
}

func TestPickFiltersOpen(t *testing.T) {
	filters, kinds, first := pickFilters(openExtensions(), kindNovel)
	var names []string
	for _, f := range filters {
		names = append(names, f.Name)
	}
	if !slices.Equal(kinds, []string{kindScreenplay, kindNovel, kindOutline}) {
		t.Errorf("kinds %v", kinds)
	}
	for _, want := range []string{"All screenplay formats", " Fountain", " Final Draft (FDX)", " Screenplay (PDF)",
		"All novel formats", " Markdown", " Plain text", " Novel (PDF)", " Outline (OPML)", " Lexington settings (TOML)"} {
		if !slices.Contains(names, want) {
			t.Errorf("no %q in %q", want, names)
		}
	}
	if names[0] != "All screenplay formats" || names[first] != "All novel formats" {
		t.Errorf("first %q, chosen %q", names[0], names[first])
	}
	if _, _, first := pickFilters(openExtensions(), kindScreenplay); first != 0 {
		t.Errorf("a screenplay window's Open starts at %d", first)
	}
	// no kind preferred: the kind of the list's first format, never "other"
	for exts, want := range map[*[]string]string{&importExtensions: "All novel formats", &pictureExtensions: "All supported formats"} {
		filters, _, first := pickFilters(*exts, "")
		if filters[first].Name != want {
			t.Errorf("%v starts at %q", *exts, filters[first].Name)
		}
	}
	merge := mergeExtensions()
	if filters, _, first := pickFilters(merge, ""); filters[first].Name != "All screenplay formats" {
		t.Errorf("merge starts at %q", filters[first].Name)
	}
}

func TestTextIsScreenplay(t *testing.T) {
	for text, want := range map[string]bool{
		"Title: Big Fish\nAuthor: me\n\nINT. HOUSE - DAY\n":                                      true,
		"INT. KITCHEN - NIGHT\n\nShe waits.\n\nJOHN\nHello.\n\nMARY (V.O.)\nHi.\n":               true,
		"EXT. FIELD - DAY\n\nWind.\n\nEXT. ROAD - LATER\n\nMore wind.\n":                         true,
		"# Chapter One\n\nIt was a dark and stormy night. The rain fell.\n\nShe said: \"No.\"\n": false,
		"Dear friend,\n\nI hope this finds you well.\n\nYours,\nJoe\n":                           false,
		"NOTES\nsome things\n\nTODO\nmore\n":                                                     false,
	} {
		if got := textIsScreenplay(text); got != want {
			t.Errorf("%q: %v", text, got)
		}
	}
}

func TestScreenplayKind(t *testing.T) {
	line := func(ty lex.ElementType, n int) lex.Screenplay {
		var out lex.Screenplay
		for range n {
			out = append(out, lex.Line{Type: ty})
		}
		return out
	}
	script := slices.Concat(line(lex.TypeScene, 3), line(lex.TypeAction, 10), line(lex.TypeSpeaker, 8), line(lex.TypeDialog, 9))
	if k := screenplayKind(script); k != kindScreenplay {
		t.Errorf("script: %q", k)
	}
	// a book: chapters read as scenes, a poem as a speech
	book := slices.Concat(line(lex.TypeScene, 20), line(lex.TypeAction, 400), line(lex.TypeSpeaker, 4), line(lex.TypeDialog, 6))
	if k := screenplayKind(book); k != kindNovel {
		t.Errorf("book: %q", k)
	}
	if k := screenplayKind(line(lex.TypeEmpty, 9)); k != "" {
		t.Errorf("nothing: %q", k)
	}
}

func TestFileKindOfCached(t *testing.T) {
	p := filepath.Join(t.TempDir(), "notes.txt")
	os.WriteFile(p, []byte("INT. HOUSE - DAY\n\nJOHN\nHi.\n\nMARY\nHello.\n"), 0o644)
	if k := fileKindOf(p); k != kindScreenplay {
		t.Fatalf("%q", k)
	}
	if k := fileKindOf(filepath.Join(t.TempDir(), "a.fdx")); k != kindScreenplay { // by its name, unread
		t.Errorf("fdx %q", k)
	}
	os.WriteFile(p, []byte("Dear friend, a letter, long enough to be a letter."), 0o644)
	os.Chtimes(p, time.Now().Add(time.Hour), time.Now().Add(time.Hour))
	if k := fileKindOf(p); k != kindNovel { // changed: read again
		t.Errorf("changed %q", k)
	}
}

// TestSniffSpeed times telling the kinds of a folder of files, as the
// open dialog does (four at a time): SNIFF_DIR=folder.
func TestSniffSpeed(t *testing.T) {
	dir := os.Getenv("SNIFF_DIR")
	if dir == "" {
		t.Skip("set SNIFF_DIR to a folder of files")
	}
	entries, _ := os.ReadDir(dir)
	start := time.Now()
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for _, e := range entries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			s := time.Now()
			k := fileKindOf(filepath.Join(dir, e.Name()))
			mu.Lock()
			t.Logf("%6.0fms %-10s %s", float64(time.Since(s).Microseconds())/1000, k, e.Name())
			mu.Unlock()
		}()
	}
	wg.Wait()
	cold := time.Since(start)
	start = time.Now()
	for _, e := range entries {
		fileKindOf(filepath.Join(dir, e.Name()))
	}
	t.Logf("%d files: %v cold, %v cached", len(entries), cold, time.Since(start))
}
