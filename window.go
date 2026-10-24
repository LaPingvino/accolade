package main

import (
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/tooltip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/pdfin"
)

type MainWindow struct {
	fyneWindow fyne.Window
	app        *Application

	// Core components
	textEditor *editor.ScriptEditor
	editorView *container.ThemeOverride // the editor with its font settings
	headerBar  *HeaderBar
	tooltips   *tooltip.Layer
	searchBar  *SearchBar

	// UI layout
	mainContainer    *container.Split
	sidePanel        *container.Split
	toolbarContainer *fyne.Container
	// Status bar
	statusBar *fyne.Container
	// headerArea is the header bar (the toolbar) with its separator
	headerArea *fyne.Container
	// the sidebar beside the editor: the outline (View > Outline) or the
	// notes (View > Notes)
	// vim is the editor's vim mode (vim.go), on with the setting
	vim *vim
	// the ex command line (ex_window.go)
	exState  exState
	exBar    *fyne.Container
	exEntry  *exLine
	exLabel  *widget.Label
	exPrompt func(string) // the line is a prompt's (helix s), not a command
	exVisual [2]int
	// the card view in the editor's place (cards.go)
	cardsVisible bool
	// the others' changes to a shared script (changes.go)
	changes        []change
	changesVisible bool
	changesPanel   *changesPanel
	cards          *cardsView
	editorFolds    map[string]bool // folded in the editor, by foldKey (editor_folds.go)
	foldState      int             // where Shift+Tab is: foldShown, foldOverview or foldContents
	// language is the script's language ("en", "eo", ...): its scene
	// headings (language.go)
	language string
	// format is how the script's file stores it (encoding.go)
	format fileFormat
	// imported is closed when the PDF import under way is in
	imported       chan struct{}
	outline        *outlinePanel
	outlineVisible bool
	notes          *notesPanel
	notesVisible   bool

	// File management
	currentFile string
	hasChanges  bool
	// filledWith is the text the window was filled with and has no file
	// for: the sample script, or an import; while the text is that,
	// opening a file replaces it (openPath)
	filledWith string
	// shared: the script edited together with peers (collab_window.go)
	shared       *shared
	isFullscreen bool
	// menuShortcuts are the menu's shortcuts in full screen, without the
	// menu bar: on the canvas, and for the editor (menuShortcut)
	menuShortcuts map[string]menuShortcut

	// Settings
	settings *Settings

	// Find/Replace
	findVisible bool

	// Name offered by Save As for an imported (converted) document
	suggestedName string

	// Auto-save (autosave.go)
	autoSaveTimer *time.Timer
	autoSaveDelay time.Duration // overrides the setting; for tests

	statusLabel *widget.Label
	statsLabel  *widget.Label

	// Lexington integration
	lexParser *LexingtonParser

	// Current cursor position for element detection
	currentElement string
}

func NewMainWindow(app *Application) *MainWindow {
	window := &MainWindow{
		fyneWindow:   app.fyneApp.NewWindow("Accolade"),
		app:          app,
		hasChanges:   false,
		isFullscreen: false,
		findVisible:  false,
		lexParser:    NewLexingtonParser(),
	}

	window.settings = GetSettings() // shared with the preferences dialog
	window.setupUI()
	window.setupShortcuts()
	window.setupCallbacks()
	window.updateTitle()
	window.updateCurrentElement()
	window.updateStatusBar()

	return window
}

func (w *MainWindow) setupUI() {
	// Set window properties
	w.fyneWindow.SetTitle("Accolade")
	w.fyneWindow.Resize(fyne.NewSize(1000, 600))
	w.fyneWindow.CenterOnScreen()

	// Create main text editor with Courier Prime font
	w.textEditor = newScriptEditor(w)
	w.textEditor.OnCursorChanged = w.onCursorChanged

	// Add some initial content - start with a title page
	w.textEditor.SetText(defaultScript(time.Now()))
	w.filledWith = w.textEditor.Text()

	// Apply Courier Prime font for screenplay formatting
	// the editor scrolls to keep the cursor in view; the theme override
	// carries the font settings
	w.editorView = container.NewThemeOverride(editor.NewPageScroll(w.textEditor, 0), newEditorTheme("", 0))

	// Create header bar with toolbar
	w.tooltips = tooltip.NewLayer()
	w.headerBar = NewHeaderBar(w)

	// Create search bar (initially hidden)
	w.searchBar = NewSearchBar(w)

	// Create toolbar container
	// the toolbar (the header bar) can be hidden, the search bar stays
	w.headerArea = container.NewVBox(w.headerBar.container, widget.NewSeparator())
	w.toolbarContainer = container.NewVBox(
		w.headerArea,
		w.searchBar.container,
	)

	// Create status bar with element indicator
	// Status bar: messages on the left (saved, auto-saved), the element
	// under the cursor and the script's statistics on the right
	small := func(text string) *widget.Label {
		l := widget.NewLabel(text)
		l.SizeName = theme.SizeNameCaptionText
		l.Importance = widget.LowImportance
		return l
	}
	w.statusLabel = small("")
	w.statsLabel = small("")
	w.statsLabel.Alignment = fyne.TextAlignTrailing
	w.statsLabel.Truncation = fyne.TextTruncateEllipsis // a narrow window shows what fits
	w.statusBar = container.NewVBox(
		widget.NewSeparator(),
		container.NewBorder(nil, nil, w.statusLabel, nil, w.statsLabel),
	)

	w.relayout()

	// Apply initial settings
	w.applySettings()
}

