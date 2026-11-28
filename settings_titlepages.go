// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"image"
	"maps"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/rules"
	"github.com/LaPingvino/lexington/titlepage"
)

// The settings editor's Title Pages page: the title page templates of
// lexington.toml ([TitlePages.name], a [[TitlePages.name.Keys]] block a
// key), one at a time, beside a sample title page printed with it as
// it is now. Lexington's presets (standard, bold, poster) are shown to
// override: copied into the file, to change. Every control is one edit of
// the text, as on the other pages; the fields' descriptions are
// Lexington's own (rules.TitlePageFields, rules.TitleKeyFields).

// titleKeyOrder is the order a key's fields are written in.
var titleKeyOrder = []string{"Key", "Place", "X", "Y", "Align", "Font", "Size", "Style", "Caps", "Before", "Picture", "Height", "Hide"}

// titleKeyMap is a template's key as the TOML has it: its fields set.
func titleKeyMap(k rules.TitleKey) map[string]any {
	m := map[string]any{"Key": k.Key}
	put := func(name string, v any, zero bool) {
		if !zero {
			m[name] = v
		}
	}
	put("Place", k.Place, k.Place == "")
	put("X", k.X, k.X == 0)
	put("Y", k.Y, k.Y == 0)
	put("Align", k.Align, k.Align == "")
	put("Font", k.Font, k.Font == "")
	put("Size", k.Size, k.Size == 0)
	put("Style", k.Style, k.Style == "")
	put("Caps", k.Caps, !k.Caps)
	put("Before", k.Before, k.Before == 0)
	put("Picture", k.Picture, !k.Picture)
	put("Height", k.Height, k.Height == 0)
	put("Hide", k.Hide, !k.Hide)
	return m
}

