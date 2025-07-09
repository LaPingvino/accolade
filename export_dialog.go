package main

import (
	"log"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type ExportDialog struct {
	*adw.PreferencesWindow
	window *MainWindow
}

func NewExportDialog(window *MainWindow) *ExportDialog {
	return &ExportDialog{
		PreferencesWindow: adw.NewPreferencesWindow(),
		window:            window,
	}
}

func (ed *ExportDialog) Present() {
	log.Println("Export dialog presented")
	ed.PreferencesWindow.Present()
}

func (ed *ExportDialog) ExportToPDF(filename string) error {
	log.Printf("Exporting to PDF: %s", filename)
	return nil
}

func (ed *ExportDialog) ExportToHTML(filename string) error {
	log.Printf("Exporting to HTML: %s", filename)
	return nil
}

func (ed *ExportDialog) ExportToFDX(filename string) error {
	log.Printf("Exporting to FDX: %s", filename)
	return nil
}