package main

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The custom theme's editor (Preferences > Appearance > Customize
// Colours…): start from any theme, change its key colours with a live
// preview and the contrast checker's warnings, keep it as the custom
// theme, or reset it to its base.

// showThemeModes shows what the chosen theme has, and lets the light or
// dark choice be made only where there are both.
func (pd *PreferencesDialog) showThemeModes() {
	if pd.themeModes == nil || pd.variantSelect == nil {
		return
	}
	id := themeIDOf(pd.themeSelect.Selected)
	d := themeDefByID(id)
	if id == customThemeID {
		base := "slate"
		if c := loadCustomTheme(pd.settings); c != nil {
			base = c.Base
		}
		d = themeDefByID(base)
	}
	if d == nil {
		pd.themeModes.SetText("")
		return
	}
	switch d.modes() {
	case "light and dark":
		pd.variantSelect.Enable()
		pd.themeModes.SetText(d.label + " has a light and a dark variant: it follows the system's preference unless you choose one.")
	default:
		pd.variantSelect.Disable()
		pd.themeModes.SetText(d.label + " is " + d.modes() + ".")
	}
}

// themeEditor is the custom theme being edited.
type themeEditor struct {
	pd      *PreferencesDialog
	custom  *customTheme
	dark    bool // the variant being edited
	base    *widget.Select
	variant *widget.RadioGroup
	rows    *fyne.Container
	preview *fyne.Container
	summary *widget.Label
}

// customizeTheme opens the theme editor, on the custom theme there is or
// on the theme chosen.
func (pd *PreferencesDialog) customizeTheme() {
	c := loadCustomTheme(pd.settings)
	if c == nil {
		base := themeIDOf(pd.themeSelect.Selected)
		if base == customThemeID {
			base = "slate"
		}
		c = &customTheme{Base: base}
	}
	te := &themeEditor{pd: pd, custom: c}
	te.dark = te.shownDark()
	te.show()
}

// shownDark is whether the dark variant is the one to edit first: the
// one shown now, if the base has it.
func (te *themeEditor) shownDark() bool {
	d := themeDefByID(te.custom.Base)
	if d == nil || d.dark == nil {
		return false
	}
	if d.light == nil {
		return true
	}
	return fyne.CurrentApp().Settings().ThemeVariant() == theme.VariantDark
}

// working is the theme as edited, in the variant edited.
func (te *themeEditor) working() (*accoladeTheme, *themePalette, fyne.ThemeVariant) {
	v := theme.VariantLight
	if te.dark {
		v = theme.VariantDark
	}
	at := newAccoladeTheme(customThemeID, "", te.custom).(*accoladeTheme)
	return at, at.pal(v), v
}

// changes is the variant's changed colours.
func (te *themeEditor) changes() map[string]string {
	if te.dark {
		if te.custom.Dark == nil {
			te.custom.Dark = map[string]string{}
		}
		return te.custom.Dark
	}
	if te.custom.Light == nil {
		te.custom.Light = map[string]string{}
	}
	return te.custom.Light
}

func (te *themeEditor) show() {
	var baseLabels []string
	for _, d := range themeDefs {
		baseLabels = append(baseLabels, d.label)
	}
	te.base = widget.NewSelect(baseLabels, func(label string) {
		id := themeIDOf(label)
		if id == te.custom.Base {
			return
		}
		te.custom = &customTheme{Base: id}
		te.dark = te.shownDark()
		te.refresh()
	})
	te.variant = widget.NewRadioGroup([]string{"Light", "Dark"}, func(s string) {
		if s == "" {
			return
		}
		if d := s == "Dark"; d != te.dark {
			te.dark = d
			te.refresh()
		}
	})
	te.variant.Horizontal = true
	te.rows = container.NewVBox()
	te.preview = container.NewStack()
	te.summary = widget.NewLabel("")
	te.summary.Wrapping = fyne.TextWrapWord

	reset := widget.NewButtonWithIcon("Reset to Base", theme.ContentUndoIcon(), func() {
		te.custom = &customTheme{Base: te.custom.Base}
		te.refresh()
	})
	controls := container.NewVBox(
		labelled("Start from:", te.base),
		labelled("Variant:", te.variant),
		hint("Change a colour by its swatch or its #rrggbb. A ⚠ marks a colour that reads too faintly somewhere (WCAG AA: 4.5:1 for text, 3:1 for marks and edges)."),
	)
	list := container.NewVScroll(te.rows)
	list.SetMinSize(fyne.NewSize(360, 380))
	left := container.NewBorder(controls, container.NewVBox(te.summary, container.NewHBox(reset)), nil, nil, list)
	body := container.NewHSplit(left, container.NewPadded(te.preview))
	body.SetOffset(0.5)

	parent := te.pd.win
	d := dialog.NewCustomConfirm("Customize Colours", "Use as Custom Theme", "Close", body, func(ok bool) {
		if !ok {
			return
		}
		s := te.pd.settings
		saveCustomTheme(s, te.custom)
		s.SetString("theme", customThemeID)
		te.pd.themeSelect.SetSelected(themeLabel(customThemeID))
		te.pd.app.setColorScheme(customThemeID)
		s.Save()
	}, parent)
	d.Resize(fyne.NewSize(min(parent.Canvas().Size().Width-16, 900), min(parent.Canvas().Size().Height-16, 640)))
	te.refresh()
	d.Show()
}

