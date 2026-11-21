package main

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// The Export dialog's filters: part of the script, by the tags on its
// headings and sections ([[London]], a color), or a character's sides.

// scriptTags are the tags on the script's scene headings and sections,
// as lex.FilterTags reads them: a note's whole text, a color note as its
// color ([[COLOR RED]] is red). The colors come first, in the picker's
// order, then the others alphabetically; each once, in the case it is
// first written in.
func scriptTags(text string, starts []string) []string {
	lines := strings.Split(text, "\n")
	seen := map[string]bool{}
	var colors, others []string
	for _, it := range outlineOf(text, starts) {
		for _, n := range lex.Notes(lines[it.Line]) {
			tag, c := strings.TrimSpace(n), lex.NoteColor("[["+n+"]]")
			if c != "" {
				tag = c
			}
			if tag == "" || seen[strings.ToLower(tag)] {
				continue
			}
			seen[strings.ToLower(tag)] = true
			if c != "" {
				colors = append(colors, tag)
			} else {
				others = append(others, tag)
			}
		}
	}
	order := func(c string) int {
		if i := slices.Index(beatColorNames, c); i >= 0 {
			return i
		}
		return len(beatColorNames)
	}
	sort.SliceStable(colors, func(i, j int) bool {
		if order(colors[i]) != order(colors[j]) {
			return order(colors[i]) < order(colors[j])
		}
		return colors[i] < colors[j]
	})
	sort.SliceStable(others, func(i, j int) bool { return strings.ToLower(others[i]) < strings.ToLower(others[j]) })
	return append(colors, others...)
}

// scriptSpeakers are the characters who speak in the script, as
// lex.Speaker names them (what lex.Sides takes), most speeches first.
func scriptSpeakers(text string, scenes []string) []string {
	if scenes == nil {
		scenes = englishScenes
	}
	count := map[string]int{}
	for _, l := range fountain.Parse(scenes, strings.NewReader(text)) {
		if l.Type == lex.TypeSpeaker {
			if name := lex.Speaker(l.Contents); name != "" {
				count[name]++
			}
		}
	}
	names := make([]string, 0, len(count))
	for n := range count {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if count[names[i]] != count[names[j]] {
			return count[names[i]] > count[names[j]]
		}
		return names[i] < names[j]
	})
	return names
}

// filterSuffix is what the filters add to an export's file name: " -
// London, red" for tags, " - EDWARD sides" for sides; characters a file
// name cannot have are left out.
func filterSuffix(tags []string, character string) string {
	var parts []string
	if len(tags) > 0 {
		parts = append(parts, strings.Join(tags, ", "))
	}
	if character != "" {
		parts = append(parts, character+" sides")
	}
	if len(parts) == 0 {
		return ""
	}
	clean := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) || r < ' ' {
			return -1
		}
		return r
	}, strings.Join(parts, " - "))
	return " - " + clean
}

// noSides is the character choice for no sides: the whole script.
const noSides = "Everyone (no sides)"

// sidesModes are the ways to make sides, as the dialog offers them; one
// must be chosen, none is the default.
var sidesModes = []struct {
	label string
	mode  lex.SidesMode
}{
	{"Full scenes", lex.SidesFull},
	{"Cue lines", lex.SidesCue},
	{"Lines only", lex.SidesIsolated},
}

// cueLengths are how long a cue is: the whole speech before, or its last
// words (as many as the entry after it says).
var cueLengths = []string{"Whole speech", "Last words:"}

// createFilterWidgets makes the filters' widgets: a check for each of
// the script's tags, the characters for sides and how to make them.
func (ed *ExportDialog) createFilterWidgets() {
	text := ed.window.textEditor.Text()
	changed := func() { ed.updateDefaultFilename() }
	for _, tag := range scriptTags(text, ed.window.sceneStarts()) {
		c := widget.NewCheck(tag, func(bool) { changed() })
		fyne.SetAccessibleLabel(c, "Only scenes and sections tagged "+tag)
		ed.tagChecks = append(ed.tagChecks, c)
	}
	ed.characterSelect = widget.NewSelect(append([]string{noSides}, scriptSpeakers(text, ed.window.sceneHeaders())...), func(string) {
		ed.showSides()
		changed()
	})
	ed.characterSelect.SetSelected(noSides)
	var labels []string
	for _, m := range sidesModes {
		labels = append(labels, m.label)
	}
	ed.sidesRadio = widget.NewRadioGroup(labels, func(string) { ed.showSides() })
	// side by side, but on a phone one under the other (the dialog scrolls down only)
	c := ed.window.fyneWindow.Canvas().Size()
	wide := c.Width == 0 || c.Width >= 600
	ed.sidesRadio.Horizontal = wide
	fyne.SetAccessibleLabel(ed.sidesRadio, "Sides: what of their scenes")
	ed.cueRadio = widget.NewRadioGroup(cueLengths, nil)
	ed.cueRadio.Horizontal = wide
	ed.cueRadio.Required = true
	ed.cueRadio.SetSelected(cueLengths[0])
	fyne.SetAccessibleLabel(ed.cueRadio, "Cue length")
	ed.cueWordsEntry = widget.NewEntry()
	ed.cueWordsEntry.SetText("8")
	ed.cueWordsEntry.OnChanged = func(string) { ed.cueRadio.SetSelected(cueLengths[1]) }
	fyne.SetAccessibleLabel(ed.cueWordsEntry, "Cue length in words")
}

