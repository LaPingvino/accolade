package main

import (
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// The script format is one of Lexington's presets (screenplay, stage
// play, radio, ...): how the editor and the PDF and HTML exports lay
// the script out. It is the "script-format" setting, the preset's key.

// scriptFormats are the presets, then the formats of one's own from
// Lexington's settings (lexington_settings.go).
func scriptFormats() []rules.Preset {
	out := rules.Presets()
	for _, k := range customFormats() {
		out = append(out, rules.Preset{Key: k, Name: k, Description: "Your own, from the Lexington settings.", Elements: lexSettings.Elements[k]})
	}
	return out
}

// scriptFormat is the format with key, or the screenplay's.
func scriptFormat(key string) rules.Preset {
	for _, p := range scriptFormats() {
		if p.Key == key {
			return p
		}
	}
	p, _ := rules.GetPreset(rules.ScreenplayPreset)
	return p
}

// scriptFormatElements are the element rules of the format with key.
func scriptFormatElements(key string) rules.Set {
	if p := scriptFormat(key); p.Elements != nil {
		return p.Elements
	}
	return rules.Default
}

// currentScriptFormat is the element rules of the format in the settings.
func currentScriptFormat() rules.Set {
	return scriptFormatElements(GetSettings().GetString("script-format"))
}

// newScriptFormatSelect is a choice of the formats by name, showing the
// chosen one's description in desc; changed gets the chosen key.
func newScriptFormatSelect(desc *widget.Label, changed func(key string)) *widget.Select {
	return widget.NewSelect(scriptFormatNames(), func(name string) {
		for _, p := range scriptFormats() {
			if p.Name == name {
				if desc != nil {
					desc.SetText(p.Description)
				}
				if changed != nil {
					changed(p.Key)
				}
			}
		}
	})
}

// scriptFormatNames are the formats' names, in order.
func scriptFormatNames() []string {
	var names []string
	for _, p := range scriptFormats() {
		names = append(names, p.Name)
	}
	return names
}

// selectScriptFormat shows the format with key in s.
func selectScriptFormat(s *widget.Select, key string) { s.SetSelected(scriptFormat(key).Name) }

// selectedScriptFormat is the key of the format chosen in s.
func selectedScriptFormat(s *widget.Select) string {
	for _, p := range scriptFormats() {
		if p.Name == s.Selected {
			return p.Key
		}
	}
	return rules.ScreenplayPreset
}