// refresh shows the rows, the warnings and the preview of the theme as
// it is now.
func (te *themeEditor) refresh() {
	d := themeDefByID(te.custom.Base)
	if d == nil { // a base no longer there
		te.custom = &customTheme{Base: "slate"}
		d = themeDefByID("slate")
	}
	te.base.Selected = d.label
	te.base.Refresh()
	te.variant.Selected = "Light"
	if te.dark {
		te.variant.Selected = "Dark"
	}
	if d.light == nil || d.dark == nil {
		te.variant.Disable()
	} else {
		te.variant.Enable()
	}
	te.variant.Refresh()

	at, p, v := te.working()
	pairs := contrastPairs(at, v)
	low := lowKeys(pairs)
	te.rows.RemoveAll()
	for _, k := range customKeys {
		te.rows.Add(te.row(k, keyColor(p, k.id), low[k.id]))
	}
	if n := len(lowPairs(pairs)); n > 0 {
		te.summary.SetText(fmt.Sprintf("⚠ %d pairs read too faintly; the marked colours are in them.", n))
	} else {
		te.summary.SetText(fmt.Sprintf("Every pair of colours reads at WCAG AA or better (%d checked).", len(pairs)))
	}
	te.preview.Objects = []fyne.CanvasObject{themePreview(at, v)}
	te.preview.Refresh()
}

// row is a key colour's row: its name, swatch, hex and warning.
func (te *themeEditor) row(k customKey, c color.NRGBA, low []string) fyne.CanvasObject {
	swatch := canvas.NewRectangle(c)
	swatch.CornerRadius = 3
	swatch.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	swatch.StrokeWidth = 1
	swatch.SetMinSize(fyne.NewSize(28, 22))
	set := func(hexText string) {
		te.changes()[k.id] = hexOf(hexColor(hexText))
		te.refresh()
	}
	pick := widget.NewButton("", func() {
		cp := dialog.NewColorPicker(k.label, "", func(nc color.Color) { set(hexOf(nc)) }, te.pd.win)
		cp.Advanced = true
		cp.SetColor(c)
		cp.Show()
	})
	fyne.SetAccessibleLabel(pick, "Pick "+k.label)
	entry := widget.NewEntry()
	entry.SetText(hexOf(c))
	entry.OnSubmitted = func(s string) {
		if validHex(s) {
			set(s)
		}
	}
	entry.Validator = func(s string) error {
		if !validHex(s) {
			return fmt.Errorf("a colour as #rrggbb")
		}
		return nil
	}
	fyne.SetAccessibleLabel(entry, k.label+" as #rrggbb")
	name := widget.NewLabel(k.label)
	if _, changed := te.changes()[k.id]; changed {
		name.TextStyle.Bold = true
	}
	line := container.NewBorder(nil, nil,
		container.NewStack(pick, container.NewCenter(swatch)), nil,
		container.NewGridWithColumns(2, name, entry))
	if len(low) == 0 {
		return line
	}
	sort.Strings(low)
	if len(low) > 3 {
		low = append(low[:3], fmt.Sprintf("and %d more", len(low)-3))
	}
	warn := widget.NewLabel("⚠ " + strings.Join(low, "; "))
	warn.Importance = widget.WarningImportance
	warn.SizeName = theme.SizeNameCaptionText
	warn.Wrapping = fyne.TextWrapWord
	return container.NewVBox(line, warn)
}

func validHex(s string) bool {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return false
	}
	for _, r := range strings.ToLower(s) {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}

