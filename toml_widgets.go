package main

import (
	"errors"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// The pieces the settings and format editors are made of: a setting with
// what it does under its name, its default shown greyed until overridden,
// lists and tables to add to and take from, numbers checked as typed.

// fieldBox is a setting: its name, what it does (from Lexington's own
// descriptions, rules.Describe), and its control.
func fieldBox(name, doc string, control fyne.CanvasObject) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	if doc == "" {
		return container.NewVBox(title, control)
	}
	about := widget.NewLabelWithStyle(doc, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	about.Wrapping = fyne.TextWrapWord
	return container.NewVBox(title, about, control)
}

// docOf is a field's description among fields.
func docOf(fields []rules.Field, key string) string {
	f, _ := rules.Describe(fields, key)
	return f.Doc
}

// greyed is a default's value, shown as not set.
func greyed(text string) *widget.Label {
	if strings.TrimSpace(text) == "" {
		text = "(none)"
	}
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapWord
	l.Importance = widget.LowImportance
	return l
}

// pageNote is a word of explanation on a page.
func pageNote(text string) *widget.Label {
	l := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	l.Wrapping = fyne.TextWrapWord
	return l
}

// overridable is a setting with a default: while the file does not set
// it, the default greyed and Override (which sets it to the default, to
// change); once set, its control and Use the Default (which takes it out
// of the file).
func overridable(isSet bool, defaultText string, control func() fyne.CanvasObject, override, reset func()) fyne.CanvasObject {
	if !isSet {
		b := widget.NewButtonWithIcon("Override", theme.ContentAddIcon(), override)
		b.Importance = widget.LowImportance
		return container.NewBorder(nil, nil, nil, b, container.NewHBox(widget.NewLabel("Default:"), greyed(defaultText)))
	}
	b := widget.NewButtonWithIcon("Use the Default", theme.ContentUndoIcon(), reset)
	b.Importance = widget.LowImportance
	return container.NewVBox(control(), container.NewHBox(b))
}

// listText is a list as the greyed default shows it.
func listText(list []string) string {
	if list == nil {
		return ""
	}
	if len(list) == 0 {
		return "(an empty list)"
	}
	q := make([]string, len(list))
	for i, s := range list {
		q[i] = strconv.Quote(s)
	}
	return strings.Join(q, ", ")
}

// listEditor edits a list of text: a line each, with ✕; Add adds one.
// change gets the whole list after every change; redraw is for a line
// added or taken out.
func listEditor(list []string, placeholder string, change func([]string), redraw func()) fyne.CanvasObject {
	list = append([]string{}, list...)
	box := container.NewVBox()
	for i := range list {
		i := i
		en := widget.NewEntry()
		en.SetText(list[i])
		en.SetPlaceHolder(placeholder)
		en.OnChanged = func(s string) {
			list[i] = s
			change(list)
		}
		rm := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() {
			list = append(list[:i:i], list[i+1:]...)
			change(list)
			redraw()
		})
		fyne.SetAccessibleLabel(rm, "Remove "+list[i])
		box.Add(container.NewBorder(nil, nil, nil, rm, en))
	}
	add := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		list = append(list, "")
		change(list)
		redraw()
	})
	add.Importance = widget.LowImportance
	box.Add(container.NewHBox(add))
	return box
}

// tableEditor edits keys to text (title page keys, file extensions): a
// line each, key and value, with ✕; Add adds one. set is a key's value
// set, rename a key renamed (its value with it), remove one taken out.
func tableEditor(keys []string, value func(string) string, keyHint, valueHint string,
	setValue func(k, v string), rename func(old, now string) bool, remove func(k string), add func(), valueControl func(k string, key func() string) fyne.CanvasObject) fyne.CanvasObject {
	box := container.NewVBox()
	for _, k := range keys {
		k := k
		key := widget.NewEntry()
		key.SetText(k)
		key.SetPlaceHolder(keyHint)
		cur := k
		key.OnChanged = func(s string) {
			if s != cur && strings.TrimSpace(s) != "" && !strings.ContainsAny(s, "\n") && rename(cur, strings.TrimSpace(s)) {
				cur = strings.TrimSpace(s)
			}
		}
		var val fyne.CanvasObject
		if valueControl != nil {
			val = valueControl(k, func() string { return cur })
		} else {
			en := widget.NewEntry()
			en.SetText(value(k))
			en.SetPlaceHolder(valueHint)
			en.OnChanged = func(s string) { setValue(cur, s) }
			val = en
		}
		rm := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() { remove(cur) })
		fyne.SetAccessibleLabel(rm, "Remove "+k)
		box.Add(container.NewBorder(nil, nil, nil, rm, container.NewGridWithColumns(2, key, val)))
	}
	b := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), add)
	b.Importance = widget.LowImportance
	box.Add(container.NewHBox(b))
	return box
}

// numberEntry is a decimal number's entry: change gets it while what is
// typed is one in [min, max]; the entry says when it is not.
func numberEntry(v float64, min, max float64, placeholder string, change func(float64)) *widget.Entry {
	en := widget.NewEntry()
	if v != 0 {
		en.SetText(strconv.FormatFloat(v, 'f', -1, 64))
	}
	en.SetPlaceHolder(placeholder)
	en.Validator = func(s string) error {
		_, err := parseNumber(s, min, max)
		return err
	}
	en.OnChanged = func(s string) {
		if n, err := parseNumber(s, min, max); err == nil {
			change(n)
		}
	}
	return en
}

// parseNumber reads a number in [min, max] ("" is 0; a comma for the dot).
func parseNumber(s string, min, max float64) (float64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, errors.New("a number, like 1.5")
	}
	if n < min || n > max {
		return 0, errors.New("from " + strconv.FormatFloat(min, 'f', -1, 64) + " to " + strconv.FormatFloat(max, 'f', -1, 64))
	}
	return n, nil
}

// intEntry is a whole number's entry, as numberEntry.
func intEntry(v int, max int, placeholder string, change func(int)) *widget.Entry {
	en := widget.NewEntry()
	en.SetText(strconv.Itoa(v))
	en.SetPlaceHolder(placeholder)
	en.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n < 0 || n > max {
			return errors.New("a whole number from 0 to " + strconv.Itoa(max))
		}
		return nil
	}
	en.OnChanged = func(s string) {
		if en.Validator(s) == nil {
			n, _ := strconv.Atoi(strings.TrimSpace(s))
			change(n)
		}
	}
	return en
}

// pagePane is a page of the form: its settings one under the other,
// scrolling.
func pagePane(top fyne.CanvasObject, items ...fyne.CanvasObject) fyne.CanvasObject {
	box := container.NewVBox()
	for i, it := range items {
		if i > 0 {
			box.Add(widget.NewSeparator())
		}
		box.Add(it)
	}
	scroll := container.NewVScroll(container.NewPadded(box))
	if top == nil {
		return scroll
	}
	return container.NewBorder(container.NewPadded(top), nil, nil, nil, scroll)
}

// pickRow is a row choosing what a page shows (a language, an element),
// with the buttons beside it.
func pickRow(label string, pick *widget.Select, buttons ...fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil, widget.NewLabel(label), container.NewHBox(buttons...), pick)
}
