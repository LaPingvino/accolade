package main

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/fdx"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
	lexreport "github.com/LaPingvino/lexington/report"
)

// The breakdown in and out (docs/PRODUCTION.md §13): Final Draft's tags
// (FDX <TagData>, Lexington's fdx.ParseTags and FDXWriter.Tags) and
// Beat's (the Tags and TagDefinitions of its data block at the script's
// end: character ranges, read once when the script is opened, while
// Beat's "Text Length" says the text is as Beat left it). Both become
// tags anchored by their words, as Accolade's own.

// tagElementWords tags words [i, j) of the text as an existing element.
func (f *breakdownFile) tagElementWords(t *bdText, i, j int, elementID string) bool {
	key, at := t.anchorOf(i, j)
	tag := bdTag{Element: elementID, bdSceneKey: key, At: at}
	for _, x := range f.Tags {
		if x == tag {
			return false
		}
	}
	f.Tags = append(f.Tags, tag)
	return true
}

// tagRunes tags runes [start, end) of the text (to whole words) as an
// element; it tells whether a tag was added.
func (f *breakdownFile) tagRunes(t *bdText, start, end int, elementID string) bool {
	i, j := t.wordRange(start, end)
	if i == j || t.sceneOf[i] < 0 {
		return false
	}
	return f.tagElementWords(t, i, j, elementID)
}

// addSetElements adds a tag set's categories and elements (the standard
// categories by name; one's own kept).
func (f *breakdownFile) addSetElements(set *lex.TagSet) {
	for _, c := range set.Categories {
		if _, ok := f.category(c.Name); !ok {
			f.Categories = append(f.Categories, bdCategory{ID: c.ID, Name: c.Name, Color: c.Color})
		}
	}
	for _, e := range set.Elements {
		cat := e.Category
		if c, ok := set.Category(e.Category); ok {
			cat = c.Name
		}
		if _, ok := f.element(e.ID); !ok {
			f.Elements = append(f.Elements, bdElement{ID: e.ID, Category: cat, Name: e.Name})
		}
	}
}

// fdxBreakdown is an FDX's tags as a breakdown of the Fountain text
// written from it (lines: each row's line in text, -1 none).
func fdxBreakdown(data []byte, text string, starts []string) (breakdownFile, int) {
	f := breakdownFile{Version: 1}
	s, set, byRow, err := fdx.ParseTags(bytes.NewReader(data))
	if err != nil || set == nil || len(byRow) == 0 {
		return f, 0
	}
	f.addSetElements(set)
	lines := rowLines(s, text)
	bt := readBdText(text, starts)
	n := 0
	for row, tags := range byRow {
		if row >= len(lines) || lines[row] < 0 {
			continue
		}
		line := lines[row]
		lineText := string(bt.runes[bt.lineStart[line]:lineEndRune(bt, line)])
		for _, tag := range tags {
			b := lex.TagIndex(lineText, tag.Text, tag.At)
			if b < 0 {
				b = lex.TagIndex(lineText, tag.Text, 0)
			}
			if b < 0 {
				continue
			}
			start := bt.lineStart[line] + utf8.RuneCountInString(lineText[:b])
			end := start + utf8.RuneCountInString(tag.Text)
			for _, id := range tag.Elements {
				if f.tagRunes(bt, start, end, id) {
					n++
				}
			}
		}
	}
	f.pruneElements()
	return f, n
}

// lineEndRune is where a line of the text ends (before its line break).
func lineEndRune(t *bdText, line int) int {
	if line+1 < len(t.lineStart) {
		return t.lineStart[line+1] - 1
	}
	return len(t.runes)
}

// rowLines are the lines of the Fountain written from a screenplay that
// its rows are on (-1: none): each found in the next lines holding its
// text, as marksOnText finds them.
func rowLines(s lex.Screenplay, text string) []int {
	lines := strings.Split(text, "\n")
	out := make([]int, len(s))
	at := 0
	for r, row := range s {
		out[r] = -1
		key := strings.ToUpper(strings.TrimSpace(lex.StripNotes(row.Contents)))
		if key == "" || row.Type == lex.TypeBoneyard {
			continue
		}
		for k := at; k < len(lines) && k < at+40; k++ {
			if strings.Contains(strings.ToUpper(lines[k]), key) {
				out[r], at = k, k+1
				break
			}
		}
	}
	return out
}

