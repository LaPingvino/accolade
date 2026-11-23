package main

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/LaPingvino/lexington/rules"
)

// An .accolade carries the part of Lexington's configuration its script
// uses (Joop, 2026-11-20), so a language or format of one's own works on
// any machine:
//
//   - assets/lexington/lexington.toml: the script's language's scene words
//     and matchup, the Standard and its words, and an element set of
//     lexington.toml that is its format — the language and Standard in
//     full, the element set only if it differs from Lexington's defaults
//     (rules.TOMLConf.Part);
//   - assets/lexington/formats/<key>.toml: its format's file, and the
//     formats it starts from (Base), if they are one's own.
//
// Opening it, that is laid over the machine's configuration for that
// script only (MainWindow.lex): the bundle's wins there, nothing on the
// machine changes. File > Add to My Settings… (lexington_adopt.go) copies
// it, asking on each clash. Only an .accolade carries it; other bundles
// and plain files are as before. Nothing here knows a field: Lexington's
// Part, With, Items and Of do.

const (
	bundleLexDir     = "assets/lexington/"
	bundleLexFile    = bundleLexDir + "lexington.toml"
	bundleFormatsDir = bundleLexDir + "formats/"
)

// carriedConf is what a script brought: Lexington's configuration's part
// and format files by key.
type carriedConf struct {
	lex     rules.TOMLConf
	formats map[string][]byte
}

func (c *carriedConf) empty() bool {
	return c == nil || (len(c.lex.Items()) == 0 && len(c.formats) == 0)
}

// carriesConf tells whether a bundle named name carries its
// configuration: an .accolade.
func carriesConf(name string) bool {
	return strings.EqualFold(path.Ext(name), typeAccolade.ext)
}

// isCarriedEntry tells whether a bundle's file is the configuration's.
func isCarriedEntry(rel string) bool { return strings.HasPrefix(rel, bundleLexDir) }

// readCarried reads the configuration from a bundle's files (all, names
// in the bundle), nil for none. A file that cannot be read is left out:
// the script opens with the machine's configuration for it.
func readCarried(all []string, read func(string) ([]byte, error)) *carriedConf {
	c := &carriedConf{}
	for _, name := range all {
		switch {
		case name == bundleLexFile:
			data, err := read(name)
			if err != nil {
				continue
			}
			if conf, err := rules.Decode(data); err == nil {
				c.lex = conf
			}
		case strings.HasPrefix(name, bundleFormatsDir):
			// a key as a format's file name is made (formatKey): what
			// someone else's bundle names cannot reach elsewhere
			key, ok := strings.CutSuffix(strings.TrimPrefix(name, bundleFormatsDir), ".toml")
			if !ok || key == "" || formatKey(key) != key {
				continue
			}
			data, err := read(name)
			if err != nil {
				continue
			}
			var spec formatSpec
			if _, err := toml.Decode(string(data), &spec); err != nil {
				continue
			}
			if c.formats == nil {
				c.formats = map[string][]byte{}
			}
			c.formats[key] = data
		}
	}
	if c.empty() {
		return nil
	}
	return c
}

// files are the bundle's files for the configuration.
func (c *carriedConf) files() []bundleFile {
	if c.empty() {
		return nil
	}
	var out []bundleFile
	if len(c.lex.Items()) > 0 {
		if data, err := c.lex.Encode(); err == nil {
			out = append(out, bundleFile{bundleLexFile, data})
		}
	}
	for _, k := range slices.Sorted(maps.Keys(c.formats)) {
		out = append(out, bundleFile{bundleFormatsDir + k + ".toml", c.formats[k]})
	}
	return out
}

// layered is the machine's configuration with c over it.
func layered(c *carriedConf) lexConf {
	if c.empty() {
		return machineLex()
	}
	return lexConf{TOMLConf: lexSettings.With(c.lex), carried: c}
}

// lex is the configuration the window's script reads: the machine's, with
// what it brought over it.
func (w *MainWindow) lex() lexConf {
	if w == nil {
		return machineLex()
	}
	return layered(w.carried)
}

// sceneHeaders are the script's language's scene heading words.
func (w *MainWindow) sceneHeaders() []string { return w.lex().headers(w.language) }

