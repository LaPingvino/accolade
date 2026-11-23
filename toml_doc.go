package main

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/rules"
)

// A .toml file Accolade opens (File > Open, the command line, the
// desktop's Open With, Preferences' buttons) is edited in a form picked
// by what the file is (LOOP.md, Joop 2026-11-22: "settings editor when
// opening any .toml, so including custom formats"): Lexington's settings
// (lexington.toml) in the settings editor (settings_editor.go), a format
// file in the format editor (format_editor.go), anything else as text
// with a note. The text is the only state: every control of a form is one
// edit of the text (toml_text.go), checked here before it is kept; the
// form is drawn from the text again; the TOML tab shows the same text.

// tomlKind is what a .toml file is.
type tomlKind int

const (
	tomlPlain    tomlKind = iota // some other TOML: text
	tomlSettings                 // Lexington's settings (lexington.toml)
	tomlFormat                   // a format file (a preset's file)
)

// classifyTOML is what the file at path, with text, is: its name and
// folder first (lexington.toml; a file in a formats folder), then what
// it holds, by a key only that kind has and only if it reads as that kind
// (a Name or a list of Extensions alone is in many files). Text that is
// not TOML is the kind its name says (plain for a name that says
// nothing), to be mended in the TOML tab.
func classifyTOML(path, text string) tomlKind {
	if strings.EqualFold(filepath.Base(path), "lexington.toml") {
		return tomlSettings
	}
	if inFormatFolder(path) {
		return tomlFormat
	}
	m, err := tomlMap(text)
	if err != nil {
		return tomlPlain
	}
	_, langs := m["Languages"].(map[string]any)
	_, std := m["Standard"].(string)
	_, base := m["Base"].(string)
	_, accolade := m["Accolade"].(map[string]any)
	sets, elements := elementsShape(m)
	reads := func(k tomlKind) bool { // as that kind's types (what is wrong in it is mended in its editor)
		if k == tomlFormat {
			var f formatSpec
			_, err := toml.Decode(text, &f)
			return err == nil
		}
		_, err := rules.Decode([]byte(text))
		return err == nil
	}
	switch {
	case (base || accolade || elements) && reads(tomlFormat):
		return tomlFormat
	case (langs || std || sets) && reads(tomlSettings):
		return tomlSettings
	}
	return tomlPlain
}

// elementsShape tells what the text's [Elements] are: sets of elements
// (lexington.toml's), or elements of fields (a format file's); neither
// for anything else.
func elementsShape(m map[string]any) (sets, elements bool) {
	els, ok := m["Elements"].(map[string]any)
	if !ok || len(els) == 0 {
		return false, false
	}
	sets, elements = true, true
	for k, v := range els {
		t, ok := v.(map[string]any)
		if !ok {
			return false, false
		}
		if !rules.ConfigKey(k).IsValid() {
			elements = false
		}
		for f, x := range t {
			if _, isTable := x.(map[string]any); isTable {
				elements = false
			} else {
				sets = false
				if _, known := rules.Describe(rules.FormatFields(), f); !known {
					elements = false
				}
			}
		}
	}
	return sets, elements
}

// inFormatFolder reports whether path is a file in one of the format
// folders (one's own, Accolade's).
func inFormatFolder(path string) bool {
	dir := absPath(filepath.Dir(path))
	return dir == absPath(userFormatDir()) || dir == absPath(builtinFormatDir())
}

// tomlDoc is a .toml file being edited: where it is, what it is, its text
// now and as it was last read or saved.
type tomlDoc struct {
	path  string
	kind  tomlKind
	text  string
	saved string
	// key is a format's (its file's name without .toml)
	key string
}

// newTOMLDoc is the file at path with text, as it is.
func newTOMLDoc(path, text string) *tomlDoc {
	d := &tomlDoc{path: path, text: text, saved: text, kind: classifyTOML(path, text)}
	d.key = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return d
}

// readTOMLDoc reads the file at path.
func readTOMLDoc(path string) (*tomlDoc, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return newTOMLDoc(path, string(b)), nil
}

// dirty reports whether the text has changes not saved.
func (d *tomlDoc) dirty() bool { return d.text != d.saved }

// conf is the text read as Lexington's settings (zero if it cannot be).
func (d *tomlDoc) conf() rules.TOMLConf {
	c, _ := rules.Decode([]byte(d.text))
	return c
}

// spec is the text read as a format file.
func (d *tomlDoc) spec() formatSpec {
	var s formatSpec
	_, _ = toml.Decode(d.text, &s)
	return s
}

// raw is the text as TOML, nil if it is not.
func (d *tomlDoc) raw() map[string]any {
	m, err := tomlMap(d.text)
	if err != nil {
		return nil
	}
	return m
}

