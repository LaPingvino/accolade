package cite

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Front is what a document's front matter says about its citations.
type Front struct {
	Bibliography string   // the first bibliography: (Zettlr reads only the first)
	CSL          string   // csl: the style Pandoc's export uses
	Nocite       []string // nocite: the keys listed without being cited ("*": all)
}

// FrontOf reads the citation fields of text's YAML front matter (the
// --- block it may start with): bibliography as a value, a [flow, list]
// or a block list, csl, and nocite's @keys.
func FrontOf(text string) Front {
	var f Front
	first, rest, _ := strings.Cut(text, "\n")
	if strings.TrimRight(first, " \t\r") != "---" {
		return f
	}
	// the block's lines, read up to its end only (this runs as keys are
	// typed: the document's text after it is not split)
	var lines []string
	closed := false
	for rest != "" {
		var l string
		l, rest, _ = strings.Cut(rest, "\n")
		if t := strings.TrimRight(l, " \t\r"); t == "---" || t == "..." {
			closed = true
			break
		}
		lines = append(lines, l)
	}
	if !closed {
		return f
	}
	// block is the indented lines after line i (a block list, a | text)
	block := func(i int) []string {
		var out []string
		for j := i + 1; j < len(lines); j++ {
			l := strings.TrimRight(lines[j], " \t\r")
			if l != "" && l[0] != ' ' && l[0] != '\t' && l[0] != '-' {
				break
			}
			out = append(out, strings.TrimSpace(l))
		}
		return out
	}
	for i, l := range lines {
		if l == "" || l[0] == ' ' || l[0] == '\t' {
			continue
		}
		k, v, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "bibliography":
			switch {
			case strings.HasPrefix(v, "["):
				first, _, _ := strings.Cut(strings.Trim(v, "[]"), ",")
				f.Bibliography = unquote(first)
			case v != "":
				f.Bibliography = unquote(v)
			default:
				for _, b := range block(i) {
					if strings.HasPrefix(b, "-") {
						f.Bibliography = unquote(strings.TrimSpace(b[1:]))
						break
					}
				}
			}
		case "csl":
			f.CSL = unquote(v)
		case "nocite":
			words := v
			if v == "" || v == "|" || v == ">" || strings.HasPrefix(v, "|") || strings.HasPrefix(v, ">") {
				words = strings.Join(block(i), " ")
			}
			for _, w := range strings.FieldsFunc(unquote(words), func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '[' || r == ']' }) {
				if strings.HasPrefix(w, "@") && len(w) > 1 {
					f.Nocite = append(f.Nocite, strings.Trim(w[1:], "{}"))
				}
			}
		}
	}
	return f
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}

// Place is where a document is, for finding its library.
type Place struct {
	Front    string   // the front matter's bibliography
	Dir      string   // the document's folder on disk ("" for none: not saved, the browser); a bundle's: the folder it is in
	Stem     string   // the document's (or bundle's) name without its extension
	DirFiles []string // the names of the files in Dir
	Bundle   []string // a bundle's entries ("assets/refs.bib"); nil when it is no bundle
	Main     string   // Preferences' main library ("" none)
	Home     string   // the home folder, for ~/ in a path
}

// Source is a library file: an entry of the document's bundle or a file
// on disk, and why it is the one.
type Source struct {
	Bundle bool   // Name is the bundle's entry
	Name   string // the entry, or the file's path
	Why    string
}

// The reasons a library is the document's.
const (
	WhyFront  = "the front matter's bibliography"
	WhyBundle = "in the bundle"
	WhyBeside = "beside the document"
	WhyMain   = "the main library (Preferences)"
)

func (s Source) String() string {
	if s.Name == "" {
		return "no library"
	}
	return fmt.Sprintf("%s (%s)", s.Base(), s.Why)
}

// Base is the file's name.
func (s Source) Base() string {
	if s.Bundle {
		return path.Base(s.Name)
	}
	return filepath.Base(s.Name)
}

// expand gives a path from the front matter or Preferences: ~/ from the
// home folder, a relative one from dir.
func (p Place) expand(name, dir string) string {
	switch {
	case strings.HasPrefix(name, "~/") && p.Home != "":
		return filepath.Join(p.Home, name[2:])
	case filepath.IsAbs(name):
		return name
	case dir != "":
		return filepath.Join(dir, filepath.FromSlash(name))
	}
	return ""
}

