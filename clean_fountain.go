package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
)

// Export > Clean Fountain… (LOOP.md, Joop 2026-11-20): a .fountain of the
// script for other apps and other people, without Accolade's data — its
// blocks at the end (Notes, Source, Adaptation, Production, Characters)
// and the links to the source, [[from: …]] — always, and as the boxes
// say (remembered) the writer's own: notes, boneyard, scene numbers,
// OMITTED headings, the title page, the outline. What is stripped and
// how is Lexington's (fountain.Strip, the command line's -strip), its
// presets too. The script itself is not touched: the copy is written
// where one says (the browser downloads it), never over the script.
// A script in another language is written as standard Fountain, as a
// plain .fountain of it is (standardFountain), which other apps read.

// cleanPref is the settings key that remembers an option's box.
func cleanPref(key string) string { return "clean-fountain-" + key }

// cleanLabels are the boxes' labels, by fountain.StripKeys' keys: short,
// for a phone; cleanHint says the rest under them.
var cleanLabels = map[string]string{
	"notes":     "All notes [[…]]",
	"boneyard":  "All boneyard /* … */",
	"numbers":   "Scene numbers #12#",
	"omitted":   "OMITTED headings",
	"titlepage": "The title page",
	"outline":   "Sections and synopses",
}

const cleanHint = "The boneyard takes an omitted scene's kept text too; an OMITTED heading goes with the scene kept under it; " +
	"sections (# Act One) and synopses (= …) are the writer's outline."

// presetLabel is a preset's name as the list shows it: before its
// parenthesis, which presetAbout says under the list.
func presetLabel(name string) string {
	short, _, _ := strings.Cut(name, " (")
	return short
}

// presetAbout is a preset name's parenthesis, as a sentence.
func presetAbout(name string) string {
	_, about, ok := strings.Cut(name, " (")
	if !ok {
		return ""
	}
	about = strings.TrimSuffix(about, ")")
	return strings.ToUpper(about[:1]) + about[1:] + "."
}

// cleanFountainDlg is the dialog and its controls.
type cleanFountainDlg struct {
	w       *MainWindow
	d       dialog.Dialog
	text    string // the script, as the dialog opened
	preset  *widget.Select
	about   *widget.Label   // what the preset takes out
	checks  []*widget.Check // in fountain.StripKeys' order
	summary *widget.Label
	export  *widget.Button
	filling bool // a preset is setting the boxes
}

// showCleanFountain opens Export Clean Fountain….
func (w *MainWindow) showCleanFountain() {
	if w.markdown {
		return
	}
	c := w.newCleanFountain()
	c.d.Show()
}

