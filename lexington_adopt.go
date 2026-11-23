package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/BurntSushi/toml"

	"github.com/LaPingvino/lexington/rules"
)

// File > Add to My Settings… copies what a script brought
// (lexington_carried.go) into the machine's own settings: lexington.toml
// in the settings folder, one's own formats folder (the browser: the kept
// settings folder). It lists each thing; one the machine has under the
// same name, but otherwise, asks: keep mine, take theirs, or keep both
// (theirs under a new name, which this script then uses). Nothing is
// overwritten without that choice. lexington.toml is changed as text, so
// its comments and the rest stay; a change that would not read back as
// meant is refused, nothing written.

// adoptHow is what is done with one item.
type adoptHow int

const (
	adoptAdd        adoptHow = iota // a new one: added
	adoptKeepMine                   // a clash: the machine's stays
	adoptTakeTheirs                 // a clash: theirs replaces the machine's
	adoptKeepBoth                   // a clash: theirs added under another name
	adoptSkip                       // a new one left out
)

// adoptChoice is an item and what to do with it; name is keep both's.
type adoptChoice struct {
	item carriedItem
	how  adoptHow
	name string
}

// bothName is a name for theirs beside mine: key-2, -3… not yet taken.
func bothName(it carriedItem) string {
	for i := 2; ; i++ {
		n := it.key + "-" + strconv.Itoa(i)
		taken := false
		switch it.kind {
		case "language":
			taken = lexSettings.Knows(n)
		case "elements", "format":
			taken = machineLex().hasFormat(n)
		}
		if !taken {
			return n
		}
	}
}

// adopt copies the chosen items into the machine's settings. renamed are
// the new names keep both gave, by "kind:key".
func adopt(choices []adoptChoice) (renamed map[string]string, err error) {
	renamed = map[string]string{}
	text := ""
	if b, err := os.ReadFile(lexingtonConfig()); err == nil {
		text = string(b)
	}
	before, err := rules.Decode([]byte(text))
	if err != nil {
		return nil, fmt.Errorf("your lexington.toml cannot be read (%v): nothing was added", err)
	}
	want := map[rules.Item]rules.TOMLConf{} // what each changed item is to read back as
	nowText := text
	var formats []adoptChoice
	for _, c := range choices {
		it := c.item
		if c.how == adoptKeepMine || c.how == adoptSkip {
			continue
		}
		if it.kind == "format" {
			formats = append(formats, c)
			continue
		}
		item, key := rules.Item{Kind: it.kind, Key: it.key}, it.key
		theirs := it.theirs
		if c.how == adoptKeepBoth {
			if it.kind == "standard" {
				continue // one standard only
			}
			name := strings.TrimSpace(c.name)
			if _, taken := lexSettings.Of(rules.Item{Kind: it.kind, Key: name}); taken || (it.kind == "elements" && machineLex().hasFormat(name)) {
				return nil, fmt.Errorf("you have %q already: give theirs another name", name)
			}
			if theirs, err = rules.Renamed(theirs, item, name); err != nil {
				return nil, err
			}
			item.Key, key = name, name
			renamed[it.kind+":"+it.key] = name
		}
		if nowText, err = setConfItem(nowText, item, key, theirs); err != nil {
			return nil, err
		}
		want[item], _ = theirs.Of(item)
	}
	if nowText != text {
		after, err := rules.Decode([]byte(nowText))
		if err != nil {
			return nil, fmt.Errorf("adding to your lexington.toml would break it (%v): nothing was added; add them yourself in Preferences > Script > Edit Lexington Settings…", err)
		}
		if err := sameButFor(before, after, want); err != nil {
			return nil, fmt.Errorf("your lexington.toml is written in a way Accolade cannot add to safely (%v): nothing was added; add them yourself in Preferences > Script > Edit Lexington Settings…", err)
		}
	}
	// formats: what they start from first, renamed bases followed
	slices.SortStableFunc(formats, func(a, b adoptChoice) int {
		if baseOf(b.item.theirsFile) == a.item.key {
			return -1
		}
		if baseOf(a.item.theirsFile) == b.item.key {
			return 1
		}
		return 0
	})
	for _, c := range formats {
		if c.how == adoptKeepBoth {
			name := formatKey(c.name)
			if name == "" || machineLex().hasFormat(name) {
				return nil, fmt.Errorf("there is a format %q already: give theirs another name", c.name)
			}
			renamed["format:"+c.item.key] = name
		}
	}
	if nowText != text {
		if err := saveLexingtonSettings(nowText); err != nil {
			return nil, err
		}
	}
	for _, c := range formats {
		key, data := c.item.key, string(c.item.theirsFile)
		if n, ok := renamed["format:"+key]; ok {
			var spec formatSpec
			_, _ = toml.Decode(data, &spec)
			data = setRootKey(data, "Name", tomlValue(spec.Name+" ("+n+")"))
			key = n
		}
		if b := baseOf([]byte(data)); renamed["format:"+b] != "" {
			data = setRootKey(data, "Base", tomlValue(renamed["format:"+b]))
		}
		if err := saveFormat(key, data); err != nil {
			return renamed, fmt.Errorf("the format %s: %v", key, err)
		}
	}
	return renamed, nil
}