// titlePagesPage: one template, its fields and keys, and its sample.
func (e *tomlEditor) titlePagesPage() fyne.CanvasObject {
	conf := e.doc.conf()
	raw, _ := e.doc.raw()["TitlePages"].(map[string]any)
	mine := slices.Sorted(maps.Keys(raw))
	presets := rules.TitlePagePresets()
	var keys, labels []string
	for _, k := range mine {
		keys, labels = append(keys, k), append(labels, k+" (in this file)")
	}
	for _, k := range (rules.TOMLConf{}).TitlePageNames() {
		if !slices.Contains(keys, k) {
			keys, labels = append(keys, k), append(labels, k+": "+presets[k].Name+" (Lexington's)")
		}
	}
	cur := e.sel["titlepage"]
	if !slices.Contains(keys, cur) {
		cur = keys[0]
	}
	pick := widget.NewSelect(labels, nil)
	pick.SetSelected(labels[slices.Index(keys, cur)])
	pick.OnChanged = func(v string) {
		e.sel["titlepage"] = keys[slices.Index(labels, v)]
		e.redraw()
	}
	fyne.SetAccessibleLabel(pick, "Title page template")
	templateOf := func(k string) rules.TitlePage {
		if t, ok := conf.TitlePages[k]; ok && slices.Contains(mine, k) {
			return t
		}
		return presets[k]
	}
	copyTemplate := func(name string, from rules.TitlePage) {
		top := map[string]any{}
		if from.Name != "" {
			top["Name"] = from.Name
		}
		if from.Description != "" {
			top["Description"] = from.Description
		}
		if from.Centre != 0 {
			top["Centre"] = from.Centre
		}
		err := e.doc.addTable([]string{"TitlePages", name}, top, []string{"Name", "Description", "Centre"})
		for _, k := range from.Keys {
			if err != nil {
				break
			}
			err = e.addTitleKeyBlock(name, titleKeyMap(k))
		}
		e.changed(err)
		if err == nil {
			e.sel["titlepage"] = name
		}
		e.redraw()
	}
	add := widget.NewButtonWithIcon("Add…", theme.ContentAddIcon(), func() {
		name := widget.NewEntry()
		name.SetPlaceHolder("mine")
		from := widget.NewSelect(labels, nil)
		from.SetSelected(labels[slices.Index(keys, cur)])
		dialog.ShowForm("Add a Title Page Template", "Add", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Name", name), widget.NewFormItem("Start from", from)}, func(ok bool) {
			n := strings.TrimSpace(name.Text)
			if !ok || n == "" {
				return
			}
			if strings.ContainsAny(n, " .\"'[]=#") || slices.Contains(mine, n) {
				e.changed(fmt.Errorf("%q cannot be the new template's name", n))
				return
			}
			t := templateOf(keys[slices.Index(labels, from.Selected)])
			t.Name = n
			copyTemplate(n, t)
		}, e.host)
	})
	remove := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() {
		dialog.ShowConfirm("Remove "+cur+"?", "Take the title page template "+cur+" out of this file?", func(ok bool) {
			if ok {
				e.changed(e.removeTitleTemplate(cur))
				e.redraw()
			}
		}, e.host)
	})
	top := pickRow("Template:", pick, add, remove)
	about := pageNote("Where each title page key goes and how it looks, by its key: the order the keys are written in does not matter. " +
		"A key no template names goes where its language's TitleCentred says (centred, else at the foot). " +
		"Choose the template in Page Setup, or in the Title Page tab while writing the title page.")
	e.sample = nil // (the element sample's: none here)
	sample := newTitleSample()
	sample.show(conf, cur)
	beside := func(form fyne.CanvasObject) fyne.CanvasObject {
		return container.New(&besideOrUnder{side: 400, under: 280}, form, sample.object())
	}
	if !slices.Contains(mine, cur) {
		remove.Disable()
		p := presets[cur]
		over := widget.NewButtonWithIcon("Override", theme.DocumentCreateIcon(), func() { copyTemplate(cur, p) })
		form := container.NewVScroll(container.NewPadded(container.NewVBox(about,
			fieldBox(p.Name, p.Description+" Lexington's own: Override copies it into this file, to change; it then takes the place of Lexington's.",
				container.NewHBox(over)))))
		return container.NewBorder(container.NewPadded(top), nil, nil, nil, beside(form))
	}
	t := conf.TitlePages[cur]
	fields := rules.TitlePageFields()
	redrawSample := func() { sample.show(e.doc.conf(), cur) }
	setTop := func(field string, v any, empty bool) {
		var ok bool
		if empty {
			ok = e.do(drop("TitlePages", cur, field))
		} else {
			ok = e.do(set(v, "TitlePages", cur, field))
		}
		if ok {
			redrawSample()
		}
	}
	name := widget.NewEntry()
	name.SetText(t.Name)
	name.SetPlaceHolder(cur)
	name.OnChanged = func(v string) { setTop("Name", strings.TrimSpace(v), strings.TrimSpace(v) == "") }
	desc := widget.NewEntry()
	desc.SetText(t.Description)
	desc.OnChanged = func(v string) { setTop("Description", strings.TrimSpace(v), strings.TrimSpace(v) == "") }
	centre := numberEntry(t.Centre, 0, 10, "4", func(n float64) { setTop("Centre", n, n == 0) })
	keysBox := container.NewVBox()
	for i, k := range t.Keys {
		keysBox.Add(e.titleKeyCard(cur, i, len(t.Keys), k, redrawSample))
	}
	addKey := widget.NewButtonWithIcon("Add a Key", theme.ContentAddIcon(), func() {
		e.changed(e.addTitleKeyBlock(cur, map[string]any{"Key": "Logline", "Place": rules.PlaceCentre}))
		e.redraw()
	})
	form := pagePane(nil, about,
		fieldBox("Name", docOf(fields, "Name"), name),
		fieldBox("Description", docOf(fields, "Description"), desc),
		fieldBox("Centre", docOf(fields, "Centre"), centre),
		fieldBox("Keys", docOf(fields, "Keys"), container.NewVBox(keysBox, container.NewHBox(addKey))))
	return container.NewBorder(container.NewPadded(top), nil, nil, nil, beside(form))
}

// removeTitleTemplate takes the template name out of the file: its keys'
// [[blocks]] one by one, then its table (a path's drop leaves blocks).
func (e *tomlEditor) removeTitleTemplate(name string) error {
	path := keysPath(name)
	for n := len(blocksAt(e.doc.raw(), path)); n > 0; n-- {
		i := n - 1
		err := e.doc.editText(func(s string) (string, error) { return dropBlock(s, path, i) },
			func(m map[string]any) map[string]any { return withBlocksAt(m, path, slices.Delete(blocksAt(m, path), i, i+1)) })
		if err != nil {
			return err
		}
	}
	if !e.doc.has("TitlePages", name) {
		return nil
	}
	if all, _ := e.doc.raw()["TitlePages"].(map[string]any); len(all) == 1 { // the last: [TitlePages] goes too
		return e.doc.edit(drop("TitlePages"))
	}
	return e.doc.edit(drop("TitlePages", name))
}

// keysPath is the [[TitlePages.name.Keys]] blocks' path.
func keysPath(name string) []string { return []string{"TitlePages", name, "Keys"} }