// themePreview is a small picture of Accolade in a theme: the ribbon's
// tabs, the script with a scene heading, a note, a fold and the margin's
// marks, the Notes pane's cards with their edges, and a dialog.
func themePreview(th fyne.Theme, v fyne.ThemeVariant) fyne.CanvasObject {
	c := func(n fyne.ThemeColorName) color.Color { return th.Color(n, v) }
	text := func(s string, n fyne.ThemeColorName, bold, italic, mono bool, size float32) *canvas.Text {
		t := canvas.NewText(s, c(n))
		t.TextStyle = fyne.TextStyle{Bold: bold, Italic: italic, Monospace: mono}
		t.TextSize = size
		return t
	}
	box := func(fill color.Color, radius float32, content fyne.CanvasObject) fyne.CanvasObject {
		r := canvas.NewRectangle(fill)
		r.CornerRadius = radius
		return container.NewStack(r, container.New(layout.NewCustomPaddedLayout(4, 4, 6, 6), content))
	}
	edge := func(col color.Color, content fyne.CanvasObject) fyne.CanvasObject {
		bar := canvas.NewRectangle(col)
		bar.SetMinSize(fyne.NewSize(3, 0))
		return box(c(colorNameRaised), 2, container.NewBorder(nil, nil, bar, nil, container.NewPadded(content)))
	}
	caps := func(s string) string {
		if at, ok := th.(*accoladeTheme); ok && at.def != nil && at.def.caps {
			return strings.ToUpper(s)
		}
		return s
	}

	// the ribbon's tabs
	tab := func(s string, active bool) fyne.CanvasObject {
		t := text(s, theme.ColorNameForeground, active, false, false, 12)
		if !active {
			return container.NewPadded(t)
		}
		if a := c(colorNameActive); a != nil && !transparent(a) {
			t.Color = c(colorNameOnActive)
			return box(a, 2, t)
		}
		line := canvas.NewRectangle(c(theme.ColorNamePrimary))
		line.SetMinSize(fyne.NewSize(0, 2))
		return container.NewBorder(nil, line, nil, nil, container.NewPadded(t))
	}
	ribbon := box(c(theme.ColorNameHeaderBackground), 0, container.NewHBox(tab("Home", true), tab("Script", false), tab("View", false),
		layout.NewSpacer(), text(caps("Elements"), theme.ColorNameDisabled, true, false, false, 10)))

	// the script
	revision := fitInk(revisionInk("Blue"), c(theme.ColorNameBackground))
	mark := func(col color.Color) fyne.CanvasObject {
		r := canvas.NewRectangle(col)
		r.SetMinSize(fyne.NewSize(3, 14))
		return r
	}
	gutter := func(s string, m fyne.CanvasObject) fyne.CanvasObject {
		t := text(s, "accolade-fold", false, false, true, 12)
		if m == nil {
			m = canvas.NewRectangle(color.Transparent)
		}
		return container.NewHBox(t, m)
	}
	var sceneCol color.Color = c("accolade-scene-heading")
	scene := text("INT. BARN - DAY", theme.ColorNameForeground, true, false, true, 12)
	scene.Color = sceneCol
	noteCol := c("accolade-note")
	note := text("[[Is the barn hers yet?]]", theme.ColorNameForeground, false, true, true, 12)
	note.Color = noteCol
	at, _ := th.(*accoladeTheme)
	todo := color.Color(c(theme.ColorNameWarning))
	if at != nil {
		todo = marginColorOf(th, v, openTodo)
	}
	script := container.NewVBox(
		container.NewHBox(gutter("▾", nil), scene),
		container.NewHBox(gutter(" ", mark(revision)), text("Mara drags the door open.", theme.ColorNameForeground, false, false, true, 12)),
		container.NewHBox(gutter(" ", mark(todo)), note),
		container.NewHBox(gutter(" ", nil), text("                MARA", theme.ColorNameForeground, true, false, true, 12)),
		container.NewHBox(gutter(" ", nil), text("          It was always ours.", theme.ColorNameForeground, false, false, true, 12)),
		container.NewHBox(gutter("▸", nil), text("EXT. FIELD - NIGHT  · 3 lines", "accolade-fold", false, false, true, 12)),
		text("12 lines · 1 open", theme.ColorNamePlaceHolder, false, false, false, 10),
	)

	// the Notes pane
	card := func(kind string, k openKind, body string) fyne.CanvasObject {
		return edge(kindColorOf(th, v, k), container.NewVBox(
			text(caps(kind), theme.ColorNameDisabled, true, false, false, 10),
			text(body, theme.ColorNameForeground, false, false, false, 12)))
	}
	notes := box(c(theme.ColorNameMenuBackground), 3, container.NewVBox(
		text(caps("Notes"), theme.ColorNameForeground, true, false, false, 12),
		card("To do · line 3", openTodo, "Is the barn hers yet?"),
		card("Beat", openBeat, "She claims the farm"),
		card("Note", openNote, "Mother's letter"),
	))

	// a dialog
	btn := box(c(theme.ColorNamePrimary), 3, text("Save", theme.ColorNameForegroundOnPrimary, true, false, false, 12))
	dlg := box(c(theme.ColorNameOverlayBackground), 4, container.NewVBox(
		text("Save changes?", theme.ColorNameForeground, true, false, false, 13),
		text("Your script has unsaved changes.", theme.ColorNameDisabled, false, false, false, 11),
		text("What is kept", theme.ColorNameHyperlink, false, false, false, 11),
		container.NewHBox(layout.NewSpacer(), box(c(theme.ColorNameButton), 3, text("Cancel", theme.ColorNameForeground, false, false, false, 12)), btn),
	))

	sep := canvas.NewRectangle(c(theme.ColorNameSeparator))
	sep.SetMinSize(fyne.NewSize(0, 1))
	middle := container.NewBorder(nil, nil, nil, notes, container.NewPadded(script))
	bg := canvas.NewRectangle(c(theme.ColorNameBackground))
	return container.NewStack(bg, container.NewBorder(container.NewVBox(ribbon, sep), container.NewPadded(dlg), nil, nil, middle))
}

func transparent(c color.Color) bool {
	_, _, _, a := c.RGBA()
	return a == 0
}
