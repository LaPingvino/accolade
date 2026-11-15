package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/pdfin"
)

// The dialog's page breaks: the format's by default, all, none or some
// parts, as Lexington's -pagination; saved for the preview and printing.
func TestExportPagination(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Cleanup(func() { w.settings.SetString("export-pagination", "") })
	ed := NewExportDialog(w)
	ed.Show()
	if ed.paginationSelect.Selected != "Script format's" || ed.job().options("pdf", "x.pdf").Pagination != "" {
		t.Errorf("default %q", ed.paginationSelect.Selected)
	}
	parts := func() bool { return !ed.paginationKeep.Disabled() && !ed.paginationWidows.Disabled() && !ed.paginationMore.Disabled() }
	for _, c := range []struct{ choice, want string }{
		{"All (as screenplay software)", "all"},
		{"None (as before)", "none"},
		{"Script format's", ""},
	} {
		ed.paginationSelect.SetSelected(c.choice)
		if got := ed.job().options("pdf", "x.pdf").Pagination; got != c.want || parts() {
			t.Errorf("%s: %q, parts enabled %v", c.choice, got, parts())
		}
	}
	ed.paginationSelect.SetSelected("Some:")
	if !parts() {
		t.Error("Some: the parts are not enabled")
	}
	ed.paginationWidows.SetChecked(false)
	if got := ed.job().options("pdf", "x.pdf").Pagination; got != "keep+more" {
		t.Errorf("keep and more: %q", got)
	}
	if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("no name for %v", u)
	}
	ed.scroll.ScrollToBottom()
	ed.scroll.ScrollToOffset(ed.scroll.Offset.SubtractXY(0, 150))
	capture(t, w.fyneWindow.Canvas(), "export-pagination.png")
	ed.saveSettings()
	if got := w.savedExportJob().options("pdf", "x.pdf").Pagination; got != "keep+more" {
		t.Errorf("preview and print: %q", got)
	}
	ed.cancel()
	ed = NewExportDialog(w)
	if ed.paginationSelect.Selected != "Some:" || !ed.paginationKeep.Checked || ed.paginationWidows.Checked || !ed.paginationMore.Checked {
		t.Errorf("kept as %q %v %v %v", ed.paginationSelect.Selected, ed.paginationKeep.Checked, ed.paginationWidows.Checked, ed.paginationMore.Checked)
	}
	ed.paginationKeep.SetChecked(false)
	ed.paginationMore.SetChecked(false)
	if got := ed.chosenPagination(); got != "none" {
		t.Errorf("some, none of them: %q", got)
	}
}

// A speech over a page break: (MORE) with the screenplay's page breaks,
// none without.
func TestExportPaginationMore(t *testing.T) {
	var b strings.Builder
	b.WriteString("INT. HALL - DAY\n\n")
	for i := 0; i < 20; i++ {
		b.WriteString("Something happens.\n\n")
	}
	b.WriteString("ANNA\n")
	for i := 0; i < 60; i++ {
		b.WriteString("This is a long speech that goes on. ")
	}
	b.WriteString("\n")
	more := func(pagination string) bool {
		t.Helper()
		out := filepath.Join(t.TempDir(), "s.pdf")
		if err := exportAs(b.String(), out, "pdf", exportJob{format: "default", pagination: pagination}); err != nil {
			t.Fatal(err)
		}
		pages, err := pdfin.Pages(out)
		if err != nil {
			t.Fatal(err)
		}
		for _, page := range pages {
			for _, l := range page {
				if strings.Contains(l.Text, "(MORE)") {
					return true
				}
			}
		}
		return false
	}
	if !more("all") || !more("") {
		t.Error("no (MORE) with the screenplay's page breaks")
	}
	if more("none") || more("keep+widows") {
		t.Error("(MORE) without")
	}
}
