// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// The settings editor (LOOP.md, Joop 2026-11-22: "a nice UI in for
// lexington.toml … Even if you just want to use it for your CLI"):
// Lexington's settings as a form, a page per part of rules.TOMLConf.
// General: the standard language and the file extensions. Languages: one
// language at a time, each field Lexington's default greyed until
// overridden. Element Sets: one set at a time, each element with a
// sample. The Element sets hold no export options: those are flags of
// the command (and Accolade's Export dialog); the two that are settings,
// notecolors and pagination, are elements of a set, edited there.

// openSettingsEditor opens the editor of the settings file doc.
func openSettingsEditor(app *Application, parent fyne.Window, doc *tomlDoc) *tomlEditor {
	e := newTOMLEditor(app, doc)
	e.page("General", e.generalPage)
	e.page("Languages", e.languagesPage)
	e.page("Element Sets", e.setsPage)
	e.page("Title Pages", e.titlePagesPage)
	e.show(parent)
	return e
}

// knownLanguages are the languages to choose from: the file's (also in
// the older tables) and Lexington's, Hekaya (fixed) left out.
func (e *tomlEditor) knownLanguages() []string {
	set := map[string]bool{}
	raw := e.doc.raw()
	for _, t := range []string{"Languages", "Scenes", "Matchups", "Directions"} {
		if m, ok := raw[t].(map[string]any); ok {
			for k := range m {
				set[k] = true
			}
		}
	}
	for k := range rules.DefaultLanguages() {
		set[k] = true
	}
	delete(set, rules.Hekaya)
	return slices.Sorted(maps.Keys(set))
}

// inFile reports whether the file has the language (also in the older
// tables).
func (e *tomlEditor) inFile(lang string) bool {
	for _, t := range []string{"Languages", "Scenes", "Matchups", "Directions"} {
		if e.doc.has(t, lang) {
			return true
		}
	}
	return false
}

// generalPage: the standard language, the file extensions.
func (e *tomlEditor) generalPage() fyne.CanvasObject {
	conf := e.doc.conf()
	fields := rules.ConfFields()
	langs := e.knownLanguages()
	std := overridable(e.doc.has("Standard"), rules.DefaultConf().Standard, func() fyne.CanvasObject {
		s := widget.NewSelectEntry(langs)
		s.SetText(conf.Standard)
		s.Validator = func(v string) error {
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("a language")
			}
			return nil
		}
		s.OnChanged = func(v string) {
			if strings.TrimSpace(v) != "" {
				e.do(set(strings.TrimSpace(v), "Standard"))
			}
		}
		return s
	}, func() { e.doAndRedraw(set(rules.DefaultConf().Standard, "Standard")) }, func() { e.doAndRedraw(drop("Standard")) })

	own := slices.Sorted(maps.Keys(conf.Extensions))
	langChoice := append(slices.Clone(langs), rules.Hekaya)
	exts := tableEditor(own, func(k string) string { return conf.Extensions[k] }, "extension (no dot)", "language",
		func(k, v string) { e.do(set(v, "Extensions", k)) },
		func(old, now string) bool {
			now = strings.TrimPrefix(now, ".")
			if e.doc.has("Extensions", now) {
				return false
			}
			return e.do(drop("Extensions", old), set(e.doc.conf().Extensions[old], "Extensions", now))
		},
		func(k string) { e.doAndRedraw(drop("Extensions", k)) },
		func() {
			name := "ext"
			for i := 2; e.doc.has("Extensions", name); i++ {
				name = fmt.Sprintf("ext%d", i)
			}
			e.doAndRedraw(set(conf.StandardLanguage(), "Extensions", name))
		},
		func(k string, cur func() string) fyne.CanvasObject {
			s := widget.NewSelectEntry(langChoice)
			s.SetText(conf.Extensions[k])
			s.OnChanged = func(v string) {
				if strings.TrimSpace(v) != "" {
					e.do(set(strings.TrimSpace(v), "Extensions", cur()))
				}
			}
			return s
		})
	defaults := container.NewVBox()
	for _, k := range slices.Sorted(maps.Keys(rules.DefaultExtensions())) {
		if _, mine := conf.Extensions[k]; mine {
			continue
		}
		k, v := k, rules.DefaultExtensions()[k]
		b := widget.NewButtonWithIcon("Override", theme.ContentAddIcon(), func() { e.doAndRedraw(set(v, "Extensions", k)) })
		b.Importance = widget.LowImportance
		defaults.Add(container.NewBorder(nil, nil, nil, b, greyed("."+k+" → "+v+" (Lexington's)")))
	}
	return pagePane(nil,
		fieldBox("Standard", docOf(fields, "Standard"), std),
		fieldBox("Extensions", docOf(fields, "Extensions"), container.NewVBox(defaults, exts)),
	)
}

