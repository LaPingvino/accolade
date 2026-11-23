package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/fountain"

	lexreport "github.com/LaPingvino/lexington/report"
)

// The character list (LOOP.md, Joop 2026-11-20): per character a note
// space, kept in the script as a boneyard block at its end like the
// others, so it travels with the file and other Fountain apps ignore it:
//
//	/* Characters
//	ANNA: the lead, thirty-four
//	  age: 34
//	  look: a red coat, always
//	  arc: from fear to trust
//	  actor: Jane Doe
//	  photo: assets/anna.jpg
//	  She keeps the letters she never sends.
//	ben
//	*/
//
// A line that starts at the margin is a character: the name, then after
// ":" the first line of their note. The indented lines under it are
// theirs: a few fields by key (age; look, also description or
// appearance; arc; actor, also casting or played by; photo, the cast
// photo's file, kept in the bundle's assets/ or a folder beside the
// script: pictures.go), and the rest the
// note. A name alone is a character listed with nothing to say yet.
//
// The block is the script's character list. The Characters pane lists
// the names in it with those who speak, shows and edits the notes, and
// writes an entry only where it changes, the others kept as written.
// Names keep the case and the script they are written in, and are matched
// to cues by their words in capitals (charKey), so a no-caps list ("ben",
// 小明) is a list too: the language rules' `list` cue rule (a cue is a
// name in this list) reads listedNames, and the editor's "add the name to
// the list" is addedName's edit (LOOP.md, the language-rules item).

var charactersMarks = endMarks(fountain.EndCharacters)

// A Markdown document keeps its character list the same way, in an HTML
// comment before its notes (which no Markdown app shows):
//
//	<!-- Characters
//	Anna: the one who stays
//	  age: 34
//	-->
//
//	<!-- Notes
//	…
//	-->
var markdownCharsMarks = newNotesMarks("<!-- Characters", "-->", "-- >")

// charBlock is which kind of document's character list: a screenplay's
// boneyard block, or a Markdown document's comment (true).
type charBlock bool

const (
	screenplayList charBlock = false
	markdownList   charBlock = true
)

// charBlock is the window's document's.
func (w *MainWindow) charBlock() charBlock { return charBlock(w.markdown) }

// inside is what is inside the document's character list.
func (k charBlock) inside(text string) (string, bool) {
	t := k.tail(text)
	return t.inside, t.ok
}

// tail is the document's character list block, as found.
func (k charBlock) tail(text string) tailBlock {
	if k == markdownList {
		return mdCharsTail(text)
	}
	return tailSetOf(text).chars
}

// with is the edit that makes the document's character list hold inside
// ("" takes it out): where (runes) and what. A new block goes after the
// text, before the other blocks at its end.
func (k charBlock) with(text, inside string) (start, end int, block string) {
	if k == markdownList {
		t := mdCharsTail(text)
		if !t.ok { // before the notes block, or at the end
			t.start = utf8.RuneCountInString(text[:markdownNotes.startByte(text)])
		}
		return withTailBlockAt(text, t, markdownCharsMarks, inside, t.start)
	}
	return withTailBlock(text, tailSetOf(text).chars, charactersMarks, inside)
}

// list is the document's character list.
func (k charBlock) list(text string) charList {
	inside, _ := k.inside(text)
	return parseCharList(inside)
}

// mdCharsTail is a Markdown document's character list: the comment just
// before its notes block (or at its end), in bytes and in runes.
func mdCharsTail(text string) tailBlock {
	end := markdownNotes.startByte(text)
	i, inside := markdownCharsMarks.at(text[:end])
	if i < 0 {
		return tailBlock{}
	}
	start := utf8.RuneCountInString(text[:i])
	return tailBlock{bstart: i, bend: end, start: start, end: start + utf8.RuneCountInString(text[i:end]), ok: true,
		inside: strings.ReplaceAll(inside, markdownCharsMarks.escaped, markdownCharsMarks.close)}
}

// markdownEndByte is where a Markdown document's own text ends: before
// its character list and its notes.
func markdownEndByte(text string) int {
	if t := mdCharsTail(text); t.ok {
		return t.bstart
	}
	return markdownNotes.startByte(text)
}