// addTitleKeyBlock adds a key's block to the template name, at its end.
func (e *tomlEditor) addTitleKeyBlock(name string, m map[string]any) error {
	before, err := tomlMap(e.doc.text)
	if err != nil {
		return err
	}
	path := keysPath(name)
	return e.doc.keep(addBlock(e.doc.text, path, m, titleKeyOrder), withBlocksAt(before, path, append(blocksAt(before, path), m)))
}

// titleKeyCard is the card of the template's i-th key (of n): its fields,
// up, down, remove.
func (e *tomlEditor) titleKeyCard(name string, i, n int, k rules.TitleKey, changed func()) fyne.CanvasObject {
	path := keysPath(name)
	setField := func(field string, v any, empty bool) {
		value := ""
		if !empty {
			value = tomlValue(v)
		}
		err := e.doc.editText(func(s string) (string, error) { return setInBlock(s, path, i, field, value) },
			func(m map[string]any) map[string]any {
				b := blocksAt(m, path)
				if empty {
					delete(b[i], field)
				} else {
					b[i][field] = v
				}
				return withBlocksAt(m, path, b)
			})
		e.changed(err)
		if err == nil {
			changed()
		}
	}
	tip := func(o fyne.CanvasObject, field string) fyne.CanvasObject {
		l := widget.NewLabel(field + ":")
		return container.NewBorder(nil, nil, l, nil, o)
	}
	key := widget.NewEntry()
	key.SetText(k.Key)
	key.SetPlaceHolder("Logline, or * for the rest")
	key.OnChanged = func(v string) {
		if v = strings.TrimSpace(v); v != "" {
			setField("Key", v, false)
		}
	}
	fyne.SetAccessibleLabel(key, "Key")
	choice := func(field, now string, options []string, labels []string) *widget.Select {
		s := widget.NewSelect(labels, func(l string) {
			v := options[slices.Index(labels, l)]
			setField(field, v, v == "")
		})
		at := slices.Index(options, now)
		if at < 0 {
			at = 0
		}
		s.Selected = labels[at]
		fyne.SetAccessibleLabel(s, field)
		return s
	}
	places := append([]string{""}, rules.Places...)
	place := choice("Place", strings.ToLower(k.Place), places,
		[]string{"The language's", "Centre", "Top", "Foot, left", "Foot, right", "Free (X, Y)"})
	align := choice("Align", strings.ToUpper(k.Align), []string{"", "L", "C", "R"}, []string{"The place's", "Left", "Centred", "Right"})
	num := func(field string, v, min, max float64, hint string) fyne.CanvasObject {
		en := numberEntry(v, min, max, hint, func(x float64) { setField(field, x, x == 0) })
		fyne.SetAccessibleLabel(en, field)
		return tip(en, field)
	}
	style := widget.NewEntry()
	style.SetText(k.Style)
	style.SetPlaceHolder("b i u")
	style.Validator = func(s string) error {
		if strings.Trim(strings.ToLower(s), "biu ") != "" {
			return errors.New("b, i and u")
		}
		return nil
	}
	style.OnChanged = func(v string) {
		if style.Validator(v) == nil {
			v = strings.ReplaceAll(strings.ToLower(v), " ", "")
			setField("Style", v, v == "")
		}
	}
	font := widget.NewSelectEntry([]string{"CourierPrime", "Helvetica", "Times"})
	font.SetText(k.Font)
	font.SetPlaceHolder("The element set's")
	font.OnChanged = func(v string) { setField("Font", strings.TrimSpace(v), strings.TrimSpace(v) == "") }
	check := func(field, label string, on bool) *widget.Check {
		c := widget.NewCheck(label, func(b bool) { setField(field, b, !b) })
		c.Checked = on
		return c
	}
	move := func(to int) func() {
		return func() {
			err := e.doc.editText(func(s string) (string, error) { return moveBlock(s, path, i, to) },
				func(m map[string]any) map[string]any {
					b := blocksAt(m, path)
					x := b[i]
					b = slices.Delete(b, i, i+1)
					return withBlocksAt(m, path, slices.Insert(b, to, x))
				})
			e.changed(err)
			e.redraw()
		}
	}
	up := widget.NewButtonWithIcon("", theme.MoveUpIcon(), move(i-1))
	fyne.SetAccessibleLabel(up, "Move up")
	down := widget.NewButtonWithIcon("", theme.MoveDownIcon(), move(i+1))
	fyne.SetAccessibleLabel(down, "Move down")
	if i == 0 {
		up.Disable()
	}
	if i == n-1 {
		down.Disable()
	}
	rm := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		err := e.doc.editText(func(s string) (string, error) { return dropBlock(s, path, i) },
			func(m map[string]any) map[string]any {
				return withBlocksAt(m, path, slices.Delete(blocksAt(m, path), i, i+1))
			})
		e.changed(err)
		e.redraw()
	})
	fyne.SetAccessibleLabel(rm, "Remove the key")
	head := container.NewBorder(nil, nil, widget.NewLabelWithStyle(strconv.Itoa(i+1)+".", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(up, down, rm), key)
	cols := 2
	if c := e.host.Canvas().Size(); c.Width > 0 && c.Width < 600 { // a phone: one under the other
		cols = 1
	}
	grid := container.NewGridWithColumns(cols,
		tip(place, "Place"), tip(align, "Align"),
		num("Size", k.Size, 0, 72, "12"), tip(style, "Style"),
		num("Before", k.Before, 0, 40, "0 lines"), tip(font, "Font"),
		num("X", k.X, -20, 20, "free: in"), num("Y", k.Y, -20, 20, "free: in"),
		num("Height", k.Height, 0, 10, "picture: 2 in"), container.NewHBox(check("Caps", "Capitals", k.Caps),
			check("Picture", "Picture", k.Picture), check("Hide", "Hide", k.Hide)))
	return widget.NewCard("", "", container.NewVBox(head, grid))
}