// has reports whether the text sets path (a key or a table).
func (d *tomlDoc) has(path ...string) bool {
	_, ok := mapAt(d.raw(), path)
	return ok
}

// check is what is wrong with text as the kind of file the document is,
// nil for nothing.
func (d *tomlDoc) check(text string) error {
	if _, err := tomlMap(text); err != nil {
		return err
	}
	switch d.kind {
	case tomlSettings:
		c, err := rules.Decode([]byte(text))
		if err != nil {
			return err
		}
		for name, set := range c.Elements {
			for k, f := range set {
				if err := checkFormat(k, f); err != nil {
					return fmt.Errorf("set %s, %s: %w", name, k, err)
				}
			}
		}
		for name, l := range c.Languages {
			if err := checkLanguage(l); err != nil {
				return fmt.Errorf("language %s: %w", name, err)
			}
		}
	case tomlFormat:
		var s formatSpec
		if _, err := toml.Decode(text, &s); err != nil {
			return err
		}
		for k, f := range s.Elements {
			if err := checkFormat(k, f); err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
		}
		if s.Base != "" && !d.knownBase(s.Base) {
			return fmt.Errorf("no format %q to start from (Base)", s.Base)
		}
	}
	return nil
}

// notes are what the text has that Lexington does not read (elements
// it has no rule for, as old files have "note"): kept as written, shown
// under the form.
func (d *tomlDoc) notes() []string {
	var out []string
	unknown := func(where string, set rules.Set) {
		for _, k := range slices.Sorted(maps.Keys(set)) {
			if !rules.ConfigKey(k).IsValid() {
				out = append(out, fmt.Sprintf("%s%q is no element Lexington knows: it is kept, and not read.", where, k))
			}
		}
	}
	switch d.kind {
	case tomlSettings:
		c := d.conf()
		for _, n := range slices.Sorted(maps.Keys(c.Elements)) {
			unknown("Set "+n+": ", c.Elements[n])
		}
		for _, lang := range slices.Sorted(maps.Keys(c.Languages)) {
			// a new scene written by the language's form must read as one
			l := c.Language(lang)
			if h, ok := exampleOf(lang, l, l.HeadingForm()); !ok {
				out = append(out, fmt.Sprintf("%s: a new scene written by %q (%s) does not read as a scene heading: make it one of the patterns, or start it with a scene word.", lang, l.HeadingForm(), h))
			}
		}
		for _, t := range []string{"Scenes", "Matchups", "Directions"} {
			if d.has(t) {
				out = append(out, "["+t+"] is the older way of writing languages: Lexington reads it over [Languages]. A language changed here moves out of it.")
				break
			}
		}
	case tomlFormat:
		unknown("", d.spec().Elements)
	}
	return out
}

// knownBase reports whether a format file can start from base: a
// format the machine has, or Lexington's preset of its own name.
func (d *tomlDoc) knownBase(base string) bool {
	if _, ok := rules.GetPreset(base); ok {
		return true
	}
	if base == d.key && !builtinFormat(base) {
		return false // a format of one's own cannot start from itself
	}
	return slices.ContainsFunc(scriptFormats(), func(p rules.Preset) bool { return p.Key == base })
}

// checkFormat is what is wrong with an element's format.
func checkFormat(key string, f rules.Format) error {
	switch {
	case f.Left < 0 || f.Right < 0:
		return errors.New("a margin cannot be negative")
	case f.Left > 8 || f.Right > 8:
		return errors.New("a margin is wider than the paper")
	case f.Size < 0 || f.Size > 72:
		return errors.New("the size is from 1 to 72 points (0 for 12)")
	case !slices.Contains([]string{"", "L", "C", "R"}, f.Align):
		return fmt.Errorf("align %q: L, C or R", f.Align)
	}
	switch key {
	case rules.KeyPagination.String():
		if _, err := rules.ParsePagination(f.Style); err != nil {
			return err
		}
	case rules.KeyNoteColors.String():
		if _, err := lex.ParseColorScheme(f.Style); err != nil {
			return err
		}
	default:
		if strings.Trim(strings.ToLower(f.Style), "biu") != "" {
			return fmt.Errorf("style %q: b, i and u", f.Style)
		}
	}
	return nil
}

// checkLanguage is what is wrong with a language.
func checkLanguage(l rules.Language) error {
	if d := strings.ToLower(strings.TrimSpace(l.Direction)); d != "" && d != "ltr" && d != "rtl" {
		return fmt.Errorf("direction %q: ltr or rtl", l.Direction)
	}
	for _, c := range l.Cues {
		if !slices.Contains(rules.CueRules, c) {
			return fmt.Errorf("no cue rule %q", c)
		}
	}
	if l.Repeat < 0 || l.Shape.Words < 0 || l.Shape.Chars < 0 {
		return errors.New("a count cannot be negative")
	}
	return nil
}