// legacyOps move a language out of the older tables into [Languages.x]
// (they are read over it), so an edit of it counts.
func (e *tomlEditor) legacyOps(lang string) []tomlOp {
	raw := e.doc.raw()
	var ops []tomlOp
	move := func(table, field string) {
		v, ok := mapAt(raw, []string{table, lang})
		if !ok {
			return
		}
		ops = append(ops, drop(table, lang))
		ops = append(ops, set(v, "Languages", lang, field))
	}
	move("Scenes", "Scenes")
	move("Matchups", "Matchup")
	move("Directions", "Direction")
	return ops
}

// langDo makes edits of a language, moving it out of the older tables
// first.
func (e *tomlEditor) langDo(lang string, redraw bool, ops ...tomlOp) bool {
	all := append(e.legacyOps(lang), ops...)
	if redraw {
		return e.doAndRedraw(all...)
	}
	return e.do(all...)
}

// languagesPage: one language, chosen at the top, field by field.
func (e *tomlEditor) languagesPage() fyne.CanvasObject {
	langs := e.knownLanguages()
	cur := e.sel["language"]
	if !slices.Contains(langs, cur) {
		cur = rules.DefaultConf().Standard
		for _, l := range langs { // the file's first, if any
			if e.inFile(l) {
				cur = l
				break
			}
		}
	}
	labels := make([]string, len(langs))
	for i, l := range langs {
		labels[i] = l
		if e.inFile(l) {
			labels[i] += " (in this file)"
		} else {
			labels[i] += " (Lexington's)"
		}
	}
	pick := widget.NewSelect(labels, nil)
	pick.SetSelected(labels[slices.Index(langs, cur)])
	pick.OnChanged = func(v string) {
		e.sel["language"] = langs[slices.Index(labels, v)]
		e.redraw()
	}
	fyne.SetAccessibleLabel(pick, "Language")
	add := widget.NewButtonWithIcon("Add…", theme.ContentAddIcon(), func() {
		askName(e.host, "Add a Language", "Code", "eo, de, zh-Hant…", func(code string) {
			if strings.ContainsAny(code, " .\"'[]=#") {
				e.changed(fmt.Errorf("%q cannot be a language's code", code))
				return
			}
			if e.doc.has("Languages", code) {
				e.sel["language"] = code
				e.redraw()
				return
			}
			scenes := rules.TOMLConf{}.Language(code).Scenes
			if scenes == nil {
				scenes = []string{}
			}
			err := e.doc.addTable([]string{"Languages", code}, map[string]any{"Scenes": scenes}, nil)
			e.changed(err)
			if err == nil {
				e.sel["language"] = code
			}
			e.redraw()
		})
	})
	remove := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() {
		dialog.ShowConfirm("Remove "+cur+"?", "Take "+cur+" out of this file? What Lexington knows of it is its default again.", func(ok bool) {
			if ok {
				e.doAndRedraw(drop("Languages", cur), drop("Scenes", cur), drop("Matchups", cur), drop("Directions", cur))
			}
		}, e.host)
	})
	if !e.inFile(cur) {
		remove.Disable()
	}
	var items []fyne.CanvasObject
	if cur == "ar" {
		items = append(items, pageNote("Hekaya files (.hekaya) are read with Hekaya's own rules, fixed; Arabic starts as the same and is yours to change."))
	}
	for _, f := range rules.LanguageFields() {
		items = append(items, fieldBox(f.Key, f.Doc, e.languageField(cur, f)))
	}
	return pagePane(pickRow("Language:", pick, add, remove), items...)
}