// baseOf is a format file's Base.
func baseOf(data []byte) string {
	var spec formatSpec
	_, _ = toml.Decode(string(data), &spec)
	return spec.Base
}

// sameButFor checks that after is before with want's items as want has
// them, and nothing else changed.
func sameButFor(before, after rules.TOMLConf, want map[rules.Item]rules.TOMLConf) error {
	for it, w := range want {
		got, _ := after.Of(it)
		if !rules.Same(got, w) {
			return fmt.Errorf("%s %s does not read back as added", it.Kind, it.Key)
		}
	}
	items := before.Items()
	for _, it := range after.Items() {
		if !slices.Contains(items, it) {
			items = append(items, it)
		}
	}
	for _, it := range items {
		if _, changed := want[it]; changed {
			continue
		}
		a, _ := before.Of(it)
		b, _ := after.Of(it)
		if !rules.Same(a, b) {
			return fmt.Errorf("%s %s would change too", it.Kind, it.Key)
		}
	}
	return nil
}

// setConfItem is lexington.toml's text with item (named key there) as
// conf (a configuration of only that item) has it.
func setConfItem(text string, item rules.Item, key string, conf rules.TOMLConf) (string, error) {
	switch item.Kind {
	case "language":
		// as theirs has it, in full (Lexington's [Languages.key]); the
		// tables of older files lose the key
		for _, table := range []string{"Scenes", "Matchups", "Directions"} {
			text = dropTableKey(text, table, key)
		}
		l, ok := conf.Languages[key]
		if !ok {
			l = conf.Language(key)
		}
		block, err := (rules.TOMLConf{Languages: map[string]rules.Language{key: l}}).Encode()
		if err != nil {
			return "", err
		}
		text = dropTables(text, "Languages."+key)
		return appendTables(text, stripTable(string(block), "Languages")), nil
	case "extension":
		return setTableKey(text, "Extensions", key, tomlValue(conf.Extensions[key])), nil
	case "elements":
		block, err := (rules.TOMLConf{Elements: conf.Elements}).Encode()
		if err != nil {
			return "", err
		}
		text = dropTables(text, "Elements."+key)
		return appendTables(text, stripTable(string(block), "Elements")), nil
	case "standard":
		return setRootKey(text, "Standard", tomlValue(conf.Standard)), nil
	}
	return "", errors.New("nothing to add for " + item.Kind)
}

// The text edits of adopt, on toml_text.go's: a key set in [table], one
// taken out, a root key set, tables taken out or added. One that cannot
// be made (a dotted key, a value written inline) leaves the text as it
// was, which adopt then refuses (it reads the result back).

func setTableKey(text, table, key, value string) string {
	out, err := setKey(text, append(splitTOMLKey(table), key), value)
	if err != nil {
		return text
	}
	return out
}