// tomlOp is one edit: path set to value, or taken out (value nil).
type tomlOp struct {
	path  []string
	value any
}

// set is the edit setting path to v.
func set(v any, path ...string) tomlOp { return tomlOp{path: path, value: v} }

// drop is the edit taking path out.
func drop(path ...string) tomlOp { return tomlOp{path: path} }

// edit makes the edits in the text, in order, if the result reads back
// as exactly that: the same as before but for those paths, and a file of
// the document's kind without errors. Otherwise nothing changes and the
// error says why.
func (d *tomlDoc) edit(ops ...tomlOp) error {
	before, err := tomlMap(d.text)
	if err != nil {
		return fmt.Errorf("the TOML has an error; mend it in the TOML tab first: %w", err)
	}
	text, want := d.text, before
	for _, op := range ops {
		if op.value == nil {
			if _, ok := mapAt(want, op.path); !ok {
				continue
			}
			text, err = dropPath(text, op.path)
		} else {
			text, err = setPath(text, op.path, op.value)
		}
		if err != nil {
			return err
		}
		want = withAt(want, op.path, op.value)
	}
	return d.keep(text, want)
}

// ensureTable makes sure the text has the table at path, empty if it
// had none.
func (d *tomlDoc) ensureTable(path ...string) error {
	if d.has(path...) {
		return nil
	}
	before, err := tomlMap(d.text)
	if err != nil {
		return err
	}
	return d.keep(appendTables(d.text, "["+joinTOMLKey(path)+"]\n"), withAt(before, path, map[string]any{}))
}

// addTable adds the table at path with the keys of m (in order's order,
// then the rest), as a [table] of its own at the end; a table the text
// has already gets the keys set in it.
func (d *tomlDoc) addTable(path []string, m map[string]any, order []string) error {
	if d.has(path...) {
		var ops []tomlOp
		for _, k := range keysInOrder(m, order) {
			ops = append(ops, set(m[k], append(slices.Clone(path), k)...))
		}
		return d.edit(ops...)
	}
	before, err := tomlMap(d.text)
	if err != nil {
		return err
	}
	block := "[" + joinTOMLKey(path) + "]\n"
	for _, k := range keysInOrder(m, order) {
		block += tomlKeyText(k) + " = " + tomlInline(m[k]) + "\n"
	}
	return d.keep(appendTables(d.text, block), withAt(before, path, m))
}

// editText makes a change to the text that is not a path's (a [[block]]
// moved), if the result reads back as want.
func (d *tomlDoc) editText(change func(string) (string, error), want func(map[string]any) map[string]any) error {
	before, err := tomlMap(d.text)
	if err != nil {
		return fmt.Errorf("the TOML has an error; mend it in the TOML tab first: %w", err)
	}
	text, err := change(d.text)
	if err != nil {
		return err
	}
	return d.keep(text, want(before))
}

// keep takes text if it reads back as want and is right for its kind.
func (d *tomlDoc) keep(text string, want map[string]any) error {
	got, err := tomlMap(text)
	if err != nil {
		return fmt.Errorf("the edit would break the TOML (%v): nothing changed", err)
	}
	if !sameTOML(withoutEmpty(got, "Accolade"), withoutEmpty(want, "Accolade")) {
		return errors.New("the file is written in a way this edit cannot change safely: nothing changed; change it in the TOML tab")
	}
	if err := d.check(text); err != nil {
		return err
	}
	d.text = text
	return nil
}

// withoutEmpty is m without its key if that is an empty table (an
// [Accolade] header left with no buttons under it is none).
func withoutEmpty(m map[string]any, key string) map[string]any {
	if t, ok := m[key].(map[string]any); ok && len(t) == 0 {
		return withAt(m, []string{key}, nil)
	}
	return m
}

// sameTOML reports whether two TOML values are the same, their numbers
// and lists as TOML reads them.
func sameTOML(a, b map[string]any) bool {
	return reflect.DeepEqual(normalTOML(a), normalTOML(b))
}

// normalTOML is m written and read again, so a Go value and the same
// value read from TOML compare equal.
func normalTOML(m map[string]any) map[string]any {
	var b bytes.Buffer
	if err := toml.NewEncoder(&b).Encode(m); err != nil {
		return m
	}
	out, err := tomlMap(b.String())
	if err != nil {
		return m
	}
	return out
}