func (w *MainWindow) setupShortcuts() {
	// Text editor shortcuts
	w.textEditor.OnChanged = w.onTextChanged
	w.setLanguage("")                      // the preferences' until a file says otherwise
	w.fyneWindow.SetOnDropped(w.onDropped) // drop a script or a PDF to open it

	// Lines are formatted when completed with Enter (editor.go)

	// Keyboard shortcuts are attached to the main menu items (see menus.go)
	w.fyneWindow.SetMainMenu(w.buildMainMenu())
	// F6 and Shift+F6: the next or previous pane (editor, outline, notes,
	// search), whatever has the focus
	if c, ok := w.fyneWindow.Canvas().(desktop.KeyPreviewCanvas); ok {
		c.SetOnKeyPreview(func(key fyne.KeyName, mod fyne.KeyModifier) bool {
			keyArrived()
			if key != fyne.KeyF6 || (mod != 0 && mod != fyne.KeyModifierShift) {
				return false
			}
			if mod == fyne.KeyModifierShift {
				w.cyclePane(-1)
			} else {
				w.cyclePane(1)
			}
			return true
		})
	}
}

func (w *MainWindow) setupCallbacks() {
	// Window close callback
	w.fyneWindow.SetCloseIntercept(func() {
		if w.hasChanges {
			dialog.ShowConfirm(
				"Unsaved Changes",
				"You have unsaved changes. Are you sure you want to close?",
				func(confirmed bool) {
					if confirmed {
						w.fyneWindow.Close()
					}
				},
				w.fyneWindow,
			)
		} else {
			w.fyneWindow.Close()
		}
	})
}

// TODO: Implement file drop handling when needed
// Fyne's drag and drop API may differ from GTK

func (w *MainWindow) applySettings() {
	// Apply theme
	w.app.setColorScheme(w.settings.GetString("theme"))

	// Apply editor settings
	w.applyEditorSettings()

	// Apply window settings

	// Apply font settings
	w.applyFontSettings()
}

func (w *MainWindow) applyEditorSettings() {
	// the script editor always wraps at the window width (screenplays are
	// laid out in columns)
	w.textEditor.SetLineNumbers(w.settings.GetBoolean("show-line-numbers"))
	setVisible(w.headerArea, w.settings.GetBoolean("toolbar-visible"))
	// vim mode switched on starts in normal mode, switched off types again
	if w.vim != nil {
		if on := w.editorKeys() != "standard"; on && w.vim.mode == vimInsert {
			w.vim.setMode(vimNormal)
		} else if !on && w.vim.mode != vimInsert {
			w.vim.mode = vimInsert
		}
	}
	setVisible(w.statusBar, w.settings.GetBoolean("statusbar-visible"))
	// auto-indent (formatting the line on Enter) is read when Enter is pressed (editor.go)
	w.applyScriptFormat()
}

// applyScriptFormat lays the editor out in the script format's columns.
func (w *MainWindow) applyScriptFormat() {
	w.textEditor.Format, w.textEditor.PageColumns = editorFormat(currentScriptFormat())
	w.textEditor.Refresh()
	w.shareMeta("format", w.settings.GetString("script-format"))
}

func setVisible(o fyne.CanvasObject, visible bool) {
	if o == nil {
		return
	}
	if visible {
		o.Show()
	} else {
		o.Hide()
	}
}

func (w *MainWindow) applyFontSettings() {
	w.editorView.Theme = newEditorTheme(w.settings.GetString("font-family"), w.settings.GetInt("font-size"))
	w.editorView.Refresh()
}