// beatMarker and beatEnd open and close Beat's data block
// (BeatDocumentSettings.m).
const (
	beatMarker = "/* If you're seeing this, you can remove the following stuff - BEAT:"
	beatEnd    = "END_BEAT */"
)

// beatSettings is what Accolade reads of Beat's block.
type beatSettings struct {
	TextLength *int `json:"Text Length"`
	Tags       []struct {
		Range      []int  `json:"range"`
		Type       string `json:"type"`
		Definition string `json:"definition"`
	} `json:"Tags"`
	TagDefinitions []struct {
		Type string `json:"type"`
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"TagDefinitions"`
}

// beatBreakdown reads the tags of Beat's data block in text: their
// ranges (UTF-16 units of the text before the block) as tags by words.
// why says why none were read when there were some.
func beatBreakdown(text string, starts []string) (f breakdownFile, n int, why string) {
	f = breakdownFile{Version: 1}
	at := strings.LastIndex(text, beatMarker)
	if at < 0 {
		return f, 0, ""
	}
	block := text[at+len(beatMarker):]
	end := strings.LastIndex(block, beatEnd)
	if end < 0 {
		return f, 0, ""
	}
	var s beatSettings
	if err := json.Unmarshal([]byte(strings.TrimSpace(block[:end])), &s); err != nil || len(s.Tags) == 0 {
		return f, 0, ""
	}
	body := text[:at]
	units := utf16.Encode([]rune(body))
	// Beat counts the text without its block; the line breaks before the
	// block may or may not be counted
	if n := s.TextLength; n != nil && (*n > len(units) || *n < len(utf16.Encode([]rune(strings.TrimRight(body, "\n"))))) {
		return f, 0, "Beat's tags not read: the script was changed since Beat saved them"
	}
	// a UTF-16 offset → a rune offset
	runeAt := make([]int, len(units)+1)
	r := 0
	for u := 0; u < len(units); {
		runeAt[u] = r
		if utf16.IsSurrogate(rune(units[u])) && u+1 < len(units) {
			runeAt[u+1] = r
			u += 2
		} else {
			u++
		}
		r++
	}
	runeAt[len(units)] = r
	bt := readBdText(text, starts)
	ids := map[string]string{} // Beat's definition → the element
	for _, d := range s.TagDefinitions {
		cat := d.Type
		if c, ok := lex.TagCategoryOf(d.Type); ok {
			cat = c.Name
		}
		name := strings.TrimSpace(d.Name)
		if c, _ := lex.TagCategoryOf(cat); c.Name == lex.CastCategory {
			name = strings.ToUpper(name)
		}
		ids[d.ID] = f.elementFor(cat, name).ID
	}
	for _, t := range s.Tags {
		if len(t.Range) < 2 || t.Range[0] < 0 || t.Range[1] <= 0 || t.Range[0]+t.Range[1] > len(units) {
			continue
		}
		id := ids[t.Definition]
		if id == "" {
			continue
		}
		if f.tagRunes(bt, runeAt[t.Range[0]], runeAt[t.Range[0]+t.Range[1]], id) {
			n++
		}
	}
	f.pruneElements()
	return f, n, ""
}

// mergeBreakdown adds another breakdown's elements and tags to f (an
// element of the same category and name is the same; a tag already
// there is not added again). It tells how many tags were added.
func (f *breakdownFile) merge(o breakdownFile) int {
	for _, c := range o.Categories {
		if _, ok := f.category(c.Name); !ok {
			f.Categories = append(f.Categories, c)
		}
	}
	idOf := map[string]string{}
	for _, e := range o.Elements {
		idOf[e.ID] = f.elementFor(e.Category, e.Name).ID
	}
	n := 0
	for _, t := range o.Tags {
		t.Element = idOf[t.Element]
		if t.Element == "" {
			continue
		}
		dup := false
		for _, x := range f.Tags {
			dup = dup || x == t
		}
		if !dup {
			f.Tags = append(f.Tags, t)
			n++
		}
	}
	f.pruneElements()
	return n
}

// keepImportedBreakdown adds an imported breakdown to the script's.
func (w *MainWindow) keepImportedBreakdown(o breakdownFile, from string) int {
	w.breakdownGen++
	f := w.breakdownNow()
	n := f.merge(o)
	if n > 0 {
		w.saveBreakdown()
	}
	return n
}

// readBeatTags reads Beat's tags of a script just opened, once: those
// already in the breakdown are not added again.
func (w *MainWindow) readBeatTags() {
	if w.markdown || w.textEditor == nil {
		return
	}
	text := w.textEditor.Text()
	if !strings.Contains(text, beatMarker) {
		return
	}
	o, _, why := beatBreakdown(text, w.sceneStarts())
	if why != "" {
		w.setStatus(why)
		return
	}
	if n := w.keepImportedBreakdown(o, "Beat"); n > 0 {
		w.setStatus("Read " + plural(n, "tag") + " from Beat's data: the breakdown keeps them by their words")
	}
}

// withBreakdown is an FDX export job with the breakdown: the scenes'
// lengths, and the tags (the cast from the cues among them) when the
// script has a breakdown.
func (w *MainWindow) withBreakdown(j exportJob, text string) exportJob {
	j.sceneLengths = true
	if w.markdown {
		return j
	}
	f, bt, ps := w.breakdownNowFound()
	if !sameString(bt.text, text) {
		bt = readBdText(text, w.sceneStarts())
		ps = bt.place(f)
	}
	if len(f.Elements) == 0 {
		return j
	}
	set, lineTags := exportTags(f, bt, ps, j.scenes, learnedAliases(text))
	j.breakdown, j.lineTags = set, lineTags
	return j
}

// learnedAliases are the characters' other names the adaptation block
// learned (ANNIE → ANNA).
func learnedAliases(text string) map[string]string {
	aliases := map[string]string{}
	for _, a := range learnedOf(text).Aliases {
		for _, n := range a.Names {
			aliases[strings.ToUpper(n)] = a.Cue
		}
	}
	return aliases
}

// exportTags are the breakdown as Lexington writes it: the set, and the
// tags by line (from 1), the cast from the cues added.
func exportTags(f *breakdownFile, bt *bdText, ps []placed, scenes []string, aliases map[string]string) (*lex.TagSet, map[int][]lex.LineTag) {
	set := &lex.TagSet{}
	for _, c := range f.Categories {
		set.Categories = append(set.Categories, lex.TagCategory{ID: c.ID, Name: c.Name, Color: c.Color})
	}
	for _, e := range f.Elements {
		c, _ := f.category(e.Category)
		set.Elements = append(set.Elements, lex.TagElement{ID: e.ID, Category: c.ID, Name: e.Name})
	}
	lineTags := map[int][]lex.LineTag{}
	add := func(line int, text string, at int, id string) {
		for k, t := range lineTags[line] {
			if t.Text == text && t.At == at {
				if !slices.Contains(t.Elements, id) {
					lineTags[line][k].Elements = append(lineTags[line][k].Elements, id)
				}
				return
			}
		}
		lineTags[line] = append(lineTags[line], lex.LineTag{Text: text, At: at, Elements: []string{id}})
	}
	for _, p := range ps {
		if p.lost {
			continue
		}
		line, col := bt.lineCol(p.start)
		lineRunes := bt.runes[bt.lineStart[line]:lineEndRune(bt, line)]
		words := lex.TagPlain(string(bt.runes[p.start:p.end]), false)
		at := strings.Count(lex.TagPlain(string(lineRunes[:col]), false), words)
		add(line+1, words, at, p.element.ID)
	}
	// the cast from the cues: who speaks is tagged where they speak
	cast, _ := lex.TagCategoryOf(lex.CastCategory)
	castIDs := map[string]string{}
	for _, e := range f.Elements {
		if c, _ := f.category(e.Category); c.Name == lex.CastCategory {
			castIDs[strings.ToUpper(e.Name)] = e.ID
		}
	}
	if scenes == nil {
		scenes = englishScenes
	}
	body := bt.text[:bodyEndByte(bt.text)]
	for _, row := range fountain.ParseLines(scenes, strings.NewReader(body)) {
		if row.Type != lex.TypeSpeaker || row.Source <= 0 {
			continue
		}
		cue := lexreport.CueName(row.Contents)
		name := cue
		if a, ok := aliases[strings.ToUpper(cue)]; ok {
			name = a
		}
		if name == "" {
			continue
		}
		id := castIDs[strings.ToUpper(name)]
		if id == "" {
			id = nameID(name)
			castIDs[strings.ToUpper(name)] = id
			set.Elements = append(set.Elements, lex.TagElement{ID: id, Category: cast.ID, Name: name})
		}
		add(row.Source, cue, 0, id)
	}
	return set, lineTags
}