// candidates are the files the library may be, in order, in groups: the
// first group that has one that reads is the library. A group that is
// named (front matter, Preferences) is the library even when it cannot
// be read (it says why); a guessed one (beside, in the bundle) only when
// it reads as a library.
func (p Place) candidates() (groups [][]Source, named []bool) {
	if p.Front != "" {
		var g []Source
		if p.Bundle != nil && !filepath.IsAbs(p.Front) && !strings.HasPrefix(p.Front, "~/") {
			clean := path.Clean(strings.TrimPrefix(filepath.ToSlash(p.Front), "./"))
			for _, n := range []string{clean, "assets/" + clean} {
				if has(p.Bundle, n) {
					g = append(g, Source{Bundle: true, Name: n, Why: WhyFront})
				}
			}
		}
		if f := p.expand(p.Front, p.Dir); f != "" {
			g = append(g, Source{Name: f, Why: WhyFront})
		}
		if len(g) == 0 { // not saved yet: it says so
			g = append(g, Source{Name: p.Front, Why: WhyFront})
		}
		return [][]Source{g}, []bool{true}
	}
	if p.Bundle != nil {
		var g []Source
		for _, n := range libraryFiles(p.Bundle, "assets/", "") {
			g = append(g, Source{Bundle: true, Name: n, Why: WhyBundle})
		}
		groups, named = append(groups, g), append(named, false)
	}
	if p.Dir != "" {
		var g []Source
		for _, n := range libraryFiles(p.DirFiles, "", p.Stem) {
			g = append(g, Source{Name: filepath.Join(p.Dir, n), Why: WhyBeside})
		}
		groups, named = append(groups, g), append(named, false)
	}
	if p.Main != "" {
		if f := p.expand(p.Main, ""); f != "" {
			groups, named = append(groups, []Source{{Name: f, Why: WhyMain}}), append(named, true)
		}
	}
	return groups, named
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// libraryFiles are the names in names (under dir) a library may be, in
// the order they are tried: the document's own name's (stem.json, then
// stem.bib), then the other .bib files, then the other .json files (most
// JSON files are no library: those are tried last, and only taken if
// they read as CSL-JSON); each sorted.
func libraryFiles(names []string, dir, stem string) []string {
	var own, bibs, jsons []string
	for _, n := range names {
		if !strings.HasPrefix(n, dir) || strings.Contains(n[len(dir):], "/") {
			continue
		}
		base := n[len(dir):]
		f := FormatOf(base)
		if f == UnknownFormat || strings.HasPrefix(base, ".") {
			continue
		}
		switch {
		case stem != "" && strings.EqualFold(strings.TrimSuffix(base, path.Ext(base)), stem):
			own = append(own, n)
		case f == BibTeX:
			bibs = append(bibs, n)
		default:
			jsons = append(jsons, n)
		}
	}
	sort.Slice(own, func(i, j int) bool { // .json before .bib
		fi, fj := FormatOf(own[i]), FormatOf(own[j])
		if fi != fj {
			return fi < fj
		}
		return own[i] < own[j]
	})
	sort.Strings(bibs)
	sort.Strings(jsons)
	return append(append(own, bibs...), jsons...)
}

// Resolve finds the document's library and reads it: the front matter's
// bibliography (its path from the document's folder, or in its bundle),
// else a library in the bundle's assets/, else one beside the document,
// else Preferences' main library. read gives a source's bytes. With no
// library at all, the Source is empty and the error nil; a named library
// that cannot be read is returned with the error.
func Resolve(p Place, read func(Source) ([]byte, error)) (Source, *Library, error) {
	groups, named := p.candidates()
	for gi, g := range groups {
		var firstErr error
		for _, s := range g {
			data, err := read(s)
			if err == nil {
				var lib *Library
				lib, err = Parse(s.Name, data)
				if err == nil && (named[gi] || len(lib.Entries) > 0) {
					return s, lib, nil
				}
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %v", s.Base(), err)
			}
		}
		if named[gi] && len(g) > 0 {
			return g[0], nil, firstErr
		}
	}
	return Source{}, nil, nil
}