// File operations
func (w *MainWindow) LoadFile(filePath string) error {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".fdx", ".pdf":
		return w.importFile(filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// as stored: Windows-1252, a byte order mark, CR LF are kept for saving
	text, format := decodeScript(content)
	w.format = format
	w.setLanguage(languageOf(filePath))
	w.textEditor.SetText(withoutIndents(text, w.sceneStarts()))
	w.currentFile = filePath
	w.suggestedName = ""
	w.rememberRecent(filePath)
	w.hasChanges = false
	w.updateTitle()
	w.updateStatusBar()

	return nil
}

// importFile opens a Final Draft document or a PDF of a screenplay
// converted to Fountain. It has no file of its own until saved, so Save
// asks where to put the .fountain file instead of overwriting the
// original.
func (w *MainWindow) importFile(filePath string) error {
	var text, from string
	var err error
	if strings.EqualFold(filepath.Ext(filePath), ".pdf") {
		w.importPDFAsync(filePath)
		return nil
	} else {
		var f *os.File
		if f, err = os.Open(filePath); err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
		defer f.Close()
		text, err = importFDX(f)
		from = "Final Draft"
	}
	if err != nil {
		return fmt.Errorf("could not import %s: %v", filepath.Base(filePath), err)
	}

	w.showImported(filePath, text, from)
	return nil
}

// showImported puts an imported script in the window, unsaved.
func (w *MainWindow) showImported(filePath, text, from string) {
	w.cancelAutoSave()
	w.format = fileFormat{}
	w.textEditor.SetText(text)
	w.filledWith = w.textEditor.Text()
	w.currentFile = ""
	w.suggestedName = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)) + ".fountain"
	w.hasChanges = true
	w.updateTitle()
	w.updateStatusBar()
	w.setStatus("Imported from " + from)
}

// importPDFAsync reads a PDF in the background (a scanned one needs OCR,
// seconds a page), showing how far it is. Before scanned pages are read,
// it asks what language they are in, showing the one it guesses.
func (w *MainWindow) importPDFAsync(filePath string) {
	// the scene headings' language: the file name's (script.ru.pdf), else
	// the one its headings are in (read with every language's, not the
	// replaced script's)
	lang := languageOf(filePath)
	if scanned, pages, err := pdfin.ScannedPages(filePath); err == nil && scanned > 0 {
		if langs, tesseract := ocrLanguages(); len(langs) > 0 {
			w.askOCRLanguage(filePath, scanned, pages, w.guessOCRLanguage(lang, langs), langs, tesseract, func(chosen string) {
				w.readPDF(filePath, chosen, chosen)
			})
			return
		}
	}
	w.readPDF(filePath, lang, "")
}

// readPDF reads a PDF in the background, its scene headings in lang ("":
// whichever they are in), scanned pages read in ocrLang.
func (w *MainWindow) readPDF(filePath, lang, ocrLang string) {
	scenes := sceneHeaders(lang)
	if lang == "" {
		scenes = allSceneHeaders()
	}
	bar := widget.NewProgressBar()
	label := widget.NewLabel("Reading " + filepath.Base(filePath) + "...")
	// a long scanned script takes minutes to read: Cancel stops it
	ctx, cancel := context.WithCancel(context.Background())
	cancelBtn := widget.NewButton("Cancel", cancel)
	d := dialog.NewCustomWithoutButtons("Importing PDF", container.NewVBox(label, bar, container.NewCenter(cancelBtn)), w.fyneWindow)
	d.SetOnEscape(cancel)
	d.Show()
	done := make(chan struct{})
	w.imported = done
	go func() {
		defer close(done)
		defer cancel()
		text, skipped, err := importPDF(ctx, filePath, scenes, ocrLang, func(done, pages int) {
			fyne.Do(func() {
				bar.SetValue(float64(done) / float64(max(pages, 1)))
				label.SetText(fmt.Sprintf("Read %d of %d pages", done, pages))
				if done == pages {
					label.SetText("Read all pages; putting the screenplay together...")
				}
			})
		})
		fyne.Do(func() {
			d.Hide()
			if ctx.Err() != nil && errors.Is(err, context.Canceled) {
				return // cancelled: nothing to show
			}
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not import %s: %v", filepath.Base(filePath), err), w.fyneWindow)
				return
			}
			named := lang != "" // the file name says it (script.ru.pdf): that first
			if !named {
				lang = headingsLanguage(text, w.settings.GetString("script-language"))
			}
			w.setLanguage(lang)
			w.showImported(filePath, text, "a PDF: check the result, a PDF only shows how the script looked")
			if !named && lang != "en" { // saved, it keeps the language its headings are in
				w.suggestedName = strings.TrimSuffix(w.suggestedName, ".fountain") + "." + lang + ".fountain"
				w.updateTitle()
			}
			if len(skipped) > 0 {
				dialog.ShowInformation("Some pages were left out",
					"These pages could not be read:\n"+strings.Join(skipped, "\n"), w.fyneWindow)
			}
		})
	}()
}

func (w *MainWindow) SaveFile() error {
	if w.currentFile == "" {
		return w.SaveFileAs()
	}
	if err := w.saveToFile(w.currentFile); err != nil {
		return err
	}
	if inBrowser { // kept in the browser, and a copy to download
		offerDownload(w.currentFile)
	}
	return nil
}