// filterContent lays the filters out.
func (ed *ExportDialog) filterContent() fyne.CanvasObject {
	tags := container.NewGridWrap(fyne.NewSize(180, 36))
	for _, c := range ed.tagChecks {
		row := fyne.CanvasObject(c)
		if _, ok := noteRGB(lex.NoteColor("[[" + c.Text + "]]")); ok {
			r := newSwatch()
			showSwatch(r, lex.NoteColor("[["+c.Text+"]]"))
			row = container.NewBorder(nil, nil, container.NewCenter(r), nil, c)
		}
		tags.Add(row)
	}
	var tagPart fyne.CanvasObject = tags
	if len(ed.tagChecks) == 0 {
		tagPart = hint("No tags yet: a note on a scene heading or section, such as [[London]] or a color, is one.")
	}
	ed.cueBox = container.NewHBox(widget.NewLabel("Cue:"), ed.cueRadio, container.NewGridWrap(fyne.NewSize(60, ed.cueWordsEntry.MinSize().Height), ed.cueWordsEntry))
	ed.sidesBox = container.NewVBox(labelled("Sides:", ed.sidesRadio), ed.cueBox)
	ed.showSides()
	tagged := widget.NewLabel("Only scenes and sections tagged (none: all of them):")
	tagged.Wrapping = fyne.TextWrapWord
	return container.NewVBox(
		tagged,
		tagPart,
		labelled("Sides for character:", ed.characterSelect),
		ed.sidesBox,
	)
}

// showSides shows how to make sides with a character chosen, the cue's
// length with cue lines.
func (ed *ExportDialog) showSides() {
	if ed.sidesBox == nil {
		return
	}
	if ed.chosenCharacter() == "" {
		ed.sidesBox.Hide()
		return
	}
	ed.sidesBox.Show()
	if ed.chosenSides() == lex.SidesCue {
		ed.cueBox.Show()
	} else {
		ed.cueBox.Hide()
	}
}

// filterWidgets are the filters' widgets, for disabling with the options
// a format does not take.
func (ed *ExportDialog) filterWidgets() []fyne.Disableable {
	out := []fyne.Disableable{ed.characterSelect, ed.sidesRadio, ed.cueRadio, ed.cueWordsEntry}
	for _, c := range ed.tagChecks {
		out = append(out, c)
	}
	return out
}

// chosenTags are the tags checked.
func (ed *ExportDialog) chosenTags() []string {
	var out []string
	for _, c := range ed.tagChecks {
		if c.Checked {
			out = append(out, c.Text)
		}
	}
	return out
}

// chosenCharacter is whose sides, "" for none.
func (ed *ExportDialog) chosenCharacter() string {
	if s := ed.characterSelect.Selected; s != noSides {
		return s
	}
	return ""
}

// chosenSides is the mode chosen, "" for none yet.
func (ed *ExportDialog) chosenSides() lex.SidesMode {
	for _, m := range sidesModes {
		if m.label == ed.sidesRadio.Selected {
			return m.mode
		}
	}
	return ""
}

// chosenCueWords is the cue's length in words, 0 for the whole speech.
func (ed *ExportDialog) chosenCueWords() int {
	if ed.cueRadio.Selected != cueLengths[1] {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(ed.cueWordsEntry.Text))
	if err != nil || n < 1 {
		return -1
	}
	return n
}

// validateFilters asks for what sides need: how to make them, a cue's
// length.
func (ed *ExportDialog) validateFilters() error {
	who := ed.chosenCharacter()
	if who == "" {
		return nil
	}
	if ed.chosenSides() == "" {
		return fmt.Errorf("choose how to make %s's sides: full scenes, cue lines or lines only", who)
	}
	if ed.chosenSides() == lex.SidesCue && ed.chosenCueWords() < 0 {
		return fmt.Errorf("a cue's last words: give how many, as a number from 1")
	}
	return nil
}