func dropTableKey(text, table, key string) string {
	lines := strings.Split(text, "\n")
	s, ok := tomlTable(tomlSections(lines), splitTOMLKey(table))
	if !ok {
		return text
	}
	if i := keyLine(lines, s, key); i >= 0 {
		return strings.Join(slices.Delete(lines, i, valueEnd(lines, i)), "\n")
	}
	return text
}

func setRootKey(text, key, value string) string {
	out, err := setKey(text, []string{key}, value)
	if err != nil {
		return text
	}
	return out
}

// dropTables takes out [name] and every [name.…] table.
func dropTables(text, name string) string { return dropTablesAt(text, splitTOMLKey(name)) }

// stripTable takes an encoded block's bare [name] header out (its
// subtables say their whole name), so it can follow a file that has one.
func stripTable(block, name string) string {
	var out []string
	for _, l := range strings.Split(block, "\n") {
		if h, ok := tomlHeader(l); ok && h == name {
			continue
		}
		out = append(out, l)
	}
	return strings.TrimLeft(strings.Join(out, "\n"), "\n")
}

// appendTables adds tables at the end of the text.
func appendTables(text, block string) string {
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if text != "" {
		text += "\n"
	}
	return text + block
}

// addToMySettings shows what the script brought and copies what is
// chosen into the machine's settings.
func (w *MainWindow) addToMySettings() {
	items := w.carried.toAdd()
	if len(items) == 0 {
		dialog.ShowInformation("Add to My Settings", "This script brought nothing your settings do not have already.", w.fyneWindow)
		return
	}
	a := &adoptDialog{w: w}
	rows := container.NewVBox()
	intro := widget.NewLabel("This script brought these. It uses them already; adding them makes them yours, for every script.")
	intro.Wrapping = fyne.TextWrapWord
	rows.Add(intro)
	for _, it := range items {
		rows.Add(widget.NewSeparator())
		rows.Add(a.row(it))
	}
	scroll := container.NewVScroll(rows)
	d := dialog.NewCustomConfirm("Add to My Settings", "Add", "Cancel", scroll, func(ok bool) {
		if ok {
			a.apply()
		}
	}, w.fyneWindow)
	a.dialog = d
	d.Resize(fitDialog(w.fyneWindow, 560, 900))
	d.Show()
	w.adoptAsked = a
}

// adoptDialog is the Add to My Settings dialog as shown.
type adoptDialog struct {
	w       *MainWindow
	dialog  dialog.Dialog
	choices []*adoptChoice
	// by item, the widgets (for tests)
	picks map[string]*widget.RadioGroup
	names map[string]*widget.Entry
	adds  map[string]*widget.Check
}

const (
	pickMine   = "Keep mine"
	pickTheirs = "Take theirs"
	pickBoth   = "Keep both"
)

