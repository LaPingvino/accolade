package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Fun Things to Try (Joop 2026-11-26: "don't hide it too well"): a help
// page made from the command registry, a playful line and the key for
// each feature, by the moment it is for. In the app each line is a link
// that does it: here if it can be done on the script open, else on a
// fresh copy of the sample (a Markdown document's on a new document).
// A line stands for its command and the ones beside it (a tab's own,
// the next and previous); funQuiet are the plain ones (Save, Cut, the
// help pages). A command in neither shows under its own label, and
// fun_test.go fails for it, so a new feature gets a line.
//
// The PDF import has top billing, as it is easily missed: the page's
// first line, a drop line in a window holding nothing of its own,
// File > New from a PDF…, and the first run's welcome.

// funLine is one line of the page.
type funLine struct {
	run  string   // the command its link gives
	also []string // the commands it stands for too; "x." is all of x.*
	line string
	// sample: tried on the sample even when it could run here (it
	// locks or issues: the person's own script is left as it is)
	sample bool
}

// funMoments are the page's sections, in order.
var funMoments = []string{"Writing", "Rewriting", "Adapting", "Producing", "Exporting"}

// funSamplePDF is the first line's: the sample printed by Lexington,
// then read back in as a screenplay.
const funSamplePDF = "file.importPDFScreenplay"

