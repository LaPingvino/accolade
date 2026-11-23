package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/docs/help"
	"github.com/LaPingvino/accolade/internal/tooltip"
)

// The Help tab (LOOP.md, Joop 2026-11-20): the pages of docs/help, beside
// Notes, Source and Production (and a Markdown document's notes), shown
// as the Markdown preview shows prose, with links that go to the page or
// heading they name and a search through every page. A picker at the top
// goes to any page or cheat sheet, the cheat sheets in the document's
// mode's order; Print… makes a cheat sheet a one-page card; the globe
// opens the page online (help.URL, with Accolade's version). The Help
// menu (commands.go) has the pages, the cheat sheets, Online Help and
// Send Feedback….

// openWebPage opens a web address in the browser (a variable for tests:
// nothing is opened or sent from them).
var openWebPage = func(address string) {
	if u, err := url.Parse(address); err == nil {
		fyne.CurrentApp().OpenURL(u)
	}
}

// helpPage is page slug as the app shows it: the keyboard shortcuts made
// from the registry for the window's mode, the others as written.
func (w *MainWindow) helpPage(slug string) (help.Page, bool) {
	p, ok := help.Get(slug)
	if !ok {
		return p, false
	}
	env := cmdEnv{markdown: w.editorMarkdown(), browser: inBrowser} // the keys of what the main editor holds
	switch slug {
	case "shortcuts":
		p.Markdown = shortcutsPage(env, false)
	case "cheat-shortcuts":
		p.Markdown = shortcutsSheet(env, false)
	case "ex":
		p.Markdown = exHelpPage()
	}
	return p, true
}

// helpPages are the pages and the cheat sheets in the window's order.
func (w *MainWindow) helpPages() []help.Page {
	var out []help.Page
	for _, p := range append(help.Pages(), help.CheatsFor(w.editorMarkdown())...) {
		q, _ := w.helpPage(p.Slug)
		out = append(out, q)
	}
	return out
}

// helpPane is the Help tab.
type helpPane struct {
	w       *MainWindow
	box     fyne.CanvasObject
	picker  *widget.Select
	search  *widget.Entry
	print   *tooltip.Button
	online  *tooltip.Button
	back    *tooltip.Button
	content *fyne.Container // the page's sections, or the search's hits
	scroll  *container.Scroll

	slugs    []string // the picker's options' pages
	page     help.Page
	anchors  map[string]fyne.CanvasObject // a heading's anchor: its section
	history  []string                     // pages gone back from ("slug#anchor")
	updating bool
}

func newHelpPane(w *MainWindow) *helpPane {
	h := &helpPane{w: w}
	h.picker = widget.NewSelect(nil, func(label string) {
		if h.updating {
			return
		}
		for i, o := range h.picker.Options {
			if o == label && i < len(h.slugs) {
				h.open(h.slugs[i], "")
				return
			}
		}
	})
	h.picker.PlaceHolder = "Help pages and cheat sheets"
	fyne.SetAccessibleLabel(h.picker, "Help page")
	h.search = widget.NewEntry()
	h.search.SetPlaceHolder("Search the help")
	fyne.SetAccessibleLabel(h.search, "Search the help")
	h.search.OnChanged = func(q string) { h.find(q) }
	h.back = tooltip.NewButton(theme.NavigateBackIcon(), "Back", h.goBack, w.tooltips)
	h.print = tooltip.NewButton(theme.DocumentPrintIcon(), "Print… (a cheat sheet as a one-page card)", h.printPage, w.tooltips)
	h.online = tooltip.NewButton(theme.ComputerIcon(), "Open this page online", func() { w.helpOnline() }, w.tooltips)
	for _, b := range []*tooltip.Button{h.back, h.print, h.online} {
		b.Importance = widget.LowImportance
	}
	h.content = container.NewVBox()
	h.scroll = container.NewVScroll(h.content)
	// the picker as narrow as the pane: its longest option set no width
	top := container.NewBorder(nil, nil, h.back, container.NewHBox(h.print, h.online), container.New(narrowFirst{}, h.picker))
	h.box = container.NewBorder(container.NewVBox(top, h.search), nil, nil, nil, h.scroll)
	h.fillPicker()
	return h
}

// fillPicker lists the pages and the cheat sheets, these in the mode's
// order, keeping the page shown.
func (h *helpPane) fillPicker() {
	var opts []string
	h.slugs = nil
	for _, p := range h.w.helpPages() {
		label := p.Title
		if p.Cheat {
			label = "Cheat sheet: " + p.Title
		}
		opts = append(opts, label)
		h.slugs = append(h.slugs, p.Slug)
	}
	h.updating = true
	h.picker.Options = opts
	h.picker.ClearSelected()
	for i, s := range h.slugs {
		if s == h.page.Slug {
			h.picker.SetSelectedIndex(i)
		}
	}
	h.updating = false
	h.picker.Refresh()
}

