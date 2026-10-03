package main

import (
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/tooltip"
)

// HeaderBar is the slim bar at the top of the window: the file actions
// and undo/redo on the left, the document's name in the middle, and
// find, preview, export and preferences on the right, as icon buttons
// with tooltips. Everything else is in the menus.
type HeaderBar struct {
	window    *MainWindow
	container *fyne.Container

	newButton, openButton, saveButton *tooltip.Button
	undoButton, redoButton            *tooltip.Button
	findButton, previewButton         *tooltip.Button
	exportButton, preferencesButton   *tooltip.Button
	titleLabel                        *widget.Label
}

func NewHeaderBar(window *MainWindow) *HeaderBar {
	hb := &HeaderBar{window: window}
	tips := window.tooltips
	button := func(icon fyne.Resource, tip string, action func()) *tooltip.Button {
		b := tooltip.NewButton(icon, tip, action, tips)
		b.Importance = widget.LowImportance
		return b
	}

	hb.newButton = button(theme.DocumentCreateIcon(), "New"+keys("N"), window.NewFile)
	hb.openButton = button(theme.FolderOpenIcon(), "Open"+keys("O"), window.OpenFile)
	hb.saveButton = button(theme.DocumentSaveIcon(), "Save"+keys("S"), window.reportErr(window.SaveFile))
	hb.undoButton = button(theme.ContentUndoIcon(), "Undo"+keys("Z"), window.undo)
	hb.redoButton = button(theme.ContentRedoIcon(), "Redo"+keys("Shift+Z"), window.redo)
	hb.findButton = button(theme.SearchIcon(), "Find and replace"+keys("F"), window.showFindReplace)
	hb.previewButton = button(theme.VisibilityIcon(), "Preview"+keys("Shift+P"), window.togglePreview)
	hb.exportButton = button(theme.DownloadIcon(), "Export to PDF, HTML, FDX…"+keys("E"), window.showExportDialog)
	hb.preferencesButton = button(theme.SettingsIcon(), "Preferences"+keys(","), func() { window.app.showPreferences() })

	hb.titleLabel = widget.NewLabel("Untitled")
	hb.titleLabel.Alignment = fyne.TextAlignCenter
	hb.titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	hb.titleLabel.Truncation = fyne.TextTruncateEllipsis

	left := container.NewHBox(hb.newButton, hb.openButton, hb.saveButton, gap(), hb.undoButton, hb.redoButton)
	right := container.NewHBox(hb.findButton, hb.previewButton, hb.exportButton, gap(), hb.preferencesButton)
	hb.container = container.NewBorder(nil, nil, left, right, hb.titleLabel)
	return hb
}

// gap is a little space between groups of buttons.
func gap() fyne.CanvasObject {
	return container.NewGridWrap(fyne.NewSize(theme.Padding(), 1))
}

// keys formats a shortcut for a tooltip: " (Ctrl+S)", or " (⌘S)" on macOS.
func keys(k string) string {
	if runtime.GOOS == "darwin" {
		return " (" + strings.ReplaceAll("⌘"+k, "Shift+", "⇧") + ")"
	}
	return " (Ctrl+" + k + ")"
}

// SetTitle shows the document's name in the middle of the bar.
func (hb *HeaderBar) SetTitle(title string) {
	hb.titleLabel.SetText(title)
}

// SetVisible hides or shows the bar (for focus mode).
func (hb *HeaderBar) SetVisible(visible bool) {
	if visible {
		hb.container.Show()
	} else {
		hb.container.Hide()
	}
}