var funLines = map[string][]funLine{
	"Writing": {
		{run: funSamplePDF, also: []string{"file.importPDFMarkdown", "file.newFromPDF", "fixup."},
			line: "Open any screenplay PDF and keep writing: Accolade reads it back into a script (try it on the sample's, printed by Lexington)"},
		{run: "file.openSample", line: "Open the sample, Chekhov's chameleon half adapted, and break whatever you like"},
		{run: "file.new", also: []string{"file.newEmpty", "file.newWindow", "file.newEmptyWindow", "set.newLanguage", "set.defaultLanguage"},
			line: "Start a script in the language you write in: New brings a scene of its own"},
		{run: "format.language", also: []string{"format.rtl"}, line: "Write in Hebrew, Arabic, Japanese or Esperanto: the cues and headings follow the language"},
		{run: "file.newStructure", also: []string{"write.insertStructure", "edit.pasteStructure"}, line: "Begin from a structure, Save the Cat or the hero's journey, and fill in the beats"},
		{run: "set.everySectionBeat", line: "Plan in sections alone: turn on Every Section Is a Beat and each one waits in the Notes pane until a scene grows from it"},
		{run: "format.element.Character", also: []string{"format.element.", "write.elementKind"}, line: "Make any line any element with Ctrl+1 to 8; mostly Tab and Enter guess it first"},
		{run: "write.newScene", also: []string{"write.place", "write.character", "format.numberScenes"}, line: "Ask for a new scene and get the heading written for you"},
		{run: "write.note", also: []string{"write.unsortedNote", "view.notes"}, line: "Leave yourself a note mid-line; the Notes pane gathers them all"},
		{run: "write.marker", also: []string{"write.nextMarker", "write.previousMarker"}, line: "Drop a marker where you will come back, and hop between them"},
		{run: "view.outline", line: "Watch the outline grow beside you"},
		{run: "view.cards", also: []string{"cards."}, line: "Lay the scenes out as index cards, or on a timeline with a lane per character"},
		{run: "view.timelineStrip", also: []string{"format.color"}, line: "Colour the scenes and see the whole story as a strip over the page"},
		{run: "view.characters", also: []string{"chars."}, line: "Meet everyone who speaks, and rename one everywhere at once"},
		{run: "write.storyboardFrame", also: []string{"write.newFrame", "write.editFrame", "file.importStoryboarder"}, line: "Put a storyboard frame between the lines and draw it in Inkscape or Krita"},
		{run: "write.include", line: "Pull another file into the script, as Highland does"},
		{run: "file.titlePage", also: []string{"titlepage."}, line: "Fill the title page in as a form"},
		{run: "file.newMarkdown", also: []string{"md.", "view.references", "file.import", "file.importPandoc"},
			line: "Write the treatment or the novel next door in Markdown, footnotes and citations included"},
		{run: "set.suggest", line: "Let the source suggest the next line while you write; Tab takes it"},
		{run: "view.fullscreen", line: "Go full screen and let the rest of the world fall away"},
		{run: "set.editorKeys", line: "Write with Vim or Helix keys, if your fingers want them"},
		{run: "app.palette", line: "Forget the menus: a few letters find any command"},
		{run: "view.keyTips", also: []string{"set.ribbon", "view.ribbonLock", "set.quickRow", "view.topRowAdd", "view.topRowReset", "set.toolbar", "view.menu"},
			line: "Show a key on every ribbon button and never reach for the mouse"},
		{run: "set.theme", also: []string{"set.textLarger", "set.textSmaller", "set.font", "set.lineNumbers", "set.statusBar"}, line: "Try another theme, or larger text for a long night"},
	},
	"Rewriting": {
		{run: "structure.up", also: []string{"structure."}, line: "Move a scene or a speech up and down without cut and paste"},
		{run: "view.foldSections", also: []string{"view.foldHere", "view.headingsOnly", "view.unfoldAll"}, line: "Fold the script to its sections and see the shape of the story"},
		{run: "view.preview", also: []string{"preview."}, line: "Keep the pages open beside you; they follow your typing"},
		{run: "edit.fixLines", line: "Fix one line and let Accolade fix every line like it"},
		{run: "file.saveDraft", also: []string{"file.drafts", "collab.compare", "compare.", "set.draftsSide", "file.mergeChanges"}, line: "Save a draft before the big rewrite, and compare any two later"},
		{run: "view.changes", line: "See what changed since you opened the script"},
		{run: "revision.next", also: []string{"revision.", "view.stages"}, line: "Walk through the open items, to-dos, cuts and notes, one by one"},
		{run: "view.tableRead", also: []string{"tableread."}, line: "Hear it: the table read gives each character a voice"},
		{run: "file.share", also: []string{"file.join", "session."}, line: "Write together and follow each other's cursors"},
		{run: "view.statistics", line: "Count the pages, the scenes, and who talks most"},
	},
	"Adapting": {
		{run: "source.choose", also: []string{"source.saveAs", "source.keepInside", "view.source", "source.middle", "source.sections", "file.startScreenplay"},
			line: "Choose a story as the source and keep it beside the script"},
		{run: "source.next", also: []string{"source.previous", "source.goScene"}, line: "Step through the source passage by passage"},
		{run: "source.adapt", line: "Turn a passage into a scene draft to rewrite"},
		{run: "source.linkGuess", also: []string{"source.link", "source.linkUp", "source.update"}, line: "Let Accolade guess which passage a scene came from"},
		{run: "source.cut", line: "Mark a passage as cut, so leaving it out was a choice"},
		{run: "source.learned", line: "See what Accolade learned from the way you adapt"},
		{run: "source.lock", also: []string{"source.unlock", "source.relinkFromHere"}, sample: true, line: "Lock the source, and relink only what changed in it"},
	},
	"Producing": {
		{run: "view.role", also: []string{"view.production"}, line: "Switch between writer and production and see only what you need"},
		{run: "production.lockNumbers", also: []string{"production.unlockNumbers", "production.renumber", "production.omit", "production.restore", "set.lettering"}, sample: true,
			line: "Lock the scene numbers, add a scene, and watch it become 12A"},
		{run: "production.issue", also: []string{"production.withdraw", "production.compare", "set.revisionMarks"}, sample: true, line: "Issue a draft in its colour; the next changes get their stars"},
		{run: "production.lockPages", also: []string{"production.unlockPages", "production.revisedPages", "set.continueds"}, sample: true, line: "Lock the pages and print only the revised ones"},
		{run: "production.tag", also: []string{"production.untag", "production.showTags"}, line: "Tag the props, the cast and the wardrobe right in the text"},
		{run: "production.breakdown", also: []string{"production.nextScene", "production.sceneDone", "production.sheets"}, line: "Break the script down scene by scene"},
		{run: "production.stripboard", also: []string{"strips.", "production.addDay"}, line: "Shuffle the shoot on a stripboard"},
		{run: "production.schedule", also: []string{"production.ics", "production.oneLiner", "production.dood"}, line: "Plan the shoot days and send them to your calendar"},
		{run: "production.reports", line: "Print the reports a first AD asks for"},
	},
	"Exporting": {
		{run: "file.export", also: []string{"file.pageSetup", "set.sceneNumbers"}, line: "Export to PDF, Final Draft, Word, HTML or an e-book"},
		{run: "file.saveAs", line: "Open a screenplay PDF and keep writing it in Celtx from 2010"},
		{run: "file.print", line: "Print it, title page and all"},
		{run: "file.exportWriter", also: []string{"file.exportShooting"}, line: "Export the writer's version and the shooting version from one script"},
		{run: "file.exportClean", line: "Hand over a clean Fountain, without the notes and marks"},
		{run: "file.exportTreatment", line: "Turn the synopses into a treatment, ready to send"},
		{run: "file.exportOutline", line: "Take the outline to Workflowy or OmniOutliner"},
		{run: "file.exportBoards", line: "Give an editor the storyboard's timing as OTIO"},
		{run: "file.addSettings", line: "Keep a script's Lexington settings as your own"},
	},
}