// open shows page slug, at the heading with anchor ("" the top).
func (h *helpPane) open(slug, anchor string) bool {
	p, ok := h.w.helpPage(slug)
	if !ok {
		return false
	}
	if h.page.Slug != "" && h.page.Slug != slug {
		h.history = append(h.history, h.page.Slug)
	}
	h.show(p)
	h.scrollTo(anchor)
	return true
}

// show puts p's sections in the pane.
func (h *helpPane) show(p help.Page) {
	h.page = p
	h.anchors = map[string]fyne.CanvasObject{}
	var objs []fyne.CanvasObject
	for _, s := range help.Sections(p.Markdown) {
		rt := widget.NewRichText()
		rt.Wrapping = fyne.TextWrapWord
		rt.Segments, _ = markdownSegments(s.Markdown, -1)
		h.links(rt.Segments)
		fyne.SetAccessibleLabel(rt, p.Title+": "+orNone(s.Heading, p.Title))
		if s.Anchor != "" {
			if _, dup := h.anchors[s.Anchor]; !dup {
				h.anchors[s.Anchor] = rt
			}
		}
		objs = append(objs, rt)
	}
	h.content.Objects = objs
	h.content.Refresh()
	h.fillPicker()
	if len(h.history) == 0 {
		h.back.Disable()
	} else {
		h.back.Enable()
	}
}

func orNone(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

// scrollTo scrolls to the heading with anchor (the top for "").
func (h *helpPane) scrollTo(anchor string) {
	h.scroll.Refresh()
	y := float32(0)
	if o, ok := h.anchors[anchor]; ok && anchor != "" {
		y = o.Position().Y
	}
	h.scroll.Offset = fyne.NewPos(0, y)
	h.scroll.Refresh()
}

// links makes the links in segs go where they say: a page or heading of
// the help in the pane, a web address in the browser.
func (h *helpPane) links(segs []widget.RichTextSegment) {
	for _, s := range segs {
		switch x := s.(type) {
		case *widget.HyperlinkSegment:
			if x.URL == nil {
				continue
			}
			link := x.URL.String()
			x.OnTapped = func() { h.follow(link) }
		case *widget.ParagraphSegment:
			h.links(x.Texts)
		case *widget.ListSegment:
			h.links(x.Items)
		}
	}
}

// follow goes to a link's target.
func (h *helpPane) follow(link string) {
	t := help.Resolve(h.page.Slug, link)
	switch {
	case t.URL != "":
		openWebPage(t.URL)
	case t.Slug == h.page.Slug:
		h.scrollTo(t.Anchor)
	default:
		if !h.open(t.Slug, t.Anchor) {
			h.w.setStatus("No help page " + t.Slug)
		}
	}
}

// goBack shows the page shown before.
func (h *helpPane) goBack() {
	if len(h.history) == 0 {
		return
	}
	slug := h.history[len(h.history)-1]
	h.history = h.history[:len(h.history)-1]
	if p, ok := h.w.helpPage(slug); ok {
		h.show(p)
		h.scrollTo("")
	}
}

// find shows the sections of every page with the words typed, each a
// link to its place; nothing typed, the page again.
func (h *helpPane) find(q string) {
	if strings.TrimSpace(q) == "" {
		h.show(h.page)
		h.scrollTo("")
		return
	}
	hits := help.Search(h.w.helpPages(), q)
	var objs []fyne.CanvasObject
	head := widget.NewLabel(fmt.Sprintf("%s for “%s”", plural(len(hits), "place"), strings.TrimSpace(q)))
	head.TextStyle.Italic = true
	objs = append(objs, head)
	for _, hit := range hits {
		hit := hit
		title := hit.Page.Title
		if hit.Section.Heading != "" && hit.Section.Heading != title {
			title += " › " + hit.Section.Heading
		}
		if hit.Page.Cheat {
			title = "Cheat sheet: " + title
		}
		b := widget.NewButton(title, func() {
			h.search.SetText("")
			h.open(hit.Page.Slug, hit.Section.Anchor)
		})
		b.Alignment = widget.ButtonAlignLeading
		b.Importance = widget.LowImportance
		snip := widget.NewLabel(hit.Snippet)
		snip.Wrapping = fyne.TextWrapWord
		snip.SizeName = theme.SizeNameCaptionText
		objs = append(objs, container.NewVBox(b, snip))
	}
	h.content.Objects = objs
	h.content.Refresh()
	h.scroll.Offset = fyne.Position{}
	h.scroll.Refresh()
}

// printPage prints the page shown: a cheat sheet as a one-page card, a
// page as a web page (through the browser's print dialog, as Print does
// a Markdown document).
func (h *helpPane) printPage() {
	p := h.page
	page, err := help.HTML(p, appVersion(), p.Cheat)
	if err == nil {
		// one folder, the page written over each time: the browser reads the file while it prints,
		// so it can't go at once, and a new folder per print was never removed
		dir := filepath.Join(os.TempDir(), "accolade-help")
		if err = os.MkdirAll(dir, 0o755); err == nil {
			out := filepath.Join(dir, p.Slug+".html")
			if err = os.WriteFile(out, page, 0o644); err == nil {
				h.w.setStatus("Print: choose a printer, or Save as PDF")
				printFile(out)
				return
			}
		}
	}
	dialog.ShowError(fmt.Errorf("could not prepare the page to print: %v", err), h.w.fyneWindow)
}

// focusTarget is what takes the keyboard in the tab: the search.
func (h *helpPane) focusTarget() fyne.Focusable { return h.search }

// shown reports whether the Help tab is in sight.
func (h *helpPane) shown() bool { return h.w.paneInSight("help") }

// showHelp opens page slug in the Help tab, at the heading with anchor.
func (w *MainWindow) showHelp(slug, anchor string) {
	h := w.helpPane()
	if !h.open(slug, anchor) {
		h.open("index", "")
	}
	w.showPane("help", true)
}

// helpOnline opens the help page shown (the contents if none) online,
// for this version of Accolade.
func (w *MainWindow) helpOnline() {
	slug := "index"
	if w.help != nil && w.help.page.Slug != "" && w.help.shown() {
		slug = w.help.page.Slug
	}
	openWebPage(help.URL(slug, appVersion(), ""))
	w.setStatus("The help opens in your browser: " + help.Site + slug + ".html")
}

// sendFeedback opens a new GitHub issue in the browser with the version
// filled in; nothing is sent from Accolade.
func (w *MainWindow) sendFeedback() {
	platform := runtime.GOOS
	if inBrowser {
		platform = "browser"
	}
	mode := "screenplay"
	if w.markdown {
		mode = "Markdown"
	}
	openWebPage(help.FeedbackURL(appVersion(), platform, mode))
	w.setStatus("A new issue opens on GitHub in your browser: nothing is sent until you send it there")
}

// narrowFirst lays its object out over all of its space, asking only for
// its height: a picker whose longest option would widen the pane.
type narrowFirst struct{}

func (narrowFirst) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.Position{})
		o.Resize(size)
	}
}

func (narrowFirst) MinSize(objs []fyne.CanvasObject) fyne.Size {
	h := float32(0)
	for _, o := range objs {
		h = max(h, o.MinSize().Height)
	}
	return fyne.NewSize(120, h)
}

// helpCommands are the Help menu's commands: a page each, a cheat sheet
// each, Online Help and Send Feedback….
func helpCommands() []*command {
	var out []*command
	for _, p := range help.Pages() {
		slug, label := p.Slug, p.Title
		switch slug { // not a page's title alone where a menu has it
		case "index":
			label = "Help Contents"
		case "production":
			label = "Production Help"
		}
		out = append(out, &command{id: "help.page." + slug, label: label, run: func(w *MainWindow) { w.showHelp(slug, "") }})
	}
	for _, p := range help.Cheats() {
		slug := p.Slug
		out = append(out, &command{id: "help." + slug, label: p.Title, run: func(w *MainWindow) { w.showHelp(slug, "") }})
	}
	out = append(out,
		&command{id: "help.cheats", label: "Cheat Sheets", submenu: func(w *MainWindow) *fyne.Menu {
			var items []*fyne.MenuItem
			env := cmdEnv{markdown: w.editorMarkdown(), browser: inBrowser}
			for _, p := range help.CheatsFor(w.editorMarkdown()) {
				items = append(items, w.menuItem(cmd("help."+p.Slug), "", env))
			}
			return fyne.NewMenu("Cheat Sheets", items...)
		}},
		&command{id: "help.online", label: "Online Help", run: (*MainWindow).helpOnline},
		&command{id: "help.feedback", label: "Send Feedback…", run: (*MainWindow).sendFeedback},
	)
	return out
}

// helpMenu is the Help menu's homes.
func helpMenu() []home {
	homes := []home{at("app.palette"), sep()} // every command, by a search
	for _, p := range help.Pages() {
		homes = append(homes, at("help.page."+p.Slug))
	}
	return append(homes, at("help.cheats"), sep(), at("help.online"), at("help.feedback"), sep(), at("help.about"))
}
