package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The Preview tab: the ribbon's contextual tab while a preview is shown
// (the page view, or a Markdown document's Live Preview), brought forward
// when it opens and gone when it closes — the zoom, the pages, the PDF,
// its page setup, Text or Pages (a Markdown document's), Refresh and
// Follow Edits (pagefollow.go), Close Preview. In the Ribbon look the bar
// over the pages goes (the tab has it all); the Menus look keeps it.

// previewView is the page view shown: the window's, or the Live
// Preview's Pages; nil if none.
func (w *MainWindow) previewView() *pageView {
	if w.pagesVisible && w.pages != nil {
		return w.pages
	}
	if v := w.mdPagesView(); v != nil && w.previewVisible && w.previewPane.showsPages() {
		return v
	}
	return nil
}

// mdPagesView is the Live Preview's page view, if made.
func (w *MainWindow) mdPagesView() *pageView {
	if w.previewPane != nil && w.previewPane.pages != nil {
		return w.previewPane.pages.view
	}
	return nil
}

// livePreviewShown reports whether a document's Live Preview is the
// preview shown.
func (w *MainWindow) livePreviewShown() bool {
	return !w.pagesVisible && w.previewVisible && w.previewPane != nil
}

// The rules of the Preview tab's commands.
func previewShown(e cmdEnv) bool    { return e.preview }
func pagesInView(e cmdEnv) bool     { return e.pagesShown }
func previewPDFFile(e cmdEnv) bool  { return e.pdfFile }
func previewRefresh(e cmdEnv) bool  { return e.refreshable }
func setupOfPreview(e cmdEnv) bool  { return e.setup }
func livePreviewOnly(e cmdEnv) bool { return e.livePreview }

// previewEnv fills in what the Preview tab's rules look at.
func (w *MainWindow) previewEnv(e *cmdEnv) {
	// (a phone's pages: while their middle tab is in front)
	e.preview = w.pagesVisible && w.pages != nil && (!w.mid.has("pages") || w.mid.shown == "pages") || w.livePreviewShown()
	e.livePreview = w.livePreviewShown()
	v := w.previewView()
	e.pagesShown = v != nil && v.d != nil && len(v.d.Pages) > 0
	e.pdfFile = v != nil && (v.pdfPath != "" || v.freshPDF != nil)
	e.refreshable = e.livePreview || v != nil && (v.follow != nil || v.remake != nil)
	e.setup = e.livePreview || v != nil && v.setupLine != nil
}

// previewCommands are the Preview tab's.
func previewCommands() []*command {
	view := func(f func(v *pageView)) func(w *MainWindow) {
		return func(w *MainWindow) {
			if v := w.previewView(); v != nil {
				f(v)
			}
		}
	}
	save := &command{id: "preview.savePDF", label: "Save PDF…", icon: res(theme.DocumentSaveIcon()), when: previewShown,
		enabled: previewPDFFile, run: view(func(v *pageView) { v.savePDF() })}
	if inBrowser {
		save.label, save.icon, save.run = "Download PDF", res(theme.DownloadIcon()), view(func(v *pageView) { v.download() })
	}
	var refreshKeys []fyne.Shortcut
	refreshName := "Refresh"
	if !inBrowser { // the browser's F5 loads the page again; Ctrl+R too (and vim's redo)
		refreshKeys = []fyne.Shortcut{plainKey(fyne.KeyF5, 0)}
		refreshName += " (F5)"
	}
	return []*command{
		{id: "preview.fit", label: "Fit Width", ribbonLabel: "Fit", tip: "Fit the page's width", icon: res(theme.ViewFullScreenIcon()),
			when: previewShown, enabled: pagesInView, checked: func(w *MainWindow) bool { v := w.previewView(); return v != nil && v.fit },
			run: view(func(v *pageView) { v.setFit() })},
		{id: "preview.actual", label: "Actual Size", ribbonLabel: "100%", icon: res(theme.ZoomFitIcon()),
			when: previewShown, enabled: pagesInView, run: view(func(v *pageView) { v.setZoom(1) })},
		{id: "preview.zoomIn", label: "Zoom In", icon: res(theme.ZoomInIcon()), when: previewShown, enabled: pagesInView,
			run: view(func(v *pageView) { v.zoomBy(1) })},
		{id: "preview.zoomOut", label: "Zoom Out", icon: res(theme.ZoomOutIcon()), when: previewShown, enabled: pagesInView,
			run: view(func(v *pageView) { v.zoomBy(-1) })},
		{id: "preview.previousPage", label: "Previous Page", ribbonLabel: "Previous", icon: res(theme.NavigateBackIcon()),
			when: previewShown, enabled: pagesInView, run: view(func(v *pageView) { v.area.toPage(-1) })},
		{id: "preview.nextPage", label: "Next Page", ribbonLabel: "Next", icon: res(theme.NavigateNextIcon()),
			when: previewShown, enabled: pagesInView, run: view(func(v *pageView) { v.area.toPage(1) })},
		save,
		{id: "preview.openViewer", label: "Open in Viewer", ribbonLabel: "Open in Viewer", icon: res(theme.FileApplicationIcon()),
			when: func(e cmdEnv) bool { return e.preview && !e.browser }, enabled: previewPDFFile,
			run: view(func(v *pageView) { v.openInViewer() })},
		{id: "preview.pageSetup", label: "Page Setup…", icon: res(theme.DocumentIcon()), when: previewShown, enabled: setupOfPreview,
			ribbonLabelFor: func(w *MainWindow) string { return w.previewPaper() },
			nameFor:        func(w *MainWindow) string { return "Page Setup…: " + w.previewSetupSummary() },
			run:            func(w *MainWindow) { w.previewPageSetup() }},
		{id: "preview.text", label: "Text", tip: "The document as text", icon: res(theme.DocumentIcon()), when: livePreviewOnly,
			checked: func(w *MainWindow) bool { return w.previewPane != nil && !w.previewPane.showsPages() },
			run:     func(w *MainWindow) { w.livePreviewMode(previewText) }},
		{id: "preview.pages", label: "Pages", tip: "The document's PDF, its pages", icon: res(theme.FileIcon()), when: livePreviewOnly,
			checked: func(w *MainWindow) bool { return w.previewPane != nil && w.previewPane.showsPages() },
			run:     func(w *MainWindow) { w.livePreviewMode(previewPages) }},
		{id: "preview.refresh", label: "Refresh", tip: "Make the pages again", icon: res(theme.ViewRefreshIcon()), keys: refreshKeys,
			when: previewShown, enabled: previewRefresh,
			dot:     func(w *MainWindow) bool { v := w.previewView(); return v != nil && v.stale },
			nameFor: func(*MainWindow) string { return refreshName },
			run:     func(w *MainWindow) { w.refreshPreview() }},
		{id: "preview.follow", label: "Follow Edits", tip: "Make the pages again as the text changes", icon: res(theme.VisibilityIcon()),
			when: previewShown, enabled: previewRefresh, checked: (*MainWindow).followEdits,
			run: func(w *MainWindow) { w.setFollowEdits(!w.followEdits()) }},
		{id: "preview.close", label: "Close Preview", icon: res(theme.CancelIcon()), when: previewShown,
			run: func(w *MainWindow) { w.closePreview() }},
	}
}

// previewTab is the Preview tab's layout.
func previewTab() ribbonTab {
	return ribbonTab{title: "Preview", when: previewShown, groups: []ribbonGroup{
		{title: "Zoom", homes: []home{big("preview.fit"), at("preview.actual"), at("preview.zoomIn"), at("preview.zoomOut")}},
		{title: "Pages", homes: []home{{id: "pageCount"}, at("preview.previousPage"), at("preview.nextPage")}},
		{title: "Output", homes: []home{big("preview.savePDF"), at("preview.openViewer")}},
		{title: "Setup", homes: []home{big("preview.pageSetup")}},
		{title: "Show", when: livePreviewOnly, homes: []home{at("preview.text"), at("preview.pages")}},
		{title: "Refresh", homes: []home{big("preview.refresh"), at("preview.follow")}},
		{title: "Close", homes: []home{big("preview.close")}},
	}}
}