// newCleanFountain is the dialog for the script now (not shown).
func (w *MainWindow) newCleanFountain() *cleanFountainDlg {
	c := &cleanFountainDlg{w: w, text: w.textEditor.Text()}
	w.cleanAsked = c
	var names []string
	for _, p := range fountain.StripPresets {
		names = append(names, presetLabel(p.Name))
	}
	c.preset = widget.NewSelect(names, c.choosePreset)
	c.preset.PlaceHolder = "Your own"
	fyne.SetAccessibleLabel(c.preset, "Preset")
	boxes := container.NewVBox()
	for _, k := range fountain.StripKeys {
		ch := widget.NewCheck(cleanLabels[k.Key], func(bool) { c.changed() })
		ch.Checked = w.settings.GetBoolean(cleanPref(k.Key))
		c.checks = append(c.checks, ch)
		boxes.Add(ch)
	}
	intro := widget.NewLabel("A copy of the script for other apps and other people. Accolade's own data always goes: " +
		"the blocks at the end (Notes, Source, Adaptation, Production, Characters) and the links to the source, [[from: …]]. " +
		"The script itself is not changed.")
	intro.Wrapping = fyne.TextWrapWord
	parts := []fyne.CanvasObject{intro}
	if lang := w.cleanLanguage(); lang != "" {
		note := widget.NewLabel("Written as standard Fountain, as other apps read it: " + languageName(lang) +
			" lines that would read otherwise get their force marks.")
		note.Wrapping = fyne.TextWrapWord
		note.Importance = widget.LowImportance
		parts = append(parts, note)
	}
	bold := func(s string) fyne.CanvasObject {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	c.summary = widget.NewLabel("")
	c.summary.Wrapping = fyne.TextWrapWord
	c.about = widget.NewLabel("")
	c.about.Wrapping = fyne.TextWrapWord
	c.about.Importance = widget.LowImportance
	hint := widget.NewLabel(cleanHint)
	hint.Wrapping = fyne.TextWrapWord
	hint.Importance = widget.LowImportance
	parts = append(parts, bold("Preset"), c.preset, c.about, bold("Also take out"), boxes, hint, c.summary)
	cancel := widget.NewButton("Cancel", func() { c.d.Hide() })
	c.export = widget.NewButton("Export…", c.save)
	c.export.Importance = widget.HighImportance
	body := container.NewVScroll(container.NewVBox(parts...))
	c.d = dialog.NewCustomWithoutButtons("Export Clean Fountain", container.NewBorder(nil, w.dialogButtons(cancel, c.export), nil, nil, body), w.fyneWindow)
	c.d.Resize(fitDialog(w.fyneWindow, 560, 700))
	c.changed()
	return c
}

// options are the boxes as Strip's options.
func (c *cleanFountainDlg) options() fountain.StripOptions {
	var o fountain.StripOptions
	for i, k := range fountain.StripKeys {
		*k.Of(&o) = c.checks[i].Checked
	}
	return o
}

// choosePreset sets the boxes as the preset has them.
func (c *cleanFountainDlg) choosePreset(name string) {
	i := slices.IndexFunc(fountain.StripPresets, func(p fountain.StripPreset) bool { return presetLabel(p.Name) == name })
	if i < 0 || c.filling {
		return
	}
	o := fountain.StripPresets[i].Options
	c.filling = true
	for j, k := range fountain.StripKeys {
		c.checks[j].SetChecked(*k.Of(&o))
	}
	c.filling = false
	c.changed()
}

// changed remembers the boxes, shows the preset they are (or "Your
// own") and what the copy keeps.
func (c *cleanFountainDlg) changed() {
	if c.filling || c.summary == nil {
		return
	}
	o := c.options()
	for i, k := range fountain.StripKeys {
		c.w.settings.SetBoolean(cleanPref(k.Key), c.checks[i].Checked)
	}
	name, about := "", "Your own choice of what goes."
	for _, p := range fountain.StripPresets {
		if p.Options == o {
			name, about = presetLabel(p.Name), presetAbout(p.Name)
		}
	}
	c.about.SetText(about)
	c.filling = true
	if name == "" {
		c.preset.ClearSelected()
	} else {
		c.preset.SetSelected(name)
	}
	c.filling = false
	was, now := lineCount(c.text), lineCount(c.w.cleanText(c.text, o))
	if was == now {
		c.summary.SetText("Nothing to take out: the copy is the script as it is.")
	} else {
		c.summary.SetText(fmt.Sprintf("The copy: %s of the script's %d.", plural(now, "line"), was))
	}
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimSuffix(s, "\n"), "\n") + 1
}

// cleanLanguage is the language whose script is written as standard
// Fountain in the copy, as Save writes a plain .fountain ("" for none).
func (w *MainWindow) cleanLanguage() string {
	if w.markdown || !w.languageSaid || !w.lex().dialect(w.language) {
		return ""
	}
	return w.language
}

// cleanText is the script text as the clean copy has it.
func (w *MainWindow) cleanText(text string, o fountain.StripOptions) string {
	out := fountain.Strip(text, w.sceneHeaders(), o)
	if lang := w.cleanLanguage(); lang != "" {
		out = w.lex().standardFountain(out, lang)
	}
	return out
}

// cleanFileName is the copy's name: the script's, less its extensions,
// " - clean.fountain" (no language part: the copy is standard Fountain).
func (w *MainWindow) cleanFileName() string {
	base := w.scriptName()
	for filepath.Ext(base) != "" && len(base) > len(filepath.Ext(base)) {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	if base == "" {
		base = "Untitled"
	}
	return base + " - clean.fountain"
}

// save asks where to write the copy (the browser downloads it) and
// writes it there.
func (c *cleanFountainDlg) save() {
	c.d.Hide()
	c.w.saveNamedFile(c.w.cleanFileName(), c.writeTo)
}

// writeTo writes the copy to p, never over the script itself.
func (c *cleanFountainDlg) writeTo(p string) error {
	w := c.w
	if w.currentFile != "" && sameFile(p, w.currentFile) {
		return errors.New("that is the script itself: the clean copy needs a name of its own")
	}
	if err := writeFileAtomic(p, []byte(w.cleanText(c.text, c.options()))); err != nil {
		return err
	}
	w.setStatus("Clean Fountain saved as " + filepath.Base(p))
	return nil
}
