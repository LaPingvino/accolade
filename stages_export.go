// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"github.com/LaPingvino/lexington/fountain"
)

// The two versions one bundle exports (docs/LAYERS.md): the writer's
// version, the writer's draft as it was locked (the White issue; before
// any issue the script now) without production's layers, and the
// shooting version, the script now with all of them. Export… asks which
// (Version:), File > Export Writer's Version… and Export Shooting
// Version… open it with one chosen.

// The versions.
const (
	versionWriter   = "writer"
	versionShooting = "shooting"
)

// versionLabels are the dialog's choices, by version.
var versionLabels = map[string]string{
	versionWriter:   "Writer's version",
	versionShooting: "Shooting version",
}

// versionOf is the version a choice's label names.
func versionOf(label string) string {
	for v, l := range versionLabels {
		if l == label {
			return v
		}
	}
	return versionShooting
}

// writerStrip is what the writer's version leaves out besides
// Accolade's blocks and links (fountain.Strip always does): production's
// scene numbers and OMITTED scenes. The writer's own notes, boneyard,
// title page and outline stay.
var writerStrip = fountain.StripOptions{SceneNumbers: true, Omitted: true}

// writerDraft is the writer's draft as locked: the first issue's text
// (White) and its name, or the script now and "" before any issue.
func (w *MainWindow) writerDraft() (text, from string) {
	h := w.extrasHome()
	if ds, err := readDrafts(h); err == nil {
		for _, d := range ds {
			if d.Kind != kindIssue {
				continue
			}
			if data, err := h.readExtra(d.Text); err == nil {
				text, _ = decodeScript(data)
				return text, d.Name
			}
			break
		}
	}
	return w.textEditor.Text(), ""
}

// writerVersion is the writer's version's text, and the issue it is of.
func (w *MainWindow) writerVersion() (text, from string) {
	text, from = w.writerDraft()
	return fountain.Strip(text, w.sceneHeaders(), writerStrip), from
}

// versionAbout says what a version is, for the dialog.
func (w *MainWindow) versionAbout(v string) string {
	if v == versionWriter {
		if _, from := w.writerDraft(); from != "" {
			return "The writer's draft as locked (" + from + "): no scene numbers, OMITTED scenes, revisions or production data."
		}
		return "The script now, as the writer has it (nothing is locked yet): no scene numbers, OMITTED scenes or production data."
	}
	return "The script now with production's layers: scene numbers, OMITTED, the revisions (as the page setup shows them), locked pages."
}

// showVersionExport is Export… with a version chosen.
func (w *MainWindow) showVersionExport(v string) {
	if w.markdown {
		return
	}
	ed := NewExportDialog(w)
	ed.setVersion(v)
	ed.Show()
}