// languageField is one field of a language: its default greyed, or its
// control.
func (e *tomlEditor) languageField(lang string, f rules.Field) fyne.CanvasObject {
	path := []string{"Languages", lang, f.Key}
	legacy := map[string]string{"Scenes": "Scenes", "Matchup": "Matchups", "Direction": "Directions"}[f.Key]
	isSet := e.doc.has(path...) || legacy != "" && e.doc.has(legacy, lang)
	now := e.doc.conf().Language(lang)
	def := rules.TOMLConf{}.Language(lang)
	do := func(ops ...tomlOp) { e.langDo(lang, false, ops...) }
	reset := func() { e.langDo(lang, true, drop(path...)) }
	setList := func(l []string) { do(set(l, path...)) }
	list := func(cur, d []string, hint string) fyne.CanvasObject {
		return overridable(isSet, listText(d), func() fyne.CanvasObject {
			return listEditor(cur, hint, setList, e.redraw)
		}, func() {
			if d == nil {
				d = []string{}
			}
			e.langDo(lang, true, set(d, path...))
		}, reset)
	}
	switch f.Key {
	case "Scenes":
		return list(now.Scenes, def.Scenes, "INT")
	case "Headings": // the pattern builder (heading_builder.go)
		writeSo := func(p string) { e.langDo(lang, true, set(p, "Languages", lang, "NewHeading")) }
		var shown []string
		if len(now.Scenes) > 0 {
			ex, _ := exampleOf(lang, now, rules.SceneForm)
			shown = append(shown, "Fountain's own "+rules.SceneForm+" → "+ex)
		}
		for _, p := range def.Headings {
			ex, _ := exampleOf(lang, now, p)
			shown = append(shown, p+" → "+ex)
		}
		if len(def.Headings) == 0 {
			shown = append(shown, "no patterns of its own")
		}
		return overridable(isSet, strings.Join(shown, "\n"), func() fyne.CanvasObject {
			return headingBuilder(lang, now, now.Headings, setList, e.redraw, writeSo)
		}, func() {
			d := def.Headings
			if d == nil {
				d = []string{}
			}
			e.langDo(lang, true, set(d, path...))
		}, reset)
	case "NewHeading":
		form := now.HeadingForm()
		defForm := def.HeadingForm()
		ex, _ := exampleOf(lang, now, defForm)
		return overridable(isSet, defForm+" → "+ex, func() fyne.CanvasObject {
			choices := []string{}
			if len(now.Scenes) > 0 {
				choices = append(choices, rules.SceneForm)
			}
			for _, p := range now.Headings {
				if !slices.Contains(choices, p) {
					choices = append(choices, p)
				}
			}
			pick := widget.NewSelectEntry(choices)
			pick.SetText(now.NewHeading)
			pick.SetPlaceHolder(form)
			example := widget.NewLabel(exampleText(lang, now, form))
			example.Wrapping = fyne.TextWrapWord
			pick.OnChanged = func(v string) {
				if strings.TrimSpace(v) == "" {
					return
				}
				if e.langDo(lang, false, set(v, path...)) {
					example.SetText(exampleText(lang, e.doc.conf().Language(lang), v))
				}
			}
			fyne.SetAccessibleLabel(pick, "New scenes are written by")
			return container.NewVBox(pick, example)
		}, func() { e.langDo(lang, true, set(form, path...)) }, reset)
	case "Settings":
		return list(now.Settings, def.Settings, "INNEN")
	case "Times":
		return list(now.Times, def.Times, "TAG")
	case "Extensions":
		return list(now.Extensions, def.Extensions, "V.O.")
	case "TransitionEnds":
		return list(now.TransitionEnds, def.TransitionEnds, " TO:")
	case "Transitions":
		return list(now.Transitions, def.Transitions, "FADE OUT.")
	case "Matchup":
		roles := []string{"Inside", "Outside", "Both", "Both, the other way", "Establishing"}
		return overridable(isSet, listText(def.Matchup), func() fyne.CanvasObject {
			cur := make([]string, len(roles))
			copy(cur, now.Matchup)
			grid := container.NewGridWithColumns(2)
			for i, r := range roles {
				i := i
				en := widget.NewEntry()
				en.SetText(cur[i])
				en.OnChanged = func(v string) {
					cur[i] = v
					setList(cur)
				}
				grid.Add(widget.NewLabel(r))
				grid.Add(en)
			}
			return grid
		}, func() {
			m := def.Matchup
			if m == nil {
				m = make([]string, len(roles))
			}
			e.langDo(lang, true, set(m, path...))
		}, reset)
	case "Direction":
		return overridable(isSet, def.Direction, func() fyne.CanvasObject {
			r := widget.NewRadioGroup([]string{"ltr", "rtl"}, func(v string) {
				if v != "" {
					do(set(v, path...))
				}
			})
			r.Horizontal = true
			r.Selected = strings.ToLower(now.Direction)
			return r
		}, func() { e.langDo(lang, true, set(def.Direction, path...)) }, reset)
	case "Cues":
		return overridable(isSet, strings.Join(def.Cues, ", "), func() fyne.CanvasObject {
			box := container.NewVBox()
			checks := map[string]*widget.Check{}
			for _, c := range rules.CueRuleFields() {
				ch := widget.NewCheck(c.Key+": "+c.Doc, nil)
				ch.Checked = now.Has(c.Key)
				if c.Key == rules.CueForced {
					ch.Checked = true
					ch.Disable()
				}
				checks[c.Key] = ch
				box.Add(ch)
			}
			for _, ch := range checks {
				ch.OnChanged = func(bool) {
					var cues []string
					for _, r := range rules.CueRules {
						if checks[r].Checked {
							cues = append(cues, r)
						}
					}
					setList(cues)
				}
			}
			return box
		}, func() { e.langDo(lang, true, set(def.Cues, path...)) }, reset)
	case "Shape":
		d := def.Shape
		return overridable(isSet, fmt.Sprintf("at most %d words, %d characters (0: any), none of %q", d.Words, d.Chars, d.Ends), func() fyne.CanvasObject {
			words := intEntry(now.Shape.Words, 99, "0", func(n int) { do(set(int64(n), append(path, "Words")...)) })
			chars := intEntry(now.Shape.Chars, 999, "0", func(n int) { do(set(int64(n), append(path, "Chars")...)) })
			ends := widget.NewEntry()
			ends.SetText(now.Shape.Ends)
			ends.OnChanged = func(v string) { do(set(v, append(path, "Ends")...)) }
			return container.NewGridWithColumns(2,
				widget.NewLabel("Words"), words, widget.NewLabel("Characters"), chars, widget.NewLabel("Sentence ends"), ends)
		}, func() {
			e.langDo(lang, true, set(map[string]any{"Words": int64(d.Words), "Chars": int64(d.Chars), "Ends": d.Ends}, path...))
		}, reset)
	case "Repeat":
		return overridable(isSet, fmt.Sprint(def.Repeat), func() fyne.CanvasObject {
			return intEntry(now.Repeat, 99, "2", func(n int) {
				if n == 0 {
					return
				}
				do(set(int64(n), path...))
			})
		}, func() { e.langDo(lang, true, set(int64(max(def.Repeat, 1)), path...)) }, reset)
	case "TransitionMarks":
		return overridable(isSet, def.TransitionMarks, func() fyne.CanvasObject {
			en := widget.NewEntry()
			en.SetText(now.TransitionMarks)
			en.OnChanged = func(v string) { do(set(v, path...)) }
			return en
		}, func() { e.langDo(lang, true, set(def.TransitionMarks, path...)) }, reset)
	case "TitleKeys":
		keys := slices.Sorted(maps.Keys(now.TitleKeys))
		var shown []string
		for _, k := range slices.Sorted(maps.Keys(def.TitleKeys)) {
			shown = append(shown, k+" = "+def.TitleKeys[k])
		}
		return overridable(isSet, strings.Join(shown, ", "), func() fyne.CanvasObject {
			return tableEditor(keys, func(k string) string { return now.TitleKeys[k] }, "the language's key", "Title, Author, Credit…",
				func(k, v string) { do(set(v, append(path, k)...)) },
				func(old, nw string) bool {
					if _, taken := now.TitleKeys[nw]; taken {
						return false
					}
					ok := e.langDo(lang, false, drop(append(path, old)...), set(now.TitleKeys[old], append(path, nw)...))
					if ok {
						now.TitleKeys[nw] = now.TitleKeys[old]
						delete(now.TitleKeys, old)
					}
					return ok
				},
				func(k string) { e.langDo(lang, true, drop(append(path, k)...)) },
				func() {
					name := "KEY"
					for i := 2; now.TitleKeys[name] != ""; i++ {
						name = fmt.Sprintf("KEY%d", i)
					}
					e.langDo(lang, true, set("Title", append(path, name)...))
				}, nil)
		}, func() {
			m := map[string]any{}
			for k, v := range def.TitleKeys {
				m[k] = v
			}
			e.langDo(lang, true, set(m, path...))
		}, reset)
	case "TitleCentred": // the keys centred under the title (the rest: the foot), in order
		return list(now.TitleCentred, def.Centred(), "Title")
	}
	return widget.NewLabel("")
}