// refreshPreview makes the preview's pages again (Refresh, F5).
func (w *MainWindow) refreshPreview() {
	if w.livePreviewShown() {
		if p := w.previewPane; p.showsPages() && p.pages != nil {
			p.pages.updateAfter(w.writeEditor().Text(), 0)
		} else {
			p.update(w.writeEditor().Text())
		}
		return
	}
	if v := w.previewView(); v != nil {
		v.refresh()
	}
}

// closePreview closes the preview: the page view, or the Live Preview.
func (w *MainWindow) closePreview() {
	switch {
	case w.pagesVisible:
		w.hidePages()
	case w.previewVisible:
		w.preview() // (toggles it)
	}
	w.ribbonRefresh()
}

// livePreviewMode shows the Live Preview's text or its pages.
func (w *MainWindow) livePreviewMode(mode string) {
	if p := w.previewPane; p != nil && p.mode != nil {
		p.mode.SetSelected(mode)
	}
	w.ribbonRefresh()
}

// previewPaper is the preview's paper, the Setup button's label.
func (w *MainWindow) previewPaper() string {
	if w.livePreviewShown() || w.markdown {
		if p := w.markdownPageSetup(); p.paper != "" {
			return paperName(p.paper)
		}
		return "Default Paper"
	}
	if w.settings == nil {
		return "Page Setup"
	}
	return paperName(w.paperOf(w.pageSetup().paper))
}

// previewSetupSummary is the preview's page setup, in words.
func (w *MainWindow) previewSetupSummary() string {
	if v := w.previewView(); v != nil && v.setupLine != nil {
		v.setupLine.refresh()
		return v.setupLine.summary.Text
	}
	if w.livePreviewShown() && w.previewPane.line != nil {
		w.previewPane.line.refresh()
		return w.previewPane.line.summary.Text
	}
	return ""
}

// previewPageSetup opens the preview's Page Setup… (the pages are made
// again after it, as its line's button does).
func (w *MainWindow) previewPageSetup() {
	if v := w.previewView(); v != nil && v.setupLine != nil {
		v.setupLine.button.OnTapped()
		return
	}
	if w.livePreviewShown() && w.previewPane.line != nil {
		w.previewPane.line.button.OnTapped()
	}
}

// followPreview shows the preview's page and whether its pages are stale
// on the Preview tab (as the view scrolls and the text changes: without
// the whole ribbon's refresh).
func (r *ribbonBar) followPreview() {
	if r.selected != "Preview" || !r.w.ribbonShown() {
		return
	}
	v := r.w.previewView()
	if r.pageCount != nil {
		text := ""
		if v != nil {
			text = v.count.Text
		}
		if r.pageCount.Text != text {
			r.pageCount.SetText(text)
		}
	}
	for _, b := range r.byID["preview.refresh"] {
		b.setDot(v != nil && v.stale)
	}
	for _, b := range r.byID["preview.fit"] { // checked while the width fits
		b.follow(r.env)
	}
}

// followLook shows the bar over the pages in the Menus look, not in the
// Ribbon look (its Preview tab has it).
func (v *pageView) followLook() {
	bar := v.w == nil || !v.w.ribbonShown()
	// (a phone keeps the line with Close: the pages are in the script's place)
	setVisible(v.head, bar || v.narrow)
	setVisible(v.tools, bar)
	if v.setupLine != nil {
		setVisible(v.extra, bar)
	}
	v.fitTop()
}

// fitTop hides what is over the pages when all of it is hidden (an empty
// box would still take a line's height).
func (v *pageView) fitTop() {
	shown := false
	for _, o := range v.top.Objects {
		if o.Visible() && (o != v.extra || len(v.extra.Objects) > 0) {
			shown = true
		}
	}
	setVisible(v.top, shown)
	v.box.Refresh()
}
