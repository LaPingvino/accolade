// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import "testing"

// TestOpenedAsBar: a PDF opened without choosing says what it was opened
// as and offers the other way.
func TestOpenedAsBar(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	b := w.openedAs
	if b == nil || b.box.Visible() {
		t.Fatal("no bar, or shown at the start")
	}
	w.showOpenedAs("/x/novel.pdf", true)
	if !b.box.Visible() || b.text.Text != "Opened as Markdown: it reads as a novel" || b.other.Text != "Open as a Screenplay Instead" {
		t.Errorf("as Markdown: %q / %q", b.text.Text, b.other.Text)
	}
	w.showOpenedAs("/x/script.pdf", false)
	if b.text.Text != "Opened as a screenplay" || b.other.Text != "Open as Markdown Instead" || b.path != "/x/script.pdf" {
		t.Errorf("as a screenplay: %q / %q", b.text.Text, b.other.Text)
	}
	w.hideOpenedAs()
	if b.box.Visible() {
		t.Error("still shown")
	}
}