// funQuiet are the commands with no line: the plain ones, and the help's
// own pages ("x." is all of x.*).
var funQuiet = []string{
	"file.open", "file.openFolder", "file.recent", "file.save", "file.rename", "file.undoRename", "app.preferences",
	"edit.undo", "edit.redo", "edit.find", "edit.replace", "edit.findReplace", "edit.findNext", "edit.findPrevious",
	"edit.cut", "edit.copy", "edit.paste", "edit.selectAll", "format.bold", "format.italic", "format.underline",
	"view.documentType", "view.nextPane", "view.previousPane", "view.dock", "view.help",
	"view.paneTop", "view.paneBottom", "view.paneMiddle", "view.paneSide", "view.midNext", "view.midPrev",
	"help.",
}

// funMatch reports whether id is named in list ("x." for all of x.*).
func funMatch(list []string, id string) bool {
	return slices.ContainsFunc(list, func(s string) bool { return s == id || strings.HasSuffix(s, ".") && strings.HasPrefix(id, s) })
}

// funLineOf is the line standing for command id, if any.
func funLineOf(id string) (funLine, bool) {
	for _, m := range funMoments {
		for _, l := range funLines[m] {
			if l.run == id || funMatch(l.also, id) {
				return l, true
			}
		}
	}
	return funLine{}, false
}

// funUnlined are the commands with neither a line nor a place in funQuiet.
func funUnlined() []*command {
	var out []*command
	for _, c := range commands().commands {
		if _, ok := funLineOf(c.id); !ok && !funMatch(funQuiet, c.id) {
			out = append(out, c)
		}
	}
	return out
}

// funPage is the page: with links that do each thing in the app, plain
// for the web (web).
func funPage(web bool) string {
	var b strings.Builder
	b.WriteString("# Fun Things to Try\n\n")
	if web {
		b.WriteString("Made from Accolade's own list of commands, so nothing is left out. In the app each line is a link that does it: on your script, or on the sample when that suits better.\n")
	} else {
		b.WriteString("Made from Accolade's own list of commands, so nothing is left out. Each line does what it says: on your script, or on a fresh copy of the sample when that suits better.\n")
	}
	item := func(id, text string) {
		c := cmd(id)
		label, keys := strings.TrimSuffix(c.label, "…"), strings.Join(commandKeys(c), ", ")
		if id == funSamplePDF {
			keys = strings.Join(commandKeys(cmd("file.open")), ", ") + ", or drop it on the window"
		}
		if web {
			b.WriteString("- " + text + " (" + label + ")")
		} else {
			b.WriteString("- [" + text + "](cmd:" + url.PathEscape(id) + ") (" + label + ")")
		}
		if keys != "" {
			b.WriteString(" — **" + keys + "**")
		}
		b.WriteString("\n")
	}
	for _, m := range funMoments {
		b.WriteString("\n## " + m + "\n\n")
		for _, l := range funLines[m] {
			item(l.run, l.line)
		}
	}
	if rest := funUnlined(); len(rest) > 0 { // until fun_test.go has its line
		b.WriteString("\n## And More\n\n")
		for _, c := range rest {
			item(c.id, strings.TrimSuffix(c.label, "…"))
		}
	}
	b.WriteString("\nThe command palette (**" + strings.Join(commandKeys(cmd("app.palette")), ", ") + "**) finds every one of these by name, and the [Keyboard Shortcuts](shortcuts.md) list every key.\n")
	return b.String()
}