func (w *MainWindow) SaveFileAs() error {
	if inBrowser { // the browser asks where, for the download
		name := w.suggestedName
		if w.currentFile != "" {
			name = filepath.Base(w.currentFile)
		}
		if name == "" {
			name = "untitled.fountain"
		}
		p := filepath.Join("/files", name)
		if err := os.MkdirAll("/files", 0o755); err != nil {
			return err
		}
		if err := w.saveToFile(p); err != nil {
			return err
		}
		offerDownload(p)
		w.currentFile, w.suggestedName = p, ""
		w.updateTitle()
		return nil
	}
	fileDialog := dialog.NewFileSave(func(closer fyne.URIWriteCloser, err error) {
		if err != nil || closer == nil {
			return
		}

		filePath := closer.URI().Path()
		// written as Save does, safely: not through the dialog's file,
		// which truncates first
		closer.Close()
		if err := w.saveToFile(filePath); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save file: %v", err), w.fyneWindow)
			return
		}

		w.currentFile = filePath
		w.suggestedName = ""
		w.rememberRecent(filePath)
		if l := languageOf(filePath); l != "" {
			w.setLanguage(l)
		}
		w.hasChanges = false
		w.updateTitle()

		dialog.ShowInformation("File Saved", fmt.Sprintf("File saved to %s", filepath.Base(filePath)), w.fyneWindow)
	}, w.fyneWindow)

	// Set default filename
	if w.currentFile != "" {
		fileDialog.SetFileName(filepath.Base(w.currentFile))
	} else if w.suggestedName != "" {
		fileDialog.SetFileName(w.suggestedName)
	} else {
		fileDialog.SetFileName("untitled.fountain")
	}

	// TODO: Set file filter when Fyne supports it
	// fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".fountain", ".spmd"}))

	fileDialog.Show()
	return nil
}

func (w *MainWindow) saveToFile(filePath string) error {
	data, utf8Instead := encodeScript(w.textEditor.Text(), w.format)
	if err := writeFileAtomic(filePath, data); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	if utf8Instead {
		w.format.Windows1252 = false
		w.setStatus("Saved as UTF-8: Windows-1252 cannot hold every character now in the script")
	}

	w.cancelAutoSave()
	w.hasChanges = false
	w.updateTitle()
	return nil
}

func (w *MainWindow) OpenFile() {
	if inBrowser {
		pickFile(w.openPath)
		return
	}
	fileDialog := dialog.NewFileOpen(func(closer fyne.URIReadCloser, err error) {
		if err != nil || closer == nil {
			return
		}

		filePath := closer.URI().Path()
		defer closer.Close()
		w.openPath(filePath)
	}, w.fyneWindow)

	// TODO: Set file filter when Fyne supports it
	// fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".fountain", ".spmd", ".txt"}))

	fileDialog.Show()
}

// openPath opens a file (a script, a Final Draft document, a PDF): in
// this window if it holds only what it was filled with (the sample
// script, an import not edited since), else in a new one, so that
// nothing unsaved is replaced.
func (w *MainWindow) openPath(path string) {
	w.withTarget(func(target *MainWindow) {
		if err := target.LoadFile(path); err != nil {
			dialog.ShowError(fmt.Errorf("failed to open %s: %v", filepath.Base(path), err), target.fyneWindow)
		}
	})
}

// withTarget gives the window something new goes in: this one if it
// holds nothing of its own (no file, no edits beyond what it was filled
// with), else a new window; in the browser, which has one window, this
// one after asking.
func (w *MainWindow) withTarget(then func(*MainWindow)) {
	replaceable := w.currentFile == "" && (!w.hasChanges || w.textEditor.Text() == w.filledWith)
	switch {
	case replaceable || w.app == nil:
		then(w)
	case inBrowser:
		dialog.ShowConfirm("Replace the Script?",
			"The script here has changes not saved. Replace it? (Save first to keep it.)",
			func(ok bool) {
				if ok {
					then(w)
				}
			}, w.fyneWindow)
	default:
		then(w.app.newWindow())
	}
}

// onDropped opens the files dropped on the window, as Highland does.
func (w *MainWindow) onDropped(_ fyne.Position, uris []fyne.URI) {
	for _, u := range uris {
		if u.Scheme() == "file" {
			w.openPath(u.Path())
		}
	}
}

func (w *MainWindow) NewFile() {
	if w.hasChanges {
		dialog.ShowConfirm(
			"Unsaved Changes",
			"You have unsaved changes. Create a new file anyway?",
			func(confirmed bool) {
				if confirmed {
					w.createNewFile()
				}
			},
			w.fyneWindow,
		)
	} else {
		w.createNewFile()
	}
}