// setsPage: the element sets, one at a time.
func (e *tomlEditor) setsPage() fyne.CanvasObject {
	conf := e.doc.conf()
	raw, _ := e.doc.raw()["Elements"].(map[string]any)
	mine := slices.Sorted(maps.Keys(raw))
	var keys, labels []string
	for _, k := range mine {
		keys, labels = append(keys, k), append(labels, k+" (in this file)")
	}
	for _, p := range rules.Presets() {
		if !slices.Contains(keys, p.Key) {
			keys, labels = append(keys, p.Key), append(labels, p.Key+": "+p.Name+" (Lexington's)")
		}
	}
	cur := e.sel["set"]
	if !slices.Contains(keys, cur) {
		cur = keys[0]
	}
	pick := widget.NewSelect(labels, nil)
	pick.SetSelected(labels[slices.Index(keys, cur)])
	pick.OnChanged = func(v string) {
		e.sel["set"] = keys[slices.Index(labels, v)]
		e.redraw()
	}
	fyne.SetAccessibleLabel(pick, "Element set")
	setOf := func(k string) rules.Set {
		if s, ok := conf.Elements[k]; ok && slices.Contains(mine, k) {
			return s
		}
		p, _ := rules.GetPreset(k)
		return p.Elements
	}
	copySet := func(name string, from rules.Set) {
		var err error
		for _, f := range rules.ElementKeys() {
			if el, ok := from[f.Key]; ok {
				if err = e.doc.addTable([]string{"Elements", name, f.Key}, formatMap(el), formatOrder); err != nil {
					break
				}
			}
		}
		e.changed(err)
		if err == nil {
			e.sel["set"] = name
		}
		e.redraw()
	}
	add := widget.NewButtonWithIcon("Add…", theme.ContentAddIcon(), func() {
		name := widget.NewEntry()
		name.SetPlaceHolder("mine")
		from := widget.NewSelect(labels, nil)
		from.SetSelected(labels[slices.Index(keys, cur)])
		dialog.ShowForm("Add an Element Set", "Add", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Name", name), widget.NewFormItem("Start from", from)}, func(ok bool) {
			n := strings.TrimSpace(name.Text)
			if !ok || n == "" {
				return
			}
			if strings.ContainsAny(n, " .\"'[]=#") || slices.Contains(mine, n) {
				e.changed(fmt.Errorf("%q cannot be the new set's name", n))
				return
			}
			copySet(n, setOf(keys[slices.Index(labels, from.Selected)]))
		}, e.host)
	})
	remove := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() {
		dialog.ShowConfirm("Remove "+cur+"?", "Take the set "+cur+" out of this file?", func(ok bool) {
			if ok {
				e.doAndRedraw(drop("Elements", cur))
			}
		}, e.host)
	})
	top := pickRow("Set:", pick, add, remove)
	about := pageNote("A set here is whole: an element it has no rule for is not printed. A set named like a preset takes its place. " +
		"A format of your own is easier as a format file (Preferences > Script > New Format from This…), which changes only what differs.")
	if !slices.Contains(mine, cur) {
		remove.Disable()
		p, _ := rules.GetPreset(cur)
		over := widget.NewButtonWithIcon("Override", theme.DocumentCreateIcon(), func() { copySet(cur, p.Elements) })
		sample := newFormatSample(func() (rules.Set, bool) { return p.Elements, true }, p.Page)
		e.sample = sample
		form := container.NewVScroll(container.NewPadded(container.NewVBox(about,
			fieldBox(p.Name, p.Description+" Lexington's own: Override copies it into this file, to change; it then takes the place of Lexington's.", container.NewHBox(over)))))
		return container.NewBorder(container.NewPadded(top), nil, nil, nil,
			container.New(&besideOrUnder{side: 400, under: 280}, form, sample.object()))
	}
	return e.elementsPane("set-"+cur, top, about, []string{"Elements", cur}, conf.Elements[cur], nil, "", func() (rules.Set, bool) {
		c, err := rules.Decode([]byte(e.doc.text))
		if err != nil {
			return nil, false
		}
		return c.Elements[cur], true
	}, "")
}
