package main

import (
	"strings"

	"github.com/LaPingvino/lexington/cli"
)

// exportFormat is a format the Export dialog offers: Lexington's (all of
// them, as its command line has them), and the script's text as it is.
type exportFormat struct {
	id    string // the setting's value; for Lexington's, its -to
	label string
	ext   string
	// lexington: written by Lexington (cli.Write) from the parsed script
	lexington bool
	format    cli.Format
}

// exportFormatList are the Export dialog's formats: Lexington's, with the
// text as written (a copy of the .fountain file) and plain text.
func exportFormatList() []exportFormat {
	var list []exportFormat
	for _, f := range cli.Formats {
		label := f.Name
		if f.Key == "fountain" {
			label = "Fountain (rewritten by Lexington)"
		}
		if f.Needs != "" {
			label += " · needs " + f.Needs
		}
		list = append(list, exportFormat{id: f.Key, label: label, ext: f.Extension(), lexington: true, format: f})
		if f.Key == "fdx" {
			list = append(list,
				exportFormat{id: "text-fountain", label: "Fountain (as written)", ext: "fountain"},
				exportFormat{id: "txt", label: "Plain text", ext: "txt"})
		}
	}
	return list
}

// exportFormats are the formats' labels, in order.
var exportFormats = func() []string {
	var labels []string
	for _, f := range exportFormatList() {
		labels = append(labels, f.label)
	}
	return labels
}()

// exportFormatByLabel is the format with a label (or id), the first if
// there is none.
func exportFormatByLabel(label string) exportFormat {
	list := exportFormatList()
	for _, f := range list {
		if f.label == label {
			return f
		}
	}
	// a setting: an id, or a label of earlier versions (DOCX, TXT, Fountain)
	old := map[string]string{"fountain": "text-fountain"}
	id := strings.ToLower(label)
	if o, ok := old[id]; ok {
		id = o
	}
	for _, f := range list {
		if f.id == id {
			return f
		}
	}
	return list[0]
}

// exportFormatLabel is the label of the format a setting names.
func exportFormatLabel(setting string) string { return exportFormatByLabel(setting).label }
