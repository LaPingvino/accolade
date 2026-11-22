package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// The script format is one of Lexington's presets (screenplay, stage
// play, radio, ...): how the editor and the PDF and HTML exports lay
// the script out. It is the "script-format" setting, the preset's key.

// scriptFormats are the formats: Lexington's presets, Accolade's and
// one's own (format files, format_buttons.go), then the element sets of
// Lexington's settings; a script's (lexConf.formats) with what it
// brought over them.
func scriptFormats() []rules.Preset { return machineLex().formats() }

func (c lexConf) formats() []rules.Preset {
	out := presetsIn(builtinFormatDir(), userFormatDir())
	for _, k := range c.customFormats() {
		if !slices.ContainsFunc(out, func(p rules.Preset) bool { return p.Key == k }) {
			out = append(out, rules.Preset{Key: k, Name: k, Description: "Your own, from the Lexington settings.", Elements: c.Elements[k]})
		}
	}
	if c.carried != nil && len(c.carried.formats) > 0 {
		out, _ = rules.WithPresetFiles(out, c.carried.formats)
	}
	return out
}

// presetsSeen is the last reading of the format folders, kept while
// their .toml files are the same (in order, by name and content; the
// folders' own paths do not matter): the formats are asked for on every
// refresh of the header, the ribbon, the panes, and decoding them each
// time was most of that.
var presetsSeen struct {
	sync.Mutex
	key  string
	list []rules.Preset
}

// presetsIn is rules.PresetsIn(dirs...), decoded again only when a format
// file came, went or changed.
func presetsIn(dirs ...string) []rules.Preset {
	var key strings.Builder
	for _, dir := range dirs {
		fs, _ := filepath.Glob(filepath.Join(dir, "*.toml"))
		key.WriteString("\x00dir\x00")
		for _, f := range fs { // (Glob's order is PresetsIn's: sorted)
			b, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			fmt.Fprintf(&key, "%s\x00%d\x00", filepath.Base(f), len(b))
			key.Write(b)
		}
	}
	p := &presetsSeen
	p.Lock()
	defer p.Unlock()
	if p.list == nil || p.key != key.String() {
		p.list, _ = rules.PresetsIn(dirs...)
		p.key = key.String()
	}
	return slices.Clone(p.list)
}

// scriptFormat is the format with key, or the screenplay's.
func scriptFormat(key string) rules.Preset { return machineLex().format(key) }

func (c lexConf) format(key string) rules.Preset {
	for _, p := range c.formats() {
		if p.Key == key {
			return p
		}
	}
	p, _ := rules.GetPreset(rules.ScreenplayPreset)
	return p
}

// hasFormat tells whether there is a format with key.
func (c lexConf) hasFormat(key string) bool {
	return slices.ContainsFunc(c.formats(), func(p rules.Preset) bool { return p.Key == key })
}

// formatElements are the element rules of the format with key.
func (c lexConf) formatElements(key string) rules.Set {
	if p := c.format(key); p.Elements != nil {
		return p.Elements
	}
	return rules.Default
}

// scriptFormatKey is the key of the script's format: the one its
// .accolade names, else the one in the settings.
func (w *MainWindow) scriptFormatKey() string {
	if w.ownFormat != "" {
		return w.ownFormat
	}
	return w.settings.GetString("script-format")
}

// setScriptFormat makes key the script's format: the settings' too (the
// one new scripts start with), unless only the script has it.
func (w *MainWindow) setScriptFormat(key string) {
	if key != w.scriptFormatKey() {
		defer w.pagesAgain() // a preview's pages in the new format
	}
	if machineLex().hasFormat(key) {
		w.ownFormat = ""
		w.settings.SetString("script-format", key)
		w.followInPreferences("script-format")
		return
	}
	w.ownFormat = key
}

// currentScriptFormat is the element rules of the script's format.
func (w *MainWindow) currentScriptFormat() rules.Set {
	return w.lex().formatElements(w.scriptFormatKey())
}

// newScriptFormatSelect is a choice of the formats by name (conf's: the
// machine's, or a script's), showing the chosen one's description in
// desc; changed gets the chosen key.
func newScriptFormatSelect(conf func() lexConf, desc *widget.Label, changed func(key string)) *widget.Select {
	return widget.NewSelect(conf().formatNames(), func(name string) {
		for _, p := range conf().formats() {
			if p.Name == name {
				if desc != nil {
					desc.SetText(p.Description)
				}
				if changed != nil {
					changed(p.Key)
				}
				return
			}
		}
	})
}

// scriptFormatNames are the formats' names, in order.
func scriptFormatNames() []string { return machineLex().formatNames() }

func (c lexConf) formatNames() []string {
	var names []string
	for _, p := range c.formats() {
		names = append(names, p.Name)
	}
	return names
}

// selectScriptFormat shows the format with key in s.
func (c lexConf) selectFormat(s *widget.Select, key string) { s.SetSelected(c.format(key).Name) }

// selectedFormat is the key of the format chosen in s.
func (c lexConf) selectedFormat(s *widget.Select) string {
	for _, p := range c.formats() {
		if p.Name == s.Selected {
			return p.Key
		}
	}
	return rules.ScreenplayPreset
}
