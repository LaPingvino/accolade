package main

import (
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// The script format is one of Lexington's presets (screenplay, stage
// play, radio, ...): how the preview and the PDF and HTML exports lay
// the script out. It is the "script-format" setting, the preset's key.

// scriptFormatElements are the element rules of the format with key.
func scriptFormatElements(key string) rules.Set {
	if p, ok := rules.GetPreset(key); ok {
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
	presets := rules.Presets()
	names := make([]string, len(presets))
	for i, p := range presets {
		names[i] = p.Name
	}
	return widget.NewSelect(names, func(name string) {
		for _, p := range presets {
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

// selectScriptFormat shows the format with key in s.
func selectScriptFormat(s *widget.Select, key string) {
	p, ok := rules.GetPreset(key)
	if !ok {
		p, _ = rules.GetPreset(rules.ScreenplayPreset)
	}
	s.SetSelected(p.Name)
}

// selectedScriptFormat is the key of the format chosen in s.
func selectedScriptFormat(s *widget.Select) string {
	for _, p := range rules.Presets() {
		if p.Name == s.Selected {
			return p.Key
		}
	}
	return rules.ScreenplayPreset
}