// blocksAt are the [[path]] tables of m, copied.
func blocksAt(m map[string]any, path []string) []map[string]any {
	v, _ := mapAt(m, path)
	b, _ := v.([]map[string]any)
	out := make([]map[string]any, len(b))
	for i, x := range b {
		out[i] = maps.Clone(x)
	}
	return out
}

// withBlocksAt is m with the [[path]] tables b (none: the key gone).
func withBlocksAt(m map[string]any, path []string, b []map[string]any) map[string]any {
	if len(b) == 0 {
		if _, ok := mapAt(m, path); !ok {
			return m
		}
		return withAt(m, path, nil)
	}
	return withAt(m, path, b)
}

// The sample: a title page printed with the template.

// titleSampleText is the title page the sample prints: Fountain's keys,
// and a logline (the presets place one).
const titleSampleText = `Title: The Lighthouse Keeper
Credit: Written by
Author: Ada Writer
Source: Based on the story by A. N. Other
Logline: A keeper who never sleeps finds a ship that never sails.
Draft date: 23 November 2026
Contact:
    Ada Writer
    ada@example.com
`

// titleSample is the page's sample title page.
type titleSample struct {
	img  *canvas.Image
	note *widget.Label
	box  fyne.CanvasObject
}

func newTitleSample() *titleSample {
	s := &titleSample{img: canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1))), note: widget.NewLabel("")}
	s.img.FillMode = canvas.ImageFillContain
	s.img.SetMinSize(fyne.NewSize(255, 330))
	s.note.Wrapping = fyne.TextWrapWord
	s.note.Importance = widget.LowImportance
	s.box = container.NewBorder(nil, s.note, nil, nil, s.img)
	return s
}

func (s *titleSample) object() fyne.CanvasObject { return s.box }

// show prints the sample with the template key of conf.
func (s *titleSample) show(conf rules.TOMLConf, key string) {
	text := titleSampleText
	if t, ok := conf.TitlePage(key); ok {
		for _, k := range t.Keys {
			if k.Picture && k.Key != rules.OtherKeys {
				text += k.Key + ": ![Key art](key-art.png)\n"
				break
			}
		}
	}
	img, err := titleSampleImage(conf, key, text, nil)
	if err != nil {
		s.note.SetText(err.Error())
		return
	}
	s.note.SetText("A sample title page (a picture key prints as a box with its file's name).")
	s.img.Image = img
	s.img.Refresh()
}

// titleSampleImage is text's title page printed with the template key
// of conf (in English, on Letter), its first page as an image; the
// computer's fonts are asked for what Courier Prime lacks in English too.
func titleSampleImage(conf rules.TOMLConf, key, text string, pictures map[string][]byte) (image.Image, error) {
	const lang = "en"
	tpl, ok := titlepage.For(conf, key, lang)
	if !ok {
		return nil, fmt.Errorf("no title page template %q", key)
	}
	s := fountain.ParseIn(conf.Language(lang), strings.NewReader(strings.TrimRight(text, "\n")+"\n\n.THE SCRIPT\n"))
	var d pdf.Drawing
	pw := &pdf.PDFWriter{Elements: rules.Default, TitlePage: &tpl, TitlePictures: pictures, Record: &d, Language: fontLanguage(lang)}
	if _, err := pw.Render(s); err != nil {
		return nil, err
	}
	if len(d.Pages) == 0 {
		return nil, errors.New("no page drawn")
	}
	return d.Image(0, 0.5), nil
}