// editorKeys is how the editor's keys work: "standard", "vim" or
// "helix" (an older "vim-mode" setting is vim).
func (w *MainWindow) editorKeys() string {
	if w == nil || w.settings == nil {
		return "standard"
	}
	switch k := w.settings.GetString("editor-keys"); k {
	case "vim", "helix":
		return k
	}
	if w.settings.GetBoolean("vim-mode") {
		return "vim"
	}
	return "standard"
}

// setLanguage makes the script's language lang ("" for the one in the
// preferences): its scene headings in the editor and exports.
func (w *MainWindow) setLanguage(lang string) {
	if lang == "" && w.settings != nil {
		lang = w.settings.GetString("script-language")
	}
	if sceneConf[lang] == nil {
		lang = "en"
	}
	w.language = lang
	if w.headerBar != nil && w.headerBar.place != nil {
		// the place button in the language: BIN./BUI. in Dutch
		if _, _, both, ok := placeWords(sceneHeaders(lang)); ok {
			w.headerBar.place.SetText(both)
		}
	}
	if w.textEditor != nil {
		w.textEditor.SceneStarts = sceneStarts(lang)
		w.textEditor.Refresh()
	}
	w.shareMeta("language", lang)
}

// sceneStarts are what scene headings start with in the script's
// language (English without a window).
func (w *MainWindow) sceneStarts() []string {
	if w == nil {
		return nil
	}
	return sceneStarts(w.language)
}

// newScriptTemplate is what a new script starts with: a title page and
// a scene to write over.
func newScriptTemplate() string {
	return strings.Join([]string{
		"Title: UNTITLED SCREENPLAY",
		"Credit: Written by",
		"Author: Your Name",
		"Draft date: " + time.Now().Format("January 2, 2006"),
		"Contact:",
		"    Your Name",
		"    Your Address",
		"    Your Phone",
		"    Your Email",
		"",
		"FADE IN:",
		"",
		"INT. LIVING ROOM - DAY",
		"",
		"A simple room with basic furniture. Light streams through the windows.",
		"",
		"JOHN sits at a desk, typing on a laptop. He looks frustrated.",
		"",
		"JOHN",
		"(sighing)",
		"This screenplay isn't writing itself.",
		"",
		"He takes a sip of coffee and continues typing.",
		"",
		"> FADE OUT.",
		"",
	}, "\n")
}

func (w *MainWindow) createNewFile() {
	w.format = fileFormat{} // a new script is UTF-8
	w.textEditor.SetText(newScriptTemplate())
	w.currentFile = ""
	w.hasChanges = false
	w.updateTitle()
	w.updateStatusBar()
}

// UI operations
func (w *MainWindow) Show() {
	w.fyneWindow.Show()
}

func (w *MainWindow) IsEmpty() bool {
	return strings.TrimSpace(w.textEditor.Text()) == ""
}

func (w *MainWindow) HasUnsavedChanges() bool {
	return w.hasChanges
}

// kept is the script in a window as the web version keeps it, for after
// a reload (browser_js.go).
type kept struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
	File     string `json:"file,omitempty"` // in the browser's memory (kept too)
	Name     string `json:"name,omitempty"` // an import's name to save it as
}

// restore puts a kept script back: its file again, if it still has it,
// with the changes not saved.
func (w *MainWindow) restore(d kept) {
	if d.File != "" && w.LoadFile(d.File) == nil {
		if d.Text != w.textEditor.Text() {
			w.textEditor.SetText(d.Text)
		}
		w.setStatus("Your script, as this browser kept it")
		return
	}
	w.setLanguage(d.Language)
	w.textEditor.SetText(d.Text)
	w.suggestedName = d.Name
	w.updateTitle()
	w.setStatus("Your script not saved yet, as this browser kept it")
}

func (w *MainWindow) onTextChanged(text string) {
	w.hasChanges = true
	if inBrowser {
		keepDraft(kept{Text: text, Language: w.language, File: w.currentFile, Name: w.suggestedName})
	}
	w.shareEdit(text)
	w.updateTitle()
	w.updateStatusBar()
	w.updateCurrentElement()
	if w.outlineVisible && w.outline != nil {
		w.outline.update(text)
	}
	if w.cardsVisible && w.cards != nil {
		w.cards.update(text)
	}
	if w.notesVisible && w.notes != nil {
		w.notes.update(text)
	}
	if w.searchBar != nil && w.searchBar.IsVisible() {
		w.searchBar.updateSearch() // the matches move with the text
	}

	w.scheduleAutoSave()
}

func (w *MainWindow) updateTitle() {
	title := "Accolade"

	if w.currentFile != "" {
		filename := filepath.Base(w.currentFile)
		if w.hasChanges {
			title = "• " + filename + " - Accolade"
		} else {
			title = filename + " - Accolade"
		}
	} else if w.suggestedName != "" {
		title = "• " + w.suggestedName + " (not saved yet) - Accolade"
	} else if w.hasChanges {
		title = "• Untitled - Accolade"
	}

	w.fyneWindow.SetTitle(title)
	if w.headerBar != nil {
		name := strings.TrimSuffix(title, " - Accolade")
		if name == "Accolade" {
			name = "Untitled"
		}
		w.headerBar.SetTitle(name)
	}
}

