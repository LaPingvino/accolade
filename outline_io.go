package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/opml"
)

// The script's structure out of Accolade and into it (Joop 2026-11-24:
// "for the outline export, maybe also allow import and support opml.
// Software like Workflowy also support this." / "If you look for it, you
// need to be able to recognize it, don't hide it too well"): the
// sections, scene headings, synopses and notes the outline reads, as an
// outline (OPML) for an outliner, or a treatment to read (PDF, Word);
// and an outline back in, through File › Open (an OPML file, known by
// its content) or Edit › Paste as Structure (OPML or indented text from
// an outliner). The scenes' own text stays in the script: an outline
// has only their headings and synopses.

// outlineDoc is the script's structure as an outline: its title page,
// sections (holding what is in them), scenes (their headings as
// written), each heading's own synopsis as its note, and the synopses
// and notes standing on their own (beats) as items, where they are. The
// scenes' other lines and the blocks at the script's end are left out.
func outlineDoc(text string, starts []string) opml.Doc {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	end := strings.Count(text[:bodyEndByte(text)], "\n")
	if bodyEndByte(text) == len(text) {
		end = len(lines)
	}
	var d opml.Doc
	tp := titlePageEnd(lines)
	if tp > 0 {
		raw := lines[:tp]
		page := strings.TrimSpace(strings.Join(raw, "\n"))
		d.Title = scriptTitleOf(raw)
		d.Nodes = append(d.Nodes, &opml.Node{Text: "Title page", Note: page, Kind: opml.KindTitlePage})
	}
	items := outlineOf(text, starts)
	heads := map[int]outlineItem{}
	for _, it := range items {
		heads[it.Line] = it
	}
	type open struct {
		level int
		n     *opml.Node
	}
	var sections []open
	var scene *opml.Node
	add := func(n *opml.Node) { // where it is: in the scene, else the section, else the top
		switch {
		case scene != nil:
			scene.Kids = append(scene.Kids, n)
		case len(sections) > 0:
			s := sections[len(sections)-1].n
			s.Kids = append(s.Kids, n)
		default:
			d.Nodes = append(d.Nodes, n)
		}
	}
	bone := false
	for i := tp; i < end; {
		t := strings.TrimSpace(lines[i])
		switch { // a boneyard is not the script
		case bone:
			bone = !strings.Contains(t, "*/")
			i++
			continue
		case strings.HasPrefix(t, "/*"):
			bone = !strings.Contains(t[2:], "*/")
			i++
			continue
		}
		if t == "" {
			i++
			continue
		}
		j := i // the paragraph: lines i to j
		for j+1 < end && strings.TrimSpace(lines[j+1]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[j+1]), "/*") {
			if _, head := heads[j+1]; head {
				break
			}
			j++
		}
		if it, ok := heads[i]; ok {
			n := &opml.Node{}
			k := i + 1 // its own lines: synopses and notes touching it
			var note []string
			for ; k <= j; k++ {
				lt := strings.TrimSpace(lines[k])
				if isSynopsisLine(lt) {
					note = append(note, strings.TrimSpace(lt[1:]))
				} else if isNoteLine(lt) {
					note = append(note, lt)
				} else {
					break
				}
			}
			hasSyn := false
			for _, l := range note {
				hasSyn = hasSyn || !isNoteLine(l)
			}
			if !hasSyn && k > j { // its own synopsis may come after a blank line: the next paragraph's
				a := j + 1
				for a < end && strings.TrimSpace(lines[a]) == "" {
					a++
				}
				_, head := heads[a]
				if a < end && !head && isSynopsisLine(strings.TrimSpace(lines[a])) {
					b := a
					for b+1 < end && strings.TrimSpace(lines[b+1]) != "" {
						if _, h := heads[b+1]; h {
							break
						}
						b++
					}
					var own []string
					for _, l := range lines[a : b+1] {
						lt := strings.TrimSpace(l)
						if isSynopsisLine(lt) {
							own = append(own, strings.TrimSpace(lt[1:]))
						} else if isNoteLine(lt) {
							own = append(own, lt)
						} else {
							break // the scene's own text after it
						}
					}
					if len(note) == 0 && a == j+2 { // one blank line, as Fountain() writes it
						note, n.Apart, j = own, true, b
					}
				}
			}
			n.Note = strings.Join(note, "\n")
			if it.Scene {
				n.Kind, n.Text = opml.KindScene, t
				scene = nil
				add(n)
				scene = n
			} else {
				n.Kind, n.Text = opml.KindSection, strings.TrimSpace(strings.TrimLeft(t, "#"))
				scene = nil
				for len(sections) > 0 && sections[len(sections)-1].level >= it.Level {
					sections = sections[:len(sections)-1]
				}
				if it.Level != len(sections)+1 {
					n.Level = it.Level
				}
				add(n)
				sections = append(sections, open{it.Level, n})
			}
			i = j + 1
			continue
		}
		for _, p := range planItems(lines[i : j+1]) {
			add(p)
		}
		i = j + 1
	}
	return d
}

// planItems are a paragraph's synopses and notes as outline items: a
// paragraph that is a note ([[…]], over lines too) one item; else each run
// of synopsis lines ("= …") an item, its first line the item's text and
// the others (notes touching it too) its note, and each note line not in
// such a run an item of its own. Other lines (the scene's own text) are
// left out: a synopsis touching action is a beat still.
func planItems(ls []string) []*opml.Node {
	first := strings.TrimSpace(ls[0])
	last := strings.TrimSpace(ls[len(ls)-1])
	if strings.HasPrefix(first, "[[") && strings.HasSuffix(last, "]]") && !isSynopsisLine(first) {
		all := true // a note alone, over its lines
		for _, l := range ls {
			if isSynopsisLine(strings.TrimSpace(l)) {
				all = false
			}
		}
		if all {
			var rest []string
			for _, l := range ls[1:] {
				rest = append(rest, strings.TrimSpace(l))
			}
			return []*opml.Node{{Kind: opml.KindNote, Text: first, Note: strings.Join(rest, "\n")}}
		}
	}
	var out []*opml.Node
	var cur *opml.Node // the synopsis whose run goes on
	more := func(n *opml.Node, l string) {
		if n.Note != "" {
			n.Note += "\n"
		}
		n.Note += l
	}
	for _, l := range ls {
		t := strings.TrimSpace(l)
		switch {
		case isSynopsisLine(t) && cur != nil:
			more(cur, strings.TrimSpace(t[1:]))
		case isSynopsisLine(t):
			cur = &opml.Node{Kind: opml.KindSynopsis, Text: strings.TrimSpace(t[1:])}
			out = append(out, cur)
		case isNoteLine(t) && cur != nil:
			more(cur, t)
		case isNoteLine(t):
			out = append(out, &opml.Node{Kind: opml.KindNote, Text: t})
		default:
			cur = nil
		}
	}
	return out
}

// scriptTitleOf is the title page's Title, its lines joined.
func scriptTitleOf(lines []string) string {
	var out []string
	in := false
	for _, l := range lines {
		k, v, ok := strings.Cut(l, ":")
		switch {
		case ok && !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t"):
			in = strings.EqualFold(strings.TrimSpace(k), "title")
			if in && strings.TrimSpace(v) != "" {
				out = append(out, strings.TrimSpace(v))
			}
		case in && strings.TrimSpace(l) != "":
			out = append(out, strings.TrimSpace(l))
		}
	}
	return lex.Unstyled(strings.Join(out, " "))
}

// outlineOPML is the script's structure as an OPML file.
func outlineOPML(text string, starts []string) ([]byte, error) {
	var b bytes.Buffer
	err := opml.Write(&b, outlineDoc(text, starts))
	return b.Bytes(), err
}

// writeOutline writes the script's structure to out as OPML.
func writeOutline(text string, starts []string, out string) error {
	data, err := outlineOPML(text, starts)
	if err != nil {
		return err
	}
	return writeFileAtomic(out, data)
}

// scriptOfOutline is an outline (OPML, or indented text from an
// outliner) as a script's structure, in the language whose scene
// headings start with starts; whole, a flat list is put in a section of
// its own (withTitleSection), else (pasted into a script) its items stay
// beats where they go.
func scriptOfOutline(data []byte, starts []string, whole bool) (string, error) {
	isScene := func(l string) bool { return isSceneHeadingIn(l, starts) }
	d := opml.Doc{}
	if opml.Sniff(data) {
		var err error
		if d, err = opml.Read(bytes.NewReader(data)); err != nil {
			return "", err
		}
	} else {
		d.Nodes = opml.ParseIndented(string(data))
	}
	if whole {
		d = withTitleSection(d)
	}
	return opml.Fountain(d, isScene), nil
}

// withTitleSection is d with its items in one section named for its
// title, when it is a flat list from another outliner (none holds
// another): its items as beats with nothing above them would be in no
// section the outline and the cards show. Accolade's own outline, or one
// with sections, is as it is.
func withTitleSection(d opml.Doc) opml.Doc {
	if len(d.Nodes) == 0 {
		return d
	}
	for _, n := range d.Nodes {
		if n.Kind != "" || len(n.Kids) > 0 || opml.IsHeading(n.Text) {
			return d
		}
	}
	title := d.Title
	if title == "" {
		title = "Outline"
	}
	return opml.Doc{Title: d.Title, Nodes: []*opml.Node{{Text: title, Kind: opml.KindSection, Kids: d.Nodes}}}
}

// isOutlineFile tells whether the file at path is an outline: OPML by
// its name or its content.
func isOutlineFile(path string) bool {
	if strings.EqualFold(filepath.Ext(path), ".opml") {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	head := make([]byte, 2048)
	n, _ := f.Read(head)
	return opml.Sniff(head[:n])
}

// openOutline opens the outline at path as a new script, unsaved (its
// name suggested as a .fountain: Save never writes over the outline).
func (w *MainWindow) openOutline(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text, err := scriptOfOutline(data, w.sceneStarts(), true)
	if err != nil {
		return err
	}
	w.showImported(path, text, "an outline (OPML)")
	w.setStatus("Opened an outline (OPML) as sections and synopses: its items nested as # and ##, their notes their synopses, the leaves beats. Save keeps it as a script")
	return nil
}

// pasteStructure is Edit › Paste as Structure: an outline on the
// clipboard (OPML, or indented lines copied from Workflowy,
// OmniOutliner, a list) put in as sections, synopses and beats at the
// cursor's section boundary, as Insert Structure puts a structure, one
// undo step.
func (w *MainWindow) pasteStructure() {
	cb := w.clipboard()
	if cb == nil {
		return
	}
	text := cb.Content()
	if strings.TrimSpace(text) == "" {
		w.setStatus("Nothing on the clipboard: copy an outline (OPML, or indented lines) in your outliner first")
		return
	}
	script, err := scriptOfOutline([]byte(text), w.sceneStarts(), false)
	if err != nil {
		w.setStatus(err.Error())
		return
	}
	if strings.TrimSpace(script) == "" {
		w.setStatus("Nothing in the outline on the clipboard")
		return
	}
	w.insertStructureText(script, "The outline pasted")
}
