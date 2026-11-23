package main

import "fmt"

// The views derived from the document in front (docs/STATE.md, LOOP.md
// 2026-11-22, Joop: the preview "gets stuck when clicking swap" — "check
// for similar state change issues and ways to prevent similar bugs").
//
// Each view that describes a document records which one it shows: a
// docKey, the document and the script's epoch then. Every transition of
// what the window shows ends in mainDocChanged: a middle tab coming to the
// front (middle.show), and anything that replaces or changes the script
// (Open, New, an import, a restore, a join or leaving one, the document
// type, a rename, another source), bracketed by scriptChange, and a role
// change (which puts panes and tabs away). There
// each view in sight whose record is not what it should describe follows:
// one place, instead of each transition resetting what it remembers to.
// TestViewsFollowTheFront drives every transition and checks each view.

// docKey names the document a view describes: the document, and the
// script's epoch (the script's document stays the same object when the
// window opens another script: the epoch tells them apart).
type docKey struct {
	doc   *document
	epoch uint64
}

func (k docKey) String() string {
	if k.doc == nil {
		return fmt.Sprintf("nothing (epoch %d)", k.epoch)
	}
	return fmt.Sprintf("%s (epoch %d)", k.doc.kind, k.epoch)
}

// frontDoc is the document the window's views describe: the one in
// front in the middle; with a view in front there (the cards, the
// outline), the script.
func (w *MainWindow) frontDoc() *document {
	if d := w.main(); d != nil {
		return d
	}
	return w.scriptDoc()
}

// frontKey and scriptKey are the keys of the front document and of the
// script, now.
func (w *MainWindow) frontKey() docKey  { return docKey{w.frontDoc(), w.epoch} }
func (w *MainWindow) scriptKey() docKey { return docKey{w.scriptDoc(), w.epoch} }

// docView is a view derived from a document.
type docView struct {
	name string
	// script: it describes the script whatever is in front (the cards,
	// the characters, the source's links); else the document in front
	script bool
	// shows is what it describes now, and whether it is in sight (out of
	// sight, a view is brought up to date as it comes into sight)
	shows func() (docKey, bool)
	// follow brings it to what it should describe
	follow func()
}

// want is the key view v should have now.
func (w *MainWindow) want(v docView) docKey {
	if v.script {
		return w.scriptKey()
	}
	return w.frontKey()
}

// mainDocChanged brings each view in sight that describes another
// document than it should along. Within a transition (scriptChange) it
// waits for its end.
func (w *MainWindow) mainDocChanged() {
	if w.changing > 0 || w.fyneWindow == nil || w.dock == nil {
		return // midway, or the window being built
	}
	for _, v := range w.docViews() {
		if k, on := v.shows(); on && k != w.want(v) {
			v.follow()
		}
	}
}

// scriptChange brackets a transition that replaces or changes the
// script: `defer w.scriptChange()()`. Nested ones are one; at the end of
// the outermost the epoch moves on, so every view in sight follows the
// script as it is then (one shown midway, of a script half read, too).
func (w *MainWindow) scriptChange() func() {
	w.changing++
	return func() {
		w.changing--
		if w.changing == 0 {
			w.epoch++
			w.mainDocChanged()
		}
	}
}

// docViews are the views derived from the documents, in the order they
// follow (the commands last: they look at what the others show).
func (w *MainWindow) docViews() []docView {
	return []docView{
		// the script's Preview: shown, or being made to be shown (Export's
		// preview and a report's PDF in the page view are not followed)
		{name: "pages", shows: func() (docKey, bool) {
			coming := w.pagesComing != 0 && w.pagesComing == w.previewGen
			return w.pagesWant, w.pagesVisible && w.pages != nil && w.pagesPreview || coming
		}, follow: func() {
			if (w.scriptInMain() || w.mid.has("pages")) && !w.markdown { // (a phone's Pages tab: made again in it)
				w.previewPDF() // made again, the pages there until then
				return
			}
			w.previewGen++ // one being made is not shown
			w.hidePages()
			w.previewFollowsMain(w.main() != nil)
		}},
		{name: "live preview", shows: func() (docKey, bool) {
			if !w.livePreviewShown() {
				return docKey{}, false
			}
			return w.previewPane.of, true
		}, follow: func() {
			if w.editorMarkdown() && w.main() != nil {
				w.updateLivePreview()
				return
			}
			w.previewVisible = false
			w.relayout()
			w.previewFollowsMain(w.main() != nil)
		}},
		{name: "find bar", shows: func() (docKey, bool) {
			if w.searchBar == nil || !w.searchBar.IsVisible() {
				return docKey{}, false
			}
			return w.searchBar.of, true
		}, follow: func() { w.searchBar.updateSearch() }},
		{name: "outline", shows: func() (docKey, bool) {
			if w.outline == nil || !w.outlineInSight() {
				return docKey{}, false
			}
			return w.outline.of, true
		}, follow: func() { w.outline.refresh() }},
		{name: "cards pane", script: true, shows: func() (docKey, bool) {
			if w.cardsDock == nil || !w.paneInSight("cards") {
				return docKey{}, false
			}
			return w.cardsDock.of, true
		}, follow: func() { w.cardsDock.update(w.textEditor.Text()) }},
		{name: "cards view", script: true, shows: func() (docKey, bool) {
			if w.cards == nil || !w.cardsShown() {
				return docKey{}, false
			}
			return w.cards.of, true
		}, follow: func() { w.cards.update(w.textEditor.Text()) }},
		{name: "stripboard", script: true, shows: func() (docKey, bool) {
			if !w.stripsShown() {
				return docKey{}, false
			}
			return w.stripboard.of, true
		}, follow: func() { w.stripboard.update() }},
		{name: "characters", script: true, shows: func() (docKey, bool) {
			if w.charsPane == nil || !w.charsPane.inSight() {
				return docKey{}, false
			}
			return w.charsPane.of, true
		}, follow: func() { w.charsPane.read() }},
		{name: "source pane", script: true, shows: func() (docKey, bool) {
			if w.source == nil || !w.source.shown() {
				return docKey{}, false
			}
			return w.source.of, true
		}, follow: func() { w.source.follow() }},
		{name: "adapt dialog", script: true, shows: func() (docKey, bool) {
			if w.source == nil || w.source.adapting == nil {
				return docKey{}, false
			}
			return w.source.adapting.of, true
		}, follow: func() {
			w.source.adapting.close()
			w.setStatus("Adapt… closed: the script changed under it")
		}},
		{name: "status bar", shows: func() (docKey, bool) { return w.statusOf, w.statsLabel != nil },
			follow: func() { w.updateCurrentElement(); w.updateStatusBar() }},
		{name: "title", shows: func() (docKey, bool) { return w.titleOf, true }, follow: w.updateTitle},
		{name: "commands", shows: func() (docKey, bool) { return w.commandsOf, true }, follow: w.editorKindChanged},
	}
}