// row is one item: what it is, how it differs, and the choice.
func (a *adoptDialog) row(it carriedItem) fyne.CanvasObject {
	c := &adoptChoice{item: it, how: adoptAdd}
	a.choices = append(a.choices, c)
	if a.picks == nil {
		a.picks, a.names, a.adds = map[string]*widget.RadioGroup{}, map[string]*widget.Entry{}, map[string]*widget.Check{}
	}
	id := it.kind + ":" + it.key
	title := widget.NewLabelWithStyle(upperFirst(it.label()), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	title.Wrapping = fyne.TextWrapWord
	box := container.NewVBox(title)
	if detail := it.theirsDetail(); detail != "" {
		l := widget.NewLabel(detail)
		l.Wrapping = fyne.TextWrapWord
		box.Add(l)
	}
	if it.state == itemNew {
		check := widget.NewCheck("Add it (you have none of this name)", func(on bool) {
			c.how = map[bool]adoptHow{true: adoptAdd, false: adoptSkip}[on]
		})
		check.SetChecked(true)
		a.adds[id] = check
		box.Add(check)
		return box
	}
	mine := widget.NewLabel(it.mineDetail())
	mine.Wrapping = fyne.TextWrapWord
	box.Add(mine)
	options := []string{pickMine, pickTheirs, pickBoth}
	if it.kind == "standard" {
		options = options[:2]
	}
	name := widget.NewEntry()
	name.SetText(bothName(it))
	fyne.SetAccessibleLabel(name, "Theirs as")
	nameRow := labelled("Theirs as:", name)
	name.Disable() // until Keep both
	pick := widget.NewRadioGroup(options, func(s string) {
		c.how = map[string]adoptHow{pickMine: adoptKeepMine, pickTheirs: adoptTakeTheirs, pickBoth: adoptKeepBoth}[s]
		if c.how == 0 {
			c.how = adoptKeepMine
		}
		if c.how == adoptKeepBoth {
			name.Enable()
		} else {
			name.Disable()
		}
	})
	pick.Required = true
	pick.SetSelected(pickMine) // the safe one: nothing of mine changes
	fyne.SetAccessibleLabel(pick, "Mine or theirs: "+it.label())
	name.OnChanged = func(s string) { c.name = s }
	c.name = name.Text
	a.picks[id], a.names[id] = pick, name
	box.Add(pick)
	if it.kind != "standard" {
		box.Add(nameRow)
	}
	return box
}

// apply copies the choices, then the script uses what keep both named.
func (a *adoptDialog) apply() {
	w := a.w
	var choices []adoptChoice
	var added []string
	for _, c := range a.choices {
		choices = append(choices, *c)
		if c.how == adoptAdd || c.how == adoptTakeTheirs || (c.how == adoptKeepBoth && c.item.kind != "standard") {
			added = append(added, c.item.label())
		}
	}
	renamed, err := adopt(choices)
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	if w.app != nil {
		w.app.scriptFormatsChanged()
	}
	// keep both: this script now uses theirs under its new name
	if n := renamed["language:"+w.language]; n != "" {
		w.setLanguage(n)
		w.languageSaid = true
		w.hasChanges = true
	} else {
		w.setLanguage(w.language) // the words as now
	}
	if n := cmp2(renamed["format:"+w.scriptFormatKey()], renamed["elements:"+w.scriptFormatKey()]); n != "" {
		w.ownFormat = n
		w.applyScriptFormat()
		w.hasChanges = true
	}
	w.updateTitle()
	if len(added) == 0 {
		w.setStatus("Nothing added: your settings are as they were")
		return
	}
	w.setStatus("Added to your settings: " + strings.Join(added, ", "))
}

func cmp2(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// upperFirst is s with a capital first letter.
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// theirsDetail says what the item holds as the script brought it.
func (it carriedItem) theirsDetail() string {
	return "Theirs: " + itemDetail(it.kind, it.key, it.theirs, it.theirsFile)
}

// mineDetail says what the machine has under the same name.
func (it carriedItem) mineDetail() string {
	if it.kind == "format" {
		if data, err := os.ReadFile(filepath.Join(userFormatDir(), it.key+".toml")); err == nil {
			return "Yours: " + itemDetail(it.kind, it.key, rules.TOMLConf{}, data)
		}
		return "Yours: Accolade's own " + machineLex().format(it.key).Name
	}
	mine, _ := lexSettings.Of(rules.Item{Kind: it.kind, Key: it.key})
	if it.kind == "standard" {
		mine.Standard = lexSettings.StandardLanguage()
	}
	return "Yours: " + itemDetail(it.kind, it.key, mine, nil)
}

// itemDetail is an item's content in a line: a language's words, a
// format's name and base, the standard's language.
func itemDetail(kind, key string, conf rules.TOMLConf, file []byte) string {
	switch kind {
	case "language":
		words := conf.Scenes(key)
		if len(words) == 0 {
			words = conf.Language(key).Matchup
		}
		return strings.Join(words, ", ")
	case "elements":
		return plural(len(conf.Elements[key]), "element") + " set"
	case "extension":
		return "." + key + " files: " + languageLabel(conf.Extensions[key])
	case "standard":
		return languageLabel(conf.Standard)
	case "format":
		var spec formatSpec
		_, _ = toml.Decode(string(file), &spec)
		s := spec.Name
		if spec.Base != "" {
			s += ", from " + machineLex().format(spec.Base).Name
		}
		if n := len(spec.Elements); n > 0 {
			s += ", " + plural(n, "element") + " changed"
		}
		return s
	}
	return ""
}
