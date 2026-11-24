package main

import (
	"errors"
	"fmt"
	"image"
	"slices"
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/buffer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/rules"
	"github.com/LaPingvino/lexington/titlepage"
)

// The title page's template (Lexington's title page templates,
// docs/TITLEPAGE.md there): where each key of the title page goes on the
// printed page, and how big, whatever the order the keys are written in.
// It is a print setting as the others are (Page Setup's "Title page:",
// the "title-page-template" setting: "" the standard one, Fountain's),
// chosen where the title page is written: the Title Page tab, there
// while the cursor is in it, has the choice, the template's own keys to
// add (Logline, Picture …) and Preview, whose pages follow the choice.
// The templates are Lexington's presets (standard, bold, poster) and the
// [TitlePages] of one's lexington.toml, edited in the settings editor's
// Title Pages page.

// titleTemplateNames are the templates the script can use, as lists show
// them, in the order of their keys (rules.TOMLConf.TitlePageNames).
func (c lexConf) titleTemplateNames() (keys, names []string) {
	for _, k := range c.TitlePageNames() {
		keys = append(keys, k)
		names = append(names, c.TitlePageName(k))
	}
	return keys, names
}

// titleTemplate is the key of the template the window's script prints
// its title page with: its own, else the default (titlepage_choice.go);
// the standard one ("") if the configuration has none of that name (one's
// own, deleted since).
func (w *MainWindow) titleTemplate() string {
	if w.settings == nil {
		return ""
	}
	return w.validTitleTemplate(w.scriptTitleTemplate())
}

// validTitleTemplate is key if the script's configuration has a template
// of that name, else "" (the standard one).
func (w *MainWindow) validTitleTemplate(key string) string {
	if _, ok := w.lex().TitlePage(key); !ok {
		return ""
	}
	return key
}

// newTitleTemplateSelect is a choice of the templates; changed (if any)
// is given the key chosen.
func (w *MainWindow) newTitleTemplateSelect(changed func(key string)) *widget.Select {
	_, names := w.lex().titleTemplateNames()
	s := widget.NewSelect(names, nil)
	s.OnChanged = func(string) {
		if changed != nil {
			changed(w.chosenTitleTemplate(s))
		}
	}
	fyne.SetAccessibleLabel(s, "Title page template")
	return s
}

// selectTitleTemplate shows the template with key in s ("": the
// standard one), its list made again (one's own may have come).
func (w *MainWindow) selectTitleTemplate(s *widget.Select, key string) {
	keys, names := w.lex().titleTemplateNames()
	s.Options = names
	i := slices.Index(keys, key)
	if i < 0 {
		i = 0
	}
	s.SetSelected(names[i])
}

// chosenTitleTemplate is the key of the template chosen in s, "" for the
// standard one.
func (w *MainWindow) chosenTitleTemplate(s *widget.Select) string {
	keys, names := w.lex().titleTemplateNames()
	if i := slices.Index(names, s.Selected); i >= 0 && keys[i] != rules.StandardTitlePage {
		return keys[i]
	}
	return ""
}

// setTitleTemplate makes key the title page's template: the script's, and
// the default for new scripts (setPageSetup); Export, Print and Preview
// follow (a preview shown is made again).
func (w *MainWindow) setTitleTemplate(key string) {
	p := w.pageSetup()
	if key == rules.StandardTitlePage {
		key = ""
	}
	if p.titleTemplate == key {
		return
	}
	p.titleTemplate = key
	w.setPageSetup(p)
	w.ribbonRefresh()
	w.setStatus("Title page: " + w.lex().TitlePageName(key))
}

// titleTemplateOf is the template key names, with the script's language.
func (w *MainWindow) titleTemplateOf(key string) titlepage.Template {
	tpl, _ := titlepage.For(w.lex().TOMLConf, key, w.language)
	return tpl
}

// titleTemplateKeys are the keys the window's template names that the
// Title Page tab has no button for already (Logline, Picture,
// Registration …), in the template's order.
func (w *MainWindow) titleTemplateKeys() []rules.TitleKey {
	var out []rules.TitleKey
	for _, k := range w.titleTemplateOf(w.titleTemplate()).Page.Keys {
		key := strings.TrimSpace(k.Key)
		if key == "" || key == rules.OtherKeys || k.Hide || slices.ContainsFunc(titleKeys, func(s string) bool {
			return strings.EqualFold(s, key) || strings.EqualFold(key, "Authors") && s == "Author"
		}) || slices.ContainsFunc(out, func(o rules.TitleKey) bool { return strings.EqualFold(o.Key, key) }) {
			continue
		}
		out = append(out, k)
	}
	return out
}

