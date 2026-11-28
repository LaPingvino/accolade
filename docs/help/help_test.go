// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package help

import (
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every page and cheat sheet is embedded (or made) and has a title.
func TestPagesFound(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range All() {
		if seen[p.Slug] {
			t.Errorf("%s twice", p.Slug)
		}
		seen[p.Slug] = true
		if p.Title == p.Slug || strings.TrimSpace(p.Markdown) == "" {
			t.Errorf("%s: title %q, %d bytes", p.Slug, p.Title, len(p.Markdown))
		}
		if p.Cheat != strings.HasPrefix(p.Slug, "cheat-") {
			t.Errorf("%s: cheat %v", p.Slug, p.Cheat)
		}
	}
	// every .md here is in the catalogue
	files, _ := filepath.Glob("*.md")
	for _, f := range files {
		if !seen[strings.TrimSuffix(f, ".md")] {
			t.Errorf("%s is not in the catalogue", f)
		}
	}
	if CheatsFor(true)[0].Slug != "cheat-markdown" || CheatsFor(false)[0].Slug != "cheat-shortcuts" {
		t.Errorf("mode order: %s, %s", CheatsFor(true)[0].Slug, CheatsFor(false)[0].Slug)
	}
}

// Every link goes to a page here and a heading on it, or is a web address.
func TestLinksResolve(t *testing.T) {
	anchors := map[string]map[string]bool{}
	for _, p := range All() {
		anchors[p.Slug] = map[string]bool{}
		for _, s := range Sections(p.Markdown) {
			anchors[p.Slug][s.Anchor] = true
		}
	}
	for _, p := range All() {
		for _, l := range Links(p.Markdown) {
			tg := Resolve(p.Slug, l)
			if tg.URL != "" {
				if !strings.HasPrefix(tg.URL, "https://") {
					t.Errorf("%s: link %q", p.Slug, l)
				}
				continue
			}
			a, ok := anchors[tg.Slug]
			if !ok {
				t.Errorf("%s: link %q to no page", p.Slug, l)
				continue
			}
			if tg.Anchor != "" && !a[tg.Anchor] {
				t.Errorf("%s: link %q to no heading on %s", p.Slug, l, tg.Slug)
			}
		}
	}
}

func TestAnchor(t *testing.T) {
	for in, want := range map[string]string{
		"Scene numbers & OMITTED": "scene-numbers--omitted",
		"The `/* Notes */` block": "the--notes--block",
		"Tab and Enter":           "tab-and-enter",
		"Àccents ok":              "àccents-ok",
	} {
		if got := Anchor(plainText(in)); got != want {
			t.Errorf("Anchor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSearch(t *testing.T) {
	hits := Search(All(), "parenthetical")
	if len(hits) == 0 {
		t.Fatal("no hits")
	}
	if Search(All(), "zzqx-nothing") != nil {
		t.Error("hits for nothing")
	}
	for _, h := range Search(All(), "LOCK scene") {
		low := strings.ToLower(h.Section.Markdown)
		if !strings.Contains(low, "lock") || !strings.Contains(low, "scene") {
			t.Errorf("hit without the words: %s › %s", h.Page.Slug, h.Section.Heading)
		}
	}
}

var hrefRE = regexp.MustCompile(`href="([^"]+)"`)
var idRE = regexp.MustCompile(`id="([^"]+)"`)

// The web pages: one per page, links between them resolve to a file and
// an id on it, the version is on each, and the app's links match.
func TestSite(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSite(dir, "v9.9.9"); err != nil {
		t.Fatal(err)
	}
	ids := map[string]map[string]bool{}
	pages := map[string]string{}
	for _, p := range All() {
		b, err := os.ReadFile(filepath.Join(dir, p.Slug+".html"))
		if err != nil {
			t.Fatal(err)
		}
		pages[p.Slug+".html"] = string(b)
		ids[p.Slug+".html"] = map[string]bool{}
		for _, m := range idRE.FindAllStringSubmatch(string(b), -1) {
			ids[p.Slug+".html"][m[1]] = true
		}
		if !strings.Contains(string(b), `data-version="v9.9.9"`) || !strings.Contains(string(b), "<title>"+html.EscapeString(p.Title)) {
			t.Errorf("%s: no version or title", p.Slug)
		}
	}
	for name, b := range pages {
		for _, m := range hrefRE.FindAllStringSubmatch(b, -1) {
			h := m[1]
			if strings.HasPrefix(h, "https://") || h == "../" {
				continue
			}
			file, anchor, _ := strings.Cut(h, "#")
			if file == "" {
				file = name
			}
			if _, ok := pages[file]; !ok {
				t.Errorf("%s: link %q to no page", name, h)
			} else if anchor != "" && !ids[file][anchor] {
				t.Errorf("%s: link %q to no id", name, h)
			}
		}
	}
	if got := URL("fountain", "dev abc (2026)", "scene-headings"); got != Site+"fountain.html?v=dev+abc+%282026%29#scene-headings" {
		t.Errorf("URL = %q", got)
	}
}

// A cheat sheet prints as a card: no navigation, columns.
func TestCard(t *testing.T) {
	b, err := HTML(must("cheat-fountain"), "v1", true)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `class="card"`) || strings.Contains(s, "<nav>") {
		t.Errorf("not a card")
	}
	b, _ = HTML(must("cheat-breakdown"), "v1", true)
	if !strings.Contains(string(b), `class="swatch" style="background:#0000FF"`) {
		t.Errorf("no colour swatches")
	}
}

func TestFeedbackURL(t *testing.T) {
	u := FeedbackURL("v1.2", "linux", "screenplay")
	if !strings.HasPrefix(u, Issues+"?") || !strings.Contains(u, "v1.2") || !strings.Contains(u, "linux") {
		t.Errorf("%s", u)
	}
}