func (w *MainWindow) setStatus(text string) {
	if w.statusLabel != nil {
		w.statusLabel.SetText(text)
	}
}

func (w *MainWindow) updateStatusBar() {
	if w.statsLabel == nil {
		return // called while the window is being built
	}
	text := computeStats(w.textEditor.Text(), w.textEditor.CursorOffset(), w.sceneStarts()).String()
	if w.currentElement != "" && w.currentElement != "empty" {
		text = w.currentElement + " · " + text
	}
	if s := w.sharingStatus(); s != "" {
		text = s + " · " + text
	}
	w.statsLabel.SetText(text)
}

func (w *MainWindow) onCursorChanged() {
	w.updateCurrentElement()
	w.updateStatusBar() // the scene under the cursor
	// what Tab would complete, or nothing once it no longer applies
	if h := completionHint(w.textEditor.Text(), w.textEditor.CursorOffset(), w.sceneStarts()); h != "" {
		w.setStatus(h)
	} else if w.statusLabel != nil && strings.HasPrefix(w.statusLabel.Text, "Tab: ") {
		w.setStatus("")
	}
}

func (w *MainWindow) updateCurrentElement() {
	text := w.textEditor.Text()
	if text == "" {
		w.currentElement = "empty"
		return
	}

	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	kinds := syntax.ClassifyTyping(strings.Split(text, "\n"), w.sceneStarts(), line)
	w.currentElement = elementNames[kinds[min(line, len(kinds)-1)]]
	if w.headerBar != nil {
		w.headerBar.showElement(w.currentElement)
	}
}

// elementNames are the elements' names in the status bar.
var elementNames = map[syntax.Kind]string{
	syntax.Action: "Action", syntax.Empty: "Empty", syntax.TitlePage: "Title Page",
	syntax.SceneHeading: "Scene Heading", syntax.Character: "Character",
	syntax.Parenthetical: "Parenthetical", syntax.Dialogue: "Dialogue",
	syntax.Transition: "Transition", syntax.Centered: "Centered", syntax.Section: "Section",
	syntax.Synopsis: "Synopsis", syntax.Note: "Note", syntax.PageBreak: "Page Break",
	syntax.Lyrics: "Lyrics",
}

// previewPDF prints the script as the PDF export does, with the export
// settings, to a temporary file, and opens it in the PDF viewer.
func (w *MainWindow) previewPDF() {
	w.pdfOrPrint(w.textEditor.Text(), w.makePreviewPDF)
}

func (w *MainWindow) makePreviewPDF() {
	job := exportJob{
		format:        w.settings.GetString("script-format"),
		page:          w.settings.GetString("page-size"),
		omitTitlePage: !w.settings.GetBoolean("include-title-page"),
		numberScenes:  w.settings.GetBoolean("fountain-scene-numbers"),
		scenes:        sceneHeaders(w.language),
	}
	dir, err := os.MkdirTemp("", "accolade-preview-")
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	name := "untitled.fountain"
	if w.currentFile != "" {
		name = filepath.Base(w.currentFile)
	} else if w.suggestedName != "" {
		name = w.suggestedName
	}
	out := filepath.Join(dir, strings.TrimSuffix(name, filepath.Ext(name))+".pdf")
	content := w.textEditor.Text()
	w.setStatus("Preparing the PDF...")
	go func() {
		err := exportAs(content, out, "pdf", job)
		fyne.Do(func() {
			if err != nil {
				w.setStatus("")
				dialog.ShowError(fmt.Errorf("could not make the PDF: %v", err), w.fyneWindow)
				return
			}
			w.setStatus("PDF preview shown")
			openFile(out)
		})
	}()
}

// printFile prints an HTML file: in the browser's print dialog (the web
// version), or opened in the system's browser to print from there.
var printFile = func(path string) { openFile(path) }

// printScript prints the script as Lexington's HTML lays it out, through
// the browser: its fonts have every script, and its print dialog saves a
// PDF too.
func (w *MainWindow) printScript() {
	dir, err := os.MkdirTemp("", "accolade-print-")
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	out := filepath.Join(dir, "script.html")
	job := exportJob{
		format:        w.settings.GetString("script-format"),
		page:          w.settings.GetString("page-size"),
		omitTitlePage: !w.settings.GetBoolean("include-title-page"),
		numberScenes:  w.settings.GetBoolean("fountain-scene-numbers"),
		scenes:        sceneHeaders(w.language),
	}
	content := w.textEditor.Text()
	w.setStatus("Preparing to print...")
	go func() {
		err := exportAs(content, out, "html", job)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not prepare the script to print: %v", err), w.fyneWindow)
				return
			}
			w.setStatus("Print: choose a printer, or Save as PDF")
			printFile(out)
		})
	}()
}