// addTemplateKey adds a template's key to the title page. A picture's
// asks for the picture first (copied in as Insert › Storyboard Frame…
// does: into the bundle's assets/, or a folder beside the script) and
// writes it as a Markdown picture; with nowhere to keep one (unsaved, a
// plain file in the browser), a picture's name to fill in.
func (w *MainWindow) addTemplateKey(k rules.TitleKey) {
	if !k.Picture {
		w.addTitleKey(k.Key)
		return
	}
	put := func(src string) {
		w.addTitleKey(k.Key)
		at := w.textEditor.CursorOffset()
		text := []rune(w.textEditor.Text())
		if at > 0 && at <= len(text) && (at == len(text) || text[at] == '\n') && text[at-1] == ' ' {
			w.textEditor.Edit(func(b *buffer.Buffer) { b.Replace(at, at, "![]("+src+")") })
			w.textEditor.SetCursorOffset(at + 2) // in its alt text, to say what it shows
		}
	}
	if w.textEditor == nil {
		return
	}
	if w.bundle == nil && (w.currentFile == "" || inBrowser) {
		put("picture.png")
		w.setStatus(k.Key + " added: save the script (as an .accolade to keep pictures inside it) and name its picture")
		return
	}
	chooseFileIn(w.fyneWindow, pictureExtensions, func(picked string) {
		w.keepPicture(picked, w.pictureFolder(""), func(rel string, err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not add the picture: %v", err), w.fyneWindow)
				return
			}
			put(rel)
			w.setStatus(k.Key + " added: " + rel + " (the template prints it)")
			w.focusMain()
		})
	})
}

// titlePictures are the files the title page's pictures name, as the
// window reads pictures (beside the script, in its bundle, the browser's
// kept files): what Lexington prints them from (it reads none itself).
func (w *MainWindow) titlePictures(text string, j exportJob) map[string][]byte {
	if !strings.Contains(text, ":") {
		return nil
	}
	head := text
	if end := titlePageEnd(strings.Split(text, "\n")); end > 0 {
		head = strings.Join(strings.Split(text, "\n")[:end], "\n")
	} else {
		return nil
	}
	lang := w.lex().Language(w.language)
	s := fountain.ParseIn(lang, strings.NewReader(head+"\n\nINT. X - DAY\n"))
	out := map[string][]byte{}
	for _, b := range w.titleTemplateOf(j.titleTemplate).Layout(s) {
		for _, f := range b.Fields {
			if f.Picture == "" || out[f.Picture] != nil {
				continue
			}
			if p := w.pictures().Load(f.Picture); p.OK() {
				out[f.Picture] = p.Data
			}
		}
	}
	return out
}

// The Title Page tab's own homes: the template's choice, and a button for
// each key it names that the tab has none for.
const (
	titleTemplateHome     = "titleTemplate"
	titleTemplateKeysHome = "titleTemplateKeys"
)

// titleTemplateKey is what of the template the Title Page tab is built
// for: its key and the keys it adds ("" off the tab).
func (r *ribbonBar) titleTemplateKey(env cmdEnv) string {
	if !env.titlePage || r.w.settings == nil {
		return ""
	}
	key := r.w.titleTemplate()
	for _, k := range r.w.titleTemplateKeys() {
		key += "\x01" + k.Key
	}
	return key
}

// titleTemplateSelect is the tab's choice of the template, showing the
// page setup's.
func (r *ribbonBar) titleTemplateSelect() *widget.Select {
	w := r.w
	if r.titleTpl == nil {
		r.titleTpl = w.newTitleTemplateSelect(nil)
	}
	r.titleTpl.OnChanged = nil
	w.selectTitleTemplate(r.titleTpl, w.titleTemplate())
	r.titleTpl.OnChanged = func(string) { w.setTitleTemplate(w.chosenTitleTemplate(r.titleTpl)) }
	return r.titleTpl
}

// titleTemplateKeyButtons are the buttons adding the template's own keys.
func (r *ribbonBar) titleTemplateKeyButtons() []*ribbonButton {
	var out []*ribbonButton
	for _, k := range r.w.titleTemplateKeys() {
		k := k
		b := newRibbonButton(r, nil, false)
		b.label, b.icon = k.Key, theme.ContentAddIcon()
		b.name = "Add " + k.Key + " to the title page (the template places it)"
		b.onTap = func() { r.w.addTemplateKey(k) }
		out = append(out, b)
	}
	return out
}

// titlePageImage is the title page of text (a title page alone will
// do) as the PDF prints it with the template key, its first page drawn
// at scale pixels a point: what the Title Page dialog previews. It reads
// no files and makes no PDF file (the browser too).
func (w *MainWindow) titlePageImage(text, key string, scale float64) (image.Image, error) {
	d, err := w.titlePageDrawing(text, key)
	if err != nil {
		return nil, err
	}
	return d.Image(0, scale), nil
}

// titlePageDrawing is the drawing titlePageImage shows: in the script's
// language, its direction and the computer's fonts asked in it for what
// Courier Prime lacks (a Japanese script's Han in a Japanese face), as
// the export prints it.
func (w *MainWindow) titlePageDrawing(text, key string) (pdf.Drawing, error) {
	conf := w.lex()
	lang := conf.Language(w.language)
	s := fountain.ParseIn(lang, strings.NewReader(strings.TrimRight(text, "\n")+"\n\n.THE SCRIPT\n"))
	tpl := w.titleTemplateOf(key)
	j := w.layoutJob()
	j.titleTemplate = key
	var d pdf.Drawing
	pw := &pdf.PDFWriter{Elements: conf.formatElements(w.scriptFormatKey()), Page: w.paperOf(w.pageSetup().paper),
		TitlePage: &tpl, TitlePictures: w.titlePictures(text, j), Record: &d, RTL: lang.RTL(),
		Language: fontLanguage(w.language)}
	if _, err := pw.Render(s); err != nil {
		return d, err
	}
	if len(d.Pages) == 0 {
		return d, errors.New("no title page to show")
	}
	return d, nil
}
