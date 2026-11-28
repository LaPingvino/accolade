// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/LaPingvino/lexington/rules"
)

// The format folders are decoded again when a file comes, goes or
// changes, and the same files in another folder read as the same; the
// reading is safe from several goroutines (the background reports ask
// for formats too).
func TestPresetsInFollowsTheFiles(t *testing.T) {
	builtin, own := t.TempDir(), t.TempDir()
	writeBuiltinFormats(builtin)
	name := func(list []rules.Preset, key string) string {
		if i := slices.IndexFunc(list, func(p rules.Preset) bool { return p.Key == key }); i >= 0 {
			return list[i].Name
		}
		return ""
	}
	want, _ := rules.PresetsIn(builtin, own)
	if got := presetsIn(builtin, own); !slices.EqualFunc(got, want, func(a, b rules.Preset) bool { return a.Key == b.Key && a.Name == b.Name }) {
		t.Fatalf("presetsIn differs from rules.PresetsIn")
	}
	mine := filepath.Join(own, "mine.toml")
	os.WriteFile(mine, []byte("name = \"Mine\"\nbase = \"radio\"\n"), 0o644)
	if n := name(presetsIn(builtin, own), "mine"); n != "Mine" {
		t.Fatalf("a new file: %q", n)
	}
	os.WriteFile(mine, []byte("name = \"Mine, renamed\"\nbase = \"radio\"\n"), 0o644)
	if n := name(presetsIn(builtin, own), "mine"); n != "Mine, renamed" {
		t.Fatalf("a changed file: %q", n)
	}
	// the same files elsewhere: the same formats
	other := t.TempDir()
	writeBuiltinFormats(other)
	if n := name(presetsIn(other, own), "mine"); n != "Mine, renamed" {
		t.Fatalf("another folder: %q", n)
	}
	os.Remove(mine)
	if n := name(presetsIn(builtin, own), "mine"); n != "" {
		t.Fatalf("a removed file: %q", n)
	}
	// a list given out is the caller's: changing it changes no later one
	got := presetsIn(builtin, own)
	got[0].Name = "changed"
	if presetsIn(builtin, own)[0].Name == "changed" {
		t.Error("a caller's change reached the kept list")
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				if len(presetsIn(builtin, own)) == 0 {
					t.Error("no formats")
				}
			}
		}()
	}
	wg.Wait()
}