// openFile opens a file in the system's viewer for that kind of file.
var openFile = func(path string) {
	if u, err := url.Parse("file://" + filepath.ToSlash(path)); err == nil {
		fyne.CurrentApp().OpenURL(u)
	}
}

// setContent shows content in the window, with the tooltip layer over it.
func (w *MainWindow) setContent(content fyne.CanvasObject) {
	w.fyneWindow.SetContent(container.NewStack(content, w.tooltips))
}

// toggleOutline shows or hides the outline beside the editor.
func (w *MainWindow) toggleOutline() {
	if w.outline == nil {
		w.outline = newOutlinePanel(w)
	}
	w.outlineVisible = !w.outlineVisible
	if w.outlineVisible {
		w.outline.update(w.textEditor.Text())
	}
	w.relayout()
}

// toggleNotes shows or hides the script's notes beside the editor.
func (w *MainWindow) toggleNotes() {
	if w.notes == nil {
		w.notes = newNotesPanel(w)
	}
	w.notesVisible = !w.notesVisible
	if w.notesVisible {
		w.notes.update(w.textEditor.Text())
		w.relayout()
		w.fyneWindow.Canvas().Focus(w.notes.entry)
		return
	}
	w.relayout()
}

// relayout puts the window together: the toolbar and the status bar,
// the outline and notes on the left if shown, the editor.
func (w *MainWindow) relayout() {
	var center fyne.CanvasObject = w.editorView
	if w.cardsVisible && w.cards != nil {
		center = w.cards.box
	}
	var panels []fyne.CanvasObject // one above the other
	if w.outlineVisible && w.outline != nil {
		panels = append(panels, w.outline.box)
	}
	if w.notesVisible && w.notes != nil {
		panels = append(panels, w.notes.box)
	}
	if w.changesVisible && w.changesPanel != nil {
		panels = append(panels, w.changesPanel.box)
	}
	var side fyne.CanvasObject
	for i := len(panels) - 1; i >= 0; i-- {
		if side == nil {
			side = panels[i]
		} else {
			side = container.NewVSplit(panels[i], side)
		}
	}
	var left fyne.CanvasObject
	if side != nil {
		left = container.NewHBox(side, widget.NewSeparator())
	}
	// the objects in the order they are seen, which is the order Tab and
	// screen readers go through them (NewBorder puts the centre first:
	// Tab went from the editor to the toolbar, the side panel came last)
	objects := []fyne.CanvasObject{w.toolbarContainer}
	if left != nil {
		objects = append(objects, left)
	}
	objects = append(objects, center, w.statusBar)
	var content fyne.CanvasObject = container.New(layout.NewBorderLayout(w.toolbarContainer, w.statusBar, left, nil), objects...)
	if w.isFullscreen {
		// the way back, in sight (not only Ctrl+Shift+F)
		leave := widget.NewButtonWithIcon("Leave full screen", theme.ViewRestoreIcon(), w.toggleFullscreen)
		leave.Importance = widget.LowImportance
		content = container.NewStack(content, container.NewVBox(container.NewHBox(layout.NewSpacer(), leave)))
	}
	w.setContent(content)
}

// Find/Replace operations
func (w *MainWindow) showFindReplace() {
	w.findVisible = true
	w.searchBar.SetReplaceMode(true)
	w.searchBar.Show()
}

func (w *MainWindow) showFind() {
	w.findVisible = true
	w.searchBar.SetReplaceMode(false)
	w.searchBar.Show()
}

func (w *MainWindow) hideFindReplace() {
	w.findVisible = false
	w.searchBar.Hide()
}

func (w *MainWindow) findNext() {
	w.searchBar.findNext()
}

func (w *MainWindow) findPrevious() {
	w.searchBar.findPrevious()
}

func (w *MainWindow) replaceOne() {
	w.searchBar.replaceOne()
}

func (w *MainWindow) replaceAll() {
	w.searchBar.replaceAll()
}

// toggleFullscreen switches the distraction-free full screen: the script
// (and the outline and notes, if open), without the menu bar, toolbar
// and status bar; a button in the corner leaves it, as Ctrl+Shift+F
// does. The menu's shortcuts keep working.
func (w *MainWindow) toggleFullscreen() {
	w.isFullscreen = !w.isFullscreen
	w.fyneWindow.SetFullScreen(w.isFullscreen)
	browserFullscreen(w.isFullscreen, func() { // the browser left it (Escape)
		if w.isFullscreen {
			w.toggleFullscreen()
		}
	})
	if w.isFullscreen {
		setVisible(w.headerArea, false)
		setVisible(w.statusBar, false)
		w.fyneWindow.SetMainMenu(nil)
	} else {
		w.applyEditorSettings() // the toolbar and status bar as set
	}
	w.refreshMenu()
	w.relayout()
	w.fyneWindow.Canvas().Focus(w.textEditor)
}