// charactersBlock is what is inside the script's character list.
func charactersBlock(text string) (string, bool) { return screenplayList.inside(text) }

// withCharactersBlock is the edit that makes the script's character list
// hold inside ("" takes it out): where (runes) and what.
func withCharactersBlock(text, inside string) (start, end int, block string) {
	return screenplayList.with(text, inside)
}

// The fields a character's entry has besides the note, in the order an
// entry is written.
const (
	fieldAge = iota
	fieldLook
	fieldArc
	fieldActor
	fieldPhoto // the cast photo: a picture in the bundle's assets/ (or beside the script)
	fieldCount
)

// fieldKeys are the fields' keys as written, and fieldLabels as shown.
var (
	fieldKeys   = [fieldCount]string{"age", "look", "arc", "actor", "photo"}
	fieldLabels = [fieldCount]string{"Age", "Look", "Arc", "Actor", "Photo"}
)

// fieldOf is the field a key names (-1: none: the line is the note's).
func fieldOf(key string) int {
	switch strings.ToLower(strings.Join(strings.Fields(key), " ")) {
	case "age":
		return fieldAge
	case "look", "looks", "description", "appearance":
		return fieldLook
	case "arc":
		return fieldArc
	case "actor", "casting", "cast", "played by":
		return fieldActor
	case "photo", "picture", "headshot", "cast photo":
		return fieldPhoto
	}
	return -1
}

// charEntry is a character of the list.
type charEntry struct {
	Name   string             // as written
	Note   string             // the note: the line after the name, then the others, one a line
	Fields [fieldCount]string // age, look, arc, actor, photo
	// from, to are its lines in the block's inside: [from, to)
	from, to int
}

// key is the entry's name as cues are matched.
func (e charEntry) key() string { return charKey(e.Name) }

