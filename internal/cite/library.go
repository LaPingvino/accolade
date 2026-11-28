// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package cite

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode"
)

// Library is a bibliography as read: its entries in the file's order,
// found by key.
type Library struct {
	Entries []*Entry
	byKey   map[string]*Entry
	// Warning says what was skipped while reading ("" for nothing).
	Warning string
	// Style is the document's csl: style file, read with the library
	// for the document's csl: StyleFor (nil: none found there; a name of
	// Accolade's own styles needs no file, BuiltinStyle).
	Style     *Style
	StyleFor  string
	StyleNote string // why the csl: style is not used ("": it is)
}

// StyleOf is the style a document's csl: names, with lib its library:
// the file read with it, else one of Accolade's (apa, ieee …, by name
// or file name), else nil (Accolade's own author-date style).
func StyleOf(csl string, lib *Library) *Style {
	if csl == "" {
		return nil
	}
	if lib != nil && lib.Style != nil && lib.StyleFor == csl {
		return lib.Style
	}
	return BuiltinStyle(csl)
}

// NewLibrary makes a library of entries (the first of a key wins).
func NewLibrary(es []*Entry) *Library {
	l := &Library{byKey: make(map[string]*Entry, len(es))}
	for _, e := range es {
		if _, dup := l.byKey[e.Key]; dup || e.Key == "" {
			continue
		}
		l.byKey[e.Key] = e
		l.Entries = append(l.Entries, e)
	}
	return l
}

// Get is the entry of key, nil if the library has none (Pandoc's keys
// are case-sensitive; BibTeX's are not, so a key in another case is
// taken when there is no exact one).
func (l *Library) Get(key string) *Entry {
	if l == nil {
		return nil
	}
	if e, ok := l.byKey[key]; ok {
		return e
	}
	for k, e := range l.byKey {
		if strings.EqualFold(k, key) {
			return e
		}
	}
	return nil
}

// Format is how a library file is written.
type Format int

const (
	UnknownFormat Format = iota
	CSLJSON
	BibTeX
)

// FormatOf is the format a file's name says: .json for CSL-JSON, .bib
// (.bibtex, .biblatex) for BibTeX.
func FormatOf(name string) Format {
	switch strings.ToLower(path.Ext(strings.ReplaceAll(name, "\\", "/"))) {
	case ".json":
		return CSLJSON
	case ".bib", ".bibtex", ".biblatex":
		return BibTeX
	}
	return UnknownFormat
}

// Parse reads a library file named name.
func Parse(name string, data []byte) (*Library, error) {
	var es []*Entry
	var err error
	switch FormatOf(name) {
	case CSLJSON:
		es, err = ParseCSLJSON(data)
	case BibTeX:
		es, err = ParseBibTeX(data)
		if err != nil && len(es) > 0 { // what was skipped is said, the rest kept
			l := NewLibrary(es)
			l.Warning = err.Error()
			return l, nil
		}
	default:
		return nil, fmt.Errorf("%s: not a library Accolade reads (CSL-JSON .json, BibTeX .bib)", path.Base(name))
	}
	if err != nil {
		return nil, err
	}
	return NewLibrary(es), nil
}

// Match is the entries for what was typed after @: keys starting with
// it first, then authors' names, then keys and titles holding it; all
// of them (in the library's order of keys) for nothing typed. Case and
// accents do not count.
func (l *Library) Match(typed string, limit int) []*Entry {
	if l == nil {
		return nil
	}
	q := fold(typed)
	type scored struct {
		e     *Entry
		score int
	}
	var hits []scored
	for _, e := range l.Entries {
		s := matchScore(e, q)
		if s > 0 {
			hits = append(hits, scored{e, s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return strings.ToLower(hits[i].e.Key) < strings.ToLower(hits[j].e.Key)
	})
	out := make([]*Entry, 0, min(len(hits), max(limit, 0)))
	for _, h := range hits {
		if limit > 0 && len(out) == limit {
			break
		}
		out = append(out, h.e)
	}
	return out
}

func matchScore(e *Entry, q string) int {
	if q == "" {
		return 1
	}
	key := fold(e.Key)
	switch {
	case key == q:
		return 100
	case strings.HasPrefix(key, q):
		return 90
	}
	for _, n := range e.Creators() {
		if strings.HasPrefix(fold(n.Family), q) || strings.HasPrefix(fold(n.Literal), q) {
			return 70
		}
	}
	if strings.Contains(key, q) {
		return 50
	}
	t := fold(e.Title)
	for _, w := range strings.Fields(t) {
		if strings.HasPrefix(w, q) {
			return 40
		}
	}
	if len(q) >= 3 && strings.Contains(t, q) {
		return 30
	}
	if e.Year != "" && strings.HasPrefix(e.Year, q) {
		return 20
	}
	return 0
}

// fold is s in lower case without its accents (ö as o, ß as ss).
func fold(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'ß':
			b.WriteString("ss")
			continue
		case 'ø':
			r = 'o'
		case 'æ':
			b.WriteString("ae")
			continue
		case 'ł':
			r = 'l'
		}
		if f, ok := baseLetter[r]; ok {
			r = f
		}
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// baseLetter is the Latin letters with an accent as the letter alone.
var baseLetter = func() map[rune]rune {
	m := map[rune]rune{}
	for base, accented := range map[rune]string{
		'a': "àáâãäåāăą", 'c': "çćĉċč", 'd': "ďđ", 'e': "èéêëēĕėęě", 'g': "ĝğġģ", 'h': "ĥħ",
		'i': "ìíîïĩīĭįı", 'j': "ĵ", 'k': "ķ", 'l': "ĺļľŀ", 'n': "ñńņňŉ", 'o': "òóôõöōŏőơ",
		'r': "ŕŗř", 's': "śŝşšș", 't': "ţťŧț", 'u': "ùúûüũūŭůűųư", 'w': "ŵ", 'y': "ýÿŷ", 'z': "źżž",
	} {
		for _, r := range accented {
			m[r] = base
		}
	}
	return m
}()