// refreshMenu builds the menu again (the recent files changed): in the
// menu bar, or in full screen only its shortcuts.
func (w *MainWindow) refreshMenu() {
	c := w.fyneWindow.Canvas()
	for _, sc := range w.menuShortcuts {
		c.RemoveShortcut(sc.shortcut)
	}
	w.menuShortcuts = nil
	if !w.isFullscreen {
		w.fyneWindow.SetMainMenu(w.buildMainMenu())
		return
	}
	w.menuShortcuts = map[string]menuShortcut{}
	var add func(items []*fyne.MenuItem)
	add = func(items []*fyne.MenuItem) {
		for _, item := range items {
			if item.ChildMenu != nil {
				add(item.ChildMenu.Items)
			}
			if item.Shortcut == nil || item.Action == nil {
				continue
			}
			action := item.Action
			c.AddShortcut(item.Shortcut, func(fyne.Shortcut) { action() })
			w.menuShortcuts[item.Shortcut.ShortcutName()] = menuShortcut{item.Shortcut, action}
		}
	}
	for _, m := range w.buildMainMenu().Items {
		add(m.Items)
	}
}

// menuShortcut is a menu item's shortcut and action.
type menuShortcut struct {
	shortcut fyne.Shortcut
	action   func()
}

// menuShortcut runs the menu item with shortcut s in full screen (the
// editor gets shortcuts before the canvas does); it reports whether it did.
func (w *MainWindow) menuShortcut(s fyne.Shortcut) bool {
	if sc, ok := w.menuShortcuts[s.ShortcutName()]; ok {
		sc.action()
		return true
	}
	return false
}

// Export functionality
func (w *MainWindow) showExportDialog() {
	dialog := NewExportDialog(w)
	dialog.Show()
}

// Title page functionality
func (w *MainWindow) showTitlePageDialog() {
	dialog := NewTitlePageDialog(w)
	dialog.Show()
}

// Helper functions
func countWords(text string) int {
	if text == "" {
		return 0
	}
	words := strings.Fields(text)
	return len(words)
}

func isCharacterName(line string) bool {
	// Character names are typically ALL CAPS and don't start with common scene indicators
	if line == strings.ToUpper(line) && !strings.HasPrefix(line, "INT.") && !strings.HasPrefix(line, "EXT.") {
		// Additional checks to avoid false positives
		if !strings.Contains(line, " TO ") && !strings.HasSuffix(line, ":") {
			return len(strings.TrimSpace(line)) > 0 && len(strings.TrimSpace(line)) < 50
		}
	}
	return false
}

// defaultScript is the text a new window starts with: a title page to fill
// in and the first lines of a script.
func defaultScript(now time.Time) string {
	return strings.Join([]string{
		"Title: UNTITLED SCREENPLAY",
		"Credit: Written by",
		"Author: Your Name",
		"Draft date: " + now.Format("January 2, 2006"),
		"Contact:",
		"    Your Name",
		"    Your Address",
		"    Your Phone",
		"    Your Email",
		"",
		"FADE IN:",
		"",
		"INT. LIVING ROOM - DAY",
		"",
		"A simple room with basic furniture. Light streams through the windows.",
		"",
		"JOHN sits at a desk, typing on a laptop. He looks frustrated.",
		"",
		"JOHN",
		"(sighing)",
		"This screenplay isn't writing itself.",
		"",
		"He takes a sip of coffee and continues typing.",
		"",
		"> FADE OUT.",
	}, "\n")
}

// panes are the parts of the window that take the keyboard, in order:
// the editor, and the outline, notes and search bar when shown.
func (w *MainWindow) panes() []fyne.Focusable {
	panes := []fyne.Focusable{w.textEditor}
	// the side panel shows the outline or else the notes (relayout)
	switch {
	case w.outlineVisible && w.outline != nil:
		panes = append(panes, w.outline.list)
	case w.notesVisible && w.notes != nil:
		panes = append(panes, w.notes.entry)
	}
	if w.searchBar != nil && w.searchBar.IsVisible() {
		panes = append(panes, w.searchBar.searchEntry)
	}
	return panes
}

// cyclePane moves the focus to the pane step panes on from the one that
// has it (from anywhere else: the editor's neighbour).
func (w *MainWindow) cyclePane(step int) {
	panes := w.panes()
	at := 0
	for i, p := range panes {
		if w.fyneWindow.Canvas().Focused() == p {
			at = i
		}
	}
	n := len(panes)
	w.fyneWindow.Canvas().Focus(panes[((at+step)%n+n)%n])
}