// empty reports whether the entry says nothing but the name.
func (e charEntry) empty() bool {
	if strings.TrimSpace(e.Note) != "" {
		return false
	}
	for _, f := range e.Fields {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}

// charKey is how a name is matched: a cue's name (Lexington's: without @,
// ^, extension and notes) in capitals, its words one space apart.
func charKey(name string) string { return lexreport.CueName(name) }

// charList is the block read: its lines as written and the entries.
type charList struct {
	lines   []string
	entries []charEntry
}

// parseCharList reads the block's inside.
func parseCharList(inside string) charList {
	l := charList{}
	if inside == "" {
		return l
	}
	l.lines = strings.Split(inside, "\n")
	cur := -1
	var note []string
	end := func(to int) {
		if cur < 0 {
			return
		}
		e := &l.entries[cur]
		for len(note) > 0 && strings.TrimSpace(note[len(note)-1]) == "" {
			note = note[:len(note)-1]
		}
		e.Note, e.to = strings.Join(note, "\n"), to
		cur, note = -1, nil
	}
	for i, line := range l.lines {
		if strings.TrimSpace(line) == "" {
			if cur >= 0 && indentedAfter(l.lines, i) {
				note = append(note, "")
				continue
			}
			end(i)
			continue
		}
		if !startsIndented(line) {
			end(i)
			name, rest, _ := strings.Cut(line, ":")
			l.entries = append(l.entries, charEntry{Name: strings.TrimSpace(name), from: i})
			cur = len(l.entries) - 1
			if r := strings.TrimSpace(rest); r != "" {
				note = append(note, r)
			}
			continue
		}
		if cur < 0 {
			continue // indented before any name: kept as written
		}
		t := strings.TrimSpace(line)
		if key, value, ok := strings.Cut(t, ":"); ok {
			if f := fieldOf(key); f >= 0 && l.entries[cur].Fields[f] == "" {
				l.entries[cur].Fields[f] = strings.TrimSpace(value)
				continue
			}
		}
		note = append(note, t)
	}
	end(len(l.lines))
	return l
}

func startsIndented(line string) bool {
	return line != "" && (line[0] == ' ' || line[0] == '\t')
}

// indentedAfter reports whether the next line with words after line i is
// indented: a blank line inside an entry's note.
func indentedAfter(lines []string, i int) bool {
	for _, l := range lines[i+1:] {
		if strings.TrimSpace(l) != "" {
			return startsIndented(l)
		}
	}
	return false
}

// find is the index of the entry for name (by charKey; -1: none).
func (l charList) find(name string) int {
	k := charKey(name)
	for i, e := range l.entries {
		if e.key() == k {
			return i
		}
	}
	return -1
}

// entryLines are an entry as Accolade writes it.
func entryLines(e charEntry) []string {
	note := strings.Split(strings.Trim(e.Note, "\n"), "\n")
	head := strings.TrimSpace(e.Name)
	if first := strings.TrimSpace(note[0]); first != "" {
		head += ": " + first
	}
	out := []string{head}
	for f, v := range e.Fields {
		if v = strings.Join(strings.Fields(v), " "); v != "" {
			out = append(out, "  "+fieldKeys[f]+": "+v)
		}
	}
	for _, n := range note[1:] {
		if n = strings.TrimRightFunc(n, unicode.IsSpace); n == "" {
			out = append(out, "")
		} else {
			out = append(out, "  "+strings.TrimSpace(n))
		}
	}
	for len(out) > 1 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// inside is the block's inside with the entries as they are now: an entry
// with its lines (from, to) unchanged is kept as written, one changed
// written anew, one new (from < 0) put last.
func (l charList) inside(entries []charEntry) string {
	out := make([]string, 0, len(l.lines)+2)
	kept := map[int]charEntry{}
	var added []charEntry
	for _, e := range entries {
		if e.from < 0 {
			added = append(added, e)
		} else {
			kept[e.from] = e
		}
	}
	old := map[int]charEntry{}
	for _, e := range l.entries {
		old[e.from] = e
	}
	for i := 0; i < len(l.lines); {
		o, isEntry := old[i]
		if !isEntry {
			out = append(out, l.lines[i])
			i++
			continue
		}
		if e, ok := kept[i]; ok {
			if sameEntry(o, e) {
				out = append(out, l.lines[o.from:o.to]...)
			} else {
				out = append(out, entryLines(e)...)
			}
		}
		i = o.to
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" && len(added) > 0 {
		out = out[:len(out)-1]
	}
	for _, e := range added {
		out = append(out, entryLines(e)...)
	}
	return strings.Join(out, "\n")
}

func sameEntry(a, b charEntry) bool {
	return a.Name == b.Name && a.Note == b.Note && a.Fields == b.Fields
}

// charListOf is the script's character list.
func charListOf(text string) charList { return screenplayList.list(text) }

// listedNames are the names in the script's character list, as written:
// the `list` cue rule's (a cue is a name of the list).
func listedNames(text string) []string {
	l := charListOf(text)
	out := make([]string, 0, len(l.entries))
	for _, e := range l.entries {
		if e.Name != "" {
			out = append(out, e.Name)
		}
	}
	return out
}

// isListed reports whether name (a cue's, extension and all) is in the
// script's character list.
func isListed(text, name string) bool {
	return charListOf(text).find(name) >= 0
}

// addedName is the edit that puts name in the script's character list
// (where in runes, what; ok false if it is there already): the editor's,
// for a line marked Character that the language's rules would not read
// so (LOOP.md, the language rules), and Add to the List in the pane. The
// name is written as given, without its extension.
func addedName(text, name string) (start, end int, block string, ok bool) {
	return screenplayList.added(text, cueName(name))
}

// added is the edit that puts name in the document's character list
// (ok false if it is there already), as addedName.
func (k charBlock) added(text, name string) (start, end int, block string, ok bool) {
	name = strings.TrimSpace(name)
	l := k.list(text)
	if name == "" || l.find(name) >= 0 {
		return 0, 0, "", false
	}
	entries := append(append([]charEntry(nil), l.entries...), charEntry{Name: name, from: -1})
	start, end, block = k.with(text, l.inside(entries))
	return start, end, block, true
}