// tryFun does a line's command: here if it can be given now, else on a
// fresh copy of the sample (in this window if it holds nothing of its
// own); a Markdown document's on a new Markdown document.
func (w *MainWindow) tryFun(id string) {
	if id == funSamplePDF {
		w.openSamplePDF()
		return
	}
	c, ok := commands().byID[id]
	if !ok {
		return
	}
	l, _ := funLineOf(id)
	env := w.cmdEnv()
	e := env
	e.writer = false
	onSample := strings.HasPrefix(filepath.Base(w.currentFile), sampleName)
	if c.canRun(e) && (!l.sample || onSample) {
		w.runByName(c, nil, !c.applies(env))
		return
	}
	markdown := !c.appliesForRole(cmdEnv{}) && c.appliesForRole(cmdEnv{markdown: true})
	p := ""
	if !markdown {
		var err error
		if p, err = sampleCopy(samplesDir()); err != nil {
			dialog.ShowError(fmt.Errorf("could not open the sample: %v", err), w.fyneWindow)
			return
		}
	}
	w.withTarget(func(t *MainWindow) {
		if markdown {
			t.NewMarkdownFile()
		} else if err := t.LoadFile(p); err != nil {
			dialog.ShowError(fmt.Errorf("could not open the sample: %v", err), t.fyneWindow)
			return
		}
		t.runByName(c, nil, !c.applies(t.cmdEnv()))
	})
}

// samplePDFPath is where the sample's PDF is printed.
func samplePDFPath() string { return filepath.Join(samplesDir(), sampleName+".pdf") }

// openSamplePDF is the first line: the sample, printed by Lexington as
// its writer's version, then opened as any PDF is (read back in).
func (w *MainWindow) openSamplePDF() {
	p, err := sampleCopy(samplesDir())
	if err != nil {
		dialog.ShowError(fmt.Errorf("could not open the sample: %v", err), w.fyneWindow)
		return
	}
	w.withTarget(func(t *MainWindow) {
		if err := t.LoadFile(p); err != nil {
			dialog.ShowError(fmt.Errorf("could not open the sample: %v", err), t.fyneWindow)
			return
		}
		text, _ := t.writerVersion()
		job, out := t.writerJob(t.pageSetup(), "pdf", text, nil), samplePDFPath()
		t.setStatus("Lexington prints the sample as a PDF, then Accolade reads it back…")
		inBackground(func() error { return exportAs(text, out, "pdf", job) }, func(err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not print the sample: %v", err), t.fyneWindow)
				return
			}
			t.hasChanges = false // the sample as it came: the PDF takes its place
			t.openPDF(out)
		})
	})
}

// funCommands are the page's own: File > New from a PDF….
func funCommands() []*command {
	return []*command{
		{id: "file.newFromPDF", label: "New from a PDF…", icon: theme.UploadIcon, run: (*MainWindow).NewFromPDF},
	}
}

// NewFromPDF is File > New from a PDF…: a screenplay PDF picked, opened
// as any PDF is (a novel's offered as Markdown).
func (w *MainWindow) NewFromPDF() {
	w.choosePDF(kindScreenplay, func(p string) { w.withTarget(func(t *MainWindow) { t.openPDF(p) }) })
}

// The drop line: while the window holds nothing of its own, a calm line
// over the status bar says a PDF can be dropped on it.

// dropLineText is its text: the browser has no drop (Open a PDF… is there).
func dropLineText() string {
	if inBrowser {
		return "Got a screenplay as a PDF? Open it and keep writing"
	}
	return "Got a screenplay as a PDF? Drop it here"
}

func (w *MainWindow) newDropLine() *fyne.Container {
	l := widget.NewLabel(dropLineText())
	l.Importance = widget.LowImportance
	l.Truncation = fyne.TextTruncateEllipsis
	open := widget.NewButton("Open a PDF…", w.NewFromPDF)
	open.Importance = widget.LowImportance
	w.dropLine = container.NewBorder(nil, nil, nil, open, l)
	w.dropLine.Hide()
	return w.dropLine
}

// refreshDropLine shows the drop line while the window holds nothing of
// its own.
func (w *MainWindow) refreshDropLine() {
	if w.dropLine == nil {
		return
	}
	empty := w.currentFile == "" && w.shared == nil && w.fixupFrom == "" && w.textEditor.Text() == w.filledWith
	if empty == w.dropLine.Visible() {
		return
	}
	if empty {
		w.dropLine.Show()
	} else {
		w.dropLine.Hide()
	}
}

// welcomeOnce says, the first time Accolade starts, where its fun is.
func (w *MainWindow) welcomeOnce() {
	if w.settings == nil || w.settings.GetBoolean("welcomed") {
		return
	}
	w.settings.SetBoolean("welcomed", true)
	w.setStatus("Welcome to Accolade. Help > Fun Things to Try shows what it can do, starting with any screenplay PDF")
}
