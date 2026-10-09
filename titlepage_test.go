package main

import (
	"strings"
	"testing"
	"time"
)

func titlePageDialog(t *testing.T, w *MainWindow) *TitlePageDialog {
	t.Helper()
	d := NewTitlePageDialog(w)
	d.titleEntry.SetText("The Barn")
	d.creditEntry.SetText("Written by")
	d.authorEntry.SetText("Jane Smith")
	d.basedOnEntry.SetText("a true story")
	d.includeDate.SetChecked(true)
	d.draftDateEntry.SetText("1 May 2026")
	d.includeContact.SetChecked(true)
	d.nameEntry.SetText("Jane Smith")
	d.addressEntry.SetText("1 Writer's Lane\n\nSpringfield")
	d.phoneEntry.SetText("")
	d.emailEntry.SetText("jane@example.com")
	return d
}

const barnTitlePage = "Title: The Barn\nCredit: Written by\nAuthor: Jane Smith\nSource: a true story\nDraft date: 1 May 2026\nContact:\n    Jane Smith\n    1 Writer's Lane\n    Springfield\n    jane@example.com"

func TestGenerateTitlePage(t *testing.T) {
	w := newTestWindow(t, "")
	if got := titlePageDialog(t, w).generateTitlePageText(); got != barnTitlePage {
		t.Errorf("got:\n%s\nwant:\n%s", got, barnTitlePage)
	}

	// and lexington reads every field, contact lines included
	var fields []string
	for _, l := range parseForExport(barnTitlePage + "\n\nINT. BARN - DAY\n") {
		if l.Contents != "" {
			fields = append(fields, string(l.Type)+"="+l.Contents)
		}
	}
	want := "Title=The Barn|Credit=Written by|Author=Jane Smith|Source=a true story|Draft date=1 May 2026|Contact=Jane Smith|Contact=1 Writer's Lane|Contact=Springfield|Contact=jane@example.com|scene=INT. BARN - DAY"
	if strings.Join(fields, "|") != want {
		t.Errorf("parsed:\n%s\nwant:\n%s", strings.Join(fields, "|"), want)
	}
}

func TestInsertTitlePage(t *testing.T) {
	script := "FADE IN:\n\nINT. BARN - DAY\n\nRain."
	cases := map[string]string{
		"no title page":       script,
		"leading blank lines": "\n\n" + script,
		"replaces old one":    "Title: Old\nAuthor: Someone\nNotes: keep out\nContact:\n    Old Address\n\n\n" + script,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			w := newTestWindow(t, doc)
			titlePageDialog(t, w).insertTitlePage()
			want := barnTitlePage + "\n\n" + script
			if name == "replaces old one" { // a field the dialog has none for stays
				want = barnTitlePage + "\nNotes: keep out\n\n" + script
			}
			if w.textEditor.Text() != want {
				t.Errorf("got:\n%q\nwant:\n%q", w.textEditor.Text(), want)
			}
		})
	}
}

func TestTitlePageEnd(t *testing.T) {
	cases := []struct {
		doc  string
		want int
	}{
		{"FADE IN:\n\nINT. BARN", 0},
		{"INT. BARN: LOFT - DAY\n\nHay.", 0},
		{"Rain falls.\nTitle: in the action", 0},
		{"Title: X\n\nFADE IN:", 2},
		{"Title: X\nContact:\n    a\n\tb\n\n\nINT. BARN", 6},
		{"Title: X\nAuthor: Y", 2},
	}
	for _, c := range cases {
		if got := titlePageEnd(strings.Split(c.doc, "\n")); got != c.want {
			t.Errorf("titlePageEnd(%q) = %d, want %d", c.doc, got, c.want)
		}
	}
}

func TestDefaultScriptParses(t *testing.T) {
	var got []string
	for _, l := range parseForExport(defaultScript(time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC))) {
		if l.Contents != "" {
			got = append(got, string(l.Type)+"="+l.Contents)
		}
	}
	want := []string{
		"Title=UNTITLED SCREENPLAY", "Credit=Written by", "Author=Your Name", "Draft date=October 3, 2026",
		"Contact=Your Name", "Contact=Your Address", "Contact=Your Phone", "Contact=Your Email",
		"action=FADE IN:", "scene=INT. LIVING ROOM - DAY",
		"action=A simple room with basic furniture. Light streams through the windows.",
		"action=JOHN sits at a desk, typing on a laptop. He looks frustrated.",
		"speaker=JOHN", "paren=(sighing)", "dialog=This screenplay isn't writing itself.",
		"action=He takes a sip of coffee and continues typing.", "trans=FADE OUT.",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestStatusBarElement(t *testing.T) {
	doc := defaultScript(time.Now())
	w := newTestWindow(t, doc)
	for line, want := range map[string]string{
		"INT. LIVING ROOM - DAY":                "Scene Heading",
		"A simple room":                         "Action",
		"JOHN\n":                                "Character",
		"(sighing)":                             "Parenthetical",
		"This screenplay isn't writing itself.": "Dialogue",
		"> FADE OUT.":                           "Transition",
	} {
		i := strings.Index(doc, line)
		w.textEditor.SetCursorOffset(len([]rune(doc[:i]))+1)
		w.updateCurrentElement()
		if w.currentElement != want {
			t.Errorf("cursor in %q: element %q, want %q", line, w.currentElement, want)
		}
	}
}

// The dialog edits the script's own title page: it shows its fields,
// keeps the ones it has no field for, and Insert can be undone.
func TestTitlePageDialogKeepsTheScripts(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	script := "Title: MY FILM\nAuthor: Ann\nCopyright: (c) 2026 Ann\nNotes: second draft\nContact:\n    Ann Smith\n    1 Lane\n    ann@example.com\n\nINT. A - DAY\n\nAction.\n"
	w.textEditor.SetText(script)
	d := NewTitlePageDialog(w)
	if d.titleEntry.Text != "MY FILM" || d.authorEntry.Text != "Ann" || d.nameEntry.Text != "Ann Smith" ||
		d.emailEntry.Text != "ann@example.com" || d.addressEntry.Text != "1 Lane" {
		t.Errorf("fields: %q %q %q %q %q", d.titleEntry.Text, d.authorEntry.Text, d.nameEntry.Text, d.emailEntry.Text, d.addressEntry.Text)
	}
	d.authorEntry.SetText("Ann Smith")
	d.insertTitlePage()
	got := w.textEditor.Text()
	for _, want := range []string{"Title: MY FILM", "Author: Ann Smith", "Copyright: (c) 2026 Ann", "Notes: second draft",
		"    ann@example.com", "\n\nINT. A - DAY\n\nAction.\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("after Insert, no %q in:\n%s", want, got)
		}
	}
	if strings.Count(got, "INT. A - DAY") != 1 || strings.Contains(got, "UNTITLED") {
		t.Errorf("after Insert:\n%s", got)
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != script {
		t.Errorf("undo:\n%s", w.textEditor.Text())
	}
}