// carryFor is what of c a script in lang, laid out with format, with the
// title page template titlePage, carries: its language and the Standard
// in full, an element set of lexington.toml if it differs from
// Lexington's defaults, its format's files if they are one's own (or came
// with it), and its title page template if it is one's own (a preset
// goes by its name alone, in info.json); nil for nothing.
func carryFor(c lexConf, lang, format, titlePage string) *carriedConf {
	langs := []string{c.standard()}
	if lang != "" && lang != c.standard() {
		langs = append([]string{lang}, langs...)
	}
	var sets []string
	if _, preset := rules.GetPreset(format); !preset && format != "" && c.formatFile(format) == nil {
		sets = []string{format} // an element set of lexington.toml
	}
	var titles []string
	if _, own := c.TitlePages[titlePage]; own {
		titles = []string{titlePage}
	}
	part := c.Part(langs, sets, titles...)
	defaults := rules.DefaultConf()
	keep := rules.TOMLConf{}
	for _, it := range part.Items() {
		mine, _ := part.Of(it)
		// A language (and the Standard) goes along in full even when it is
		// Lexington's default (Joop): the script then reads the same on a
		// machine whose own words for that language were changed. Element
		// sets that are the defaults are Lexington's own everywhere.
		if def, ok := defaults.Of(it); ok && it.Kind == "elements" && rules.Same(mine, def) {
			continue
		}
		keep = keep.With(mine)
	}
	out := &carriedConf{lex: keep}
	// the format and what it starts from, as long as they are one's own
	for k := format; k != "" && out.formats[k] == nil; {
		data := c.formatFile(k)
		if data == nil {
			break
		}
		if out.formats == nil {
			out.formats = map[string][]byte{}
		}
		out.formats[k] = data
		var spec formatSpec
		_, _ = toml.Decode(string(data), &spec)
		k = spec.Base
	}
	if out.empty() {
		return nil
	}
	return out
}

// formatFile is a format's file text if it is one's own or came with the
// script (nil for a preset, Accolade's own, or none).
func (c lexConf) formatFile(key string) []byte {
	if c.carried != nil {
		if data, ok := c.carried.formats[key]; ok {
			return data
		}
	}
	if userFormat(key) {
		if data, err := os.ReadFile(filepath.Join(userFormatDir(), key+".toml")); err == nil {
			return data
		}
	}
	return nil
}

// carriedItem is one thing a script brought, as the machine has it.
type carriedItem struct {
	kind  string // "language", "elements", "extension", "titlepage", "standard" or "format"
	key   string // the language, element set or format ("" the standard)
	state itemState
	// theirs is the item as the script brought it: a configuration of
	// it alone, or a format's file
	theirs     rules.TOMLConf
	theirsFile []byte
}

type itemState int

const (
	itemSame  itemState = iota // the machine has it alike
	itemNew                    // the machine has none of that name
	itemClash                  // the machine has another of that name
)

// items are what c brought, each compared with the machine's
// configuration: languages, element sets, the standard, then formats.
func (c *carriedConf) items() []carriedItem {
	if c.empty() {
		return nil
	}
	var out []carriedItem
	machine := lexSettings
	for _, it := range c.lex.Items() {
		theirs, _ := c.lex.Of(it)
		ci := carriedItem{kind: it.Kind, key: it.Key, theirs: theirs, state: itemNew}
		if mine, ok := machine.Of(it); ok {
			ci.state = itemClash
			if rules.Same(mine, theirs) {
				ci.state = itemSame
			}
		}
		if it.Kind == "standard" && machine.StandardLanguage() == c.lex.Standard {
			ci.state = itemSame
		}
		out = append(out, ci)
	}
	for _, k := range slices.Sorted(maps.Keys(c.formats)) {
		ci := carriedItem{kind: "format", key: k, theirsFile: c.formats[k], state: itemNew}
		if userFormat(k) {
			ci.state = itemClash
			if mine, err := os.ReadFile(filepath.Join(userFormatDir(), k+".toml")); err == nil && string(mine) == string(c.formats[k]) {
				ci.state = itemSame
			}
		} else if builtinFormat(k) {
			ci.state = itemClash
		}
		out = append(out, ci)
	}
	return out
}

// toAdd are the items the machine does not have alike.
func (c *carriedConf) toAdd() []carriedItem {
	return slices.DeleteFunc(c.items(), func(it carriedItem) bool { return it.state == itemSame })
}

// label names an item for people: "Esperanto scene words (eo)", "the
// format “Radio Drama” (radio-drama.toml)".
func (it carriedItem) label() string {
	switch it.kind {
	case "language":
		return languageLabel(it.key) + " scene words"
	case "elements":
		return fmt.Sprintf("the format %q (in lexington.toml)", it.key)
	case "standard":
		return "standard Fountain's language: " + languageLabel(it.theirs.Standard)
	case rules.ItemTitlePage:
		return fmt.Sprintf("the title page template “%s”", it.theirs.TitlePageName(it.key))
	case "format":
		var spec formatSpec
		_, _ = toml.Decode(string(it.theirsFile), &spec)
		return fmt.Sprintf("the format “%s” (%s.toml)", cmp.Or(spec.Name, it.key), it.key)
	}
	return it.key
}

// broughtStatus is the status line saying what a script brought that the
// machine has not alike, "" for nothing.
func (c *carriedConf) broughtStatus() string {
	var news, clashes []string
	for _, it := range c.toAdd() {
		if it.state == itemNew {
			news = append(news, it.label())
		} else {
			clashes = append(clashes, it.label())
		}
	}
	var parts []string
	if len(news) > 0 {
		parts = append(parts, "brought "+strings.Join(news, ", "))
	}
	if len(clashes) > 0 {
		parts = append(parts, "has its own "+strings.Join(clashes, ", ")+", not as yours")
	}
	if len(parts) == 0 {
		return ""
	}
	return "This script " + strings.Join(parts, "; ") + ": used for it only. File > Add to My Settings… keeps them"
}
