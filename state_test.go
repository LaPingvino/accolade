package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor"
	lexreport "github.com/LaPingvino/lexington/report"
)

// The state invariant (docs/STATE.md): after every transition of what the
// window shows, from the script in front and from the source in front,
// each view in sight describes the document it should — the one in front,
// or the script for the script's views — as it is now, not one from
// before the transition.
func TestViewsFollowTheFront(t *testing.T) {
	other := "Title: Another\n\nINT. BARN - DAY\n\nThe other script.\n\nEXT. FIELD - DAY\n\nThe end of the other one.\n"
	type step struct {
		name string
		do   func(t *testing.T, w *MainWindow, p string)
	}
	steps := []step{
		{"show source", func(_ *testing.T, w *MainWindow, _ string) { w.showMiddle("source") }},
		{"show notes", func(_ *testing.T, w *MainWindow, _ string) { w.showMiddle("notes") }},
		{"show cards", func(_ *testing.T, w *MainWindow, _ string) { w.showMiddle("cards") }},
		{"show script", func(_ *testing.T, w *MainWindow, _ string) { w.showMiddle("") }},
		{"edit the document in front", func(_ *testing.T, w *MainWindow, _ string) {
			w.writeEditor().Replace(0, 0, "# Added\n\n")
		}},
		{"preview", func(_ *testing.T, w *MainWindow, _ string) { w.preview() }},
		{"cards, then preview", func(_ *testing.T, w *MainWindow, _ string) {
			w.showMiddle("cards")
			w.preview()
		}},
		{"next tab", func(_ *testing.T, w *MainWindow, _ string) { w.mid.step(1) }},
		{"outline to the middle", func(_ *testing.T, w *MainWindow, _ string) { w.openMiddle("outline") }},
		{"outline there and back to the side", func(_ *testing.T, w *MainWindow, _ string) {
			w.openMiddle("outline")
			w.backToSide("outline")
		}},
		{"close the source's tab", func(_ *testing.T, w *MainWindow, _ string) { w.closeMiddle("source") }},
		{"close the notes' tab", func(_ *testing.T, w *MainWindow, _ string) { w.closeMiddle("notes") }},
		{"close a pane", func(_ *testing.T, w *MainWindow, _ string) { w.togglePane("characters") }},
		{"stripboard", func(_ *testing.T, w *MainWindow, _ string) { w.toggleStripboard() }},
		{"stripboard, then back to the script", func(_ *testing.T, w *MainWindow, _ string) {
			w.showStripboard(-1, -1)
			w.showMiddle("")
		}},
		{"a scene from the stripboard", func(_ *testing.T, w *MainWindow, _ string) {
			w.showStripboard(-1, -1)
			w.stripboard.openScene(1)
		}},
		{"close the stripboard", func(_ *testing.T, w *MainWindow, _ string) { w.closeMiddle("strips") }},
		{"pages tab", func(_ *testing.T, w *MainWindow, _ string) { w.showMiddle("pages") }},
		{"close the pages", func(_ *testing.T, w *MainWindow, _ string) { w.closeMiddle("pages") }},
		{"to a phone's width", func(_ *testing.T, w *MainWindow, _ string) { atWidth(w, 380) }},
		{"to a desktop's width", func(_ *testing.T, w *MainWindow, _ string) { atWidth(w, 1100) }},
		{"open", func(t *testing.T, w *MainWindow, p string) {
			q := filepath.Join(filepath.Dir(p), "other.fountain")
			os.WriteFile(q, []byte(other), 0o644)
			if err := w.LoadFile(q); err != nil {
				t.Fatal(err)
			}
		}},
		{"new", func(_ *testing.T, w *MainWindow, _ string) { w.createNewFile(false) }},
		{"open sample", func(t *testing.T, w *MainWindow, _ string) {
			s, err := sampleCopy(t.TempDir())
			if err != nil {
				t.Skip("no sample:", err)
			}
			if err := w.LoadFile(s); err != nil {
				t.Fatal(err)
			}
		}},
		{"new Markdown document", func(_ *testing.T, w *MainWindow, _ string) { w.hasChanges = false; w.NewMarkdownFile() }},
		{"document type to Markdown", func(_ *testing.T, w *MainWindow, _ string) { w.setDocumentType(true) }},
		{"document type there and back", func(_ *testing.T, w *MainWindow, _ string) {
			w.setDocumentType(true)
			w.setDocumentType(false)
		}},
		{"rename", func(t *testing.T, w *MainWindow, p string) {
			if err := w.relocate(relocation{rename: true, from: p, to: filepath.Join(filepath.Dir(p), "film.fountain"), keepBesideName: true}, nil); err != nil {
				t.Fatal(err)
			}
		}},
		{"share", func(t *testing.T, w *MainWindow, _ string) {
			key, _ := collab.NewKey()
			w.startSharing(key, false)
			t.Cleanup(w.stopSharing)
		}},
		{"join", func(t *testing.T, w *MainWindow, _ string) {
			key, _ := collab.NewKey()
			w.startSharing(key, true)
			t.Cleanup(w.stopSharing)
		}},
		{"share and leave", func(_ *testing.T, w *MainWindow, _ string) {
			key, _ := collab.NewKey()
			w.startSharing(key, false)
			w.leaveSharing()
		}},
		{"production", func(_ *testing.T, w *MainWindow, _ string) { w.setOwnRole(roleProduction) }},
		{"writer", func(_ *testing.T, w *MainWindow, _ string) { w.setOwnRole(roleWriter) }},
		{"choose another source", func(t *testing.T, w *MainWindow, p string) {
			q := filepath.Join(filepath.Dir(p), "another.md")
			os.WriteFile(q, []byte("# Another Story\n\nA new beginning in the snow.\n\nAnd its end.\n"), 0o644)
			w.chooseSourceFile(q)
			if d := w.midDoc(); d == nil || d.kind != docSource || !strings.Contains(d.Text(), "A new beginning") {
				t.Error("the Source tab is not the new source, in front")
			}
		}},
	}
	// the fronts: the script, the source, the stripboard (the script's
	// preview shown before it); on a phone the script and its Pages tab
	fronts := []struct {
		name, id string
		phone    bool
	}{{"script", "", false}, {"source", "source", false}, {"stripboard", "strips", false},
		{"phone, script", "", true}, {"phone, pages", "pages", true}}
	for _, from := range fronts {
		for _, s := range steps {
			t.Run(from.name+" in front, "+s.name, func(t *testing.T) {
				noTrackers(t)
				w, p := sourceWindow(t, sourceScript+"\n/* Notes\n# Notes heading\n\nThe notes.\n*/\n", sourceStory)
				holdUI(t) // the script's Preview is made in the background
				if from.phone {
					atWidth(w, 380)
				}
				w.togglePane("outline")
				w.togglePane("characters")
				for _, id := range []string{"source", "notes", "cards"} {
					w.openMiddle(id)
				}
				switch from.id {
				case "strips":
					showWithPreview(t, w, "")
					w.toggleStripboard()
				case "pages":
					showWithPreview(t, w, "")
					if w.mid.shown != "pages" {
						t.Fatalf("a phone's Preview: %q in front, not the Pages tab", w.mid.shown)
					}
				default:
					showWithPreview(t, w, from.id)
				}
				if from.phone && from.id == "" {
					w.showMiddle("") // a tap on Script: the pages kept in their tab
					if !w.pagesVisible || !w.mid.has("pages") {
						t.Fatal("the Pages tab went with a tap on Script")
					}
				}
				search(w, "the")
				checkViews(t, w, "before")
				s.do(t, w, p)
				waitFor(t, "the previews", func() bool { return previewsMaking.Load() == 0 })
				checkViews(t, w, "after")
			})
		}
	}
}

// checkViews checks each view in sight against the document it should
// describe: its record (docViews), and what it shows.
func checkViews(t *testing.T, w *MainWindow, when string) {
	t.Helper()
	for _, v := range w.docViews() {
		if k, on := v.shows(); on && k != w.want(v) {
			t.Errorf("%s: %s describes %v, not %v", when, v.name, k, w.want(v))
		}
	}
	front := w.frontDoc()
	if w.outline != nil && w.outlineInSight() {
		want := (&outlinePanel{w: w}).itemsOf(front)
		if !slices.Equal(w.outline.all, want) {
			t.Errorf("%s: the outline shows %d items %v, the %s in front has %d %v", when, len(w.outline.all),
				firstItems(w.outline.all), front.kind, len(want), firstItems(want))
		}
	}
	if sb := w.searchBar; sb != nil && sb.IsVisible() {
		editors := []*editor.ScriptEditor{w.textEditor}
		for _, d := range w.openDocs() {
			editors = append(editors, d.ed)
		}
		for _, e := range editors {
			if n := len(e.Highlights()); e != w.writeEditor() && n > 0 {
				t.Errorf("%s: %d highlights in an editor not in front", when, n)
			}
		}
		if n := len(w.writeEditor().Highlights()); n != sb.totalMatches {
			t.Errorf("%s: the find bar counts %d, the editor in front has %d highlights", when, sb.totalMatches, n)
		}
	}
	if w.livePreviewShown() && w.previewPane.last != w.writeEditor().Text() {
		t.Errorf("%s: the Live Preview is not of the %s in front", when, front.kind)
	}
	if w.pagesVisible && !w.scriptInMain() && !w.mid.has("pages") {
		t.Errorf("%s: the script's pages over the %s", when, front.kind)
	}
	if tab := w.mid.has("pages"); tab != (w.narrow && w.pagesVisible) {
		t.Errorf("%s: a Pages tab %v, a phone %v, pages shown %v", when, tab, w.narrow, w.pagesVisible)
	}
	if w.mid.has("pages") && slices.Index(w.mid.tabs, "pages") != 0 {
		t.Errorf("%s: the Pages tab is not next to Script: %v", when, w.mid.tabs)
	}
	if w.stripsShown() && w.ribbon != nil && w.ribbon.selected != "Stripboard" {
		t.Errorf("%s: the stripboard in front, the ribbon's %q tab", when, w.ribbon.selected)
	}
	if d := w.midDoc(); d != nil && !strings.Contains(w.fyneWindow.Title(), d.title()) {
		t.Errorf("%s: the title %q does not name %s", when, w.fyneWindow.Title(), d.title())
	}
}

// atWidth makes the window width wide (a phone under narrowWidth).
func atWidth(w *MainWindow, width float32) {
	w.fyneWindow.Resize(fyne.NewSize(width, 800))
	w.onWindowWidth(width)
}

func firstItems(items []outlineItem) []string {
	var out []string
	for _, it := range items[:min(3, len(items))] {
		out = append(out, it.Text)
	}
	return out
}

// The outline with the script in front and with the source in front (and
// the notes), desktop and phone: SHOT_DIR=… go test -run TestOutlineScreenshots.
func TestOutlineScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, size := range []struct {
		name string
		w, h float32
	}{{"desktop", 1100, 680}, {"phone", 380, 800}} {
		w, _ := sourceWindow(t, sourceScript+"\n/* Notes\n# Anna\n\nThe door.\n\n# The frost\n\nIn the morning.\n*/\n", sourceStory)
		w.hidePane("source")
		w.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		w.onWindowWidth(size.w)
		w.showPane("outline", true)
		snap := func(name string) {
			t.Helper()
			w.fyneWindow.Content().Refresh()
			snapWindow(t, w, filepath.Join(dir, "outline-"+size.name+"-"+name+".png"))
		}
		snap("0-script")
		w.openMiddle("source")
		w.showPane("outline", true)
		snap("1-source")
		w.openMiddle("notes")
		w.showPane("outline", true)
		snap("2-notes")
	}
}

// With the source in front, a click on a passage in the outline goes to it
// in the source's editor (the source stays in front); a row does not move.
func TestOutlineGoesIntoTheSource(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.togglePane("outline")
	w.openMiddle("source")
	o := w.outline
	at := slices.IndexFunc(o.items, func(it outlineItem) bool { return strings.HasPrefix(it.Text, "The next morning") })
	if at < 0 {
		t.Fatalf("no passage in the outline: %v", firstItems(o.items))
	}
	o.list.Select(at)
	d := w.midDoc()
	if d == nil || d.kind != docSource {
		t.Fatal("the source is no longer in front")
	}
	line, _ := d.ed.Buffer().Position(d.ed.CursorOffset())
	if got := strings.Split(d.Text(), "\n")[line]; !strings.HasPrefix(got, "The next morning") {
		t.Errorf("the source's cursor is on %q", got)
	}
	script := w.textEditor.Text()
	o.moveRow(at, 0)
	if w.textEditor.Text() != script || d.Text() != sourceStory {
		t.Error("a source row moved something")
	}
}

// A report's PDF (and Export's preview) in the page view is no Preview of
// the script: a change of the preferences or of the script leaves it.
func TestReportPagesAreNotFollowed(t *testing.T) {
	w, rd := reportsWindow(t, fyne.NewSize(1100, 760))
	rd.showReportPDF(lexreport.KindSheets)
	if !w.pagesVisible || w.pages == nil {
		t.Fatal("the sheets are not shown")
	}
	d := w.pages.d
	w.changeSetting("font-size", func(s *Settings) { s.SetFloat("font-size", 17) })
	w.scriptChange()() // as at the end of Open, a rename…
	if !w.pagesVisible || w.pages.d.Pages == nil || len(w.pages.d.Pages) != len(d.Pages) || w.pagesComing != 0 {
		t.Errorf("the report's pages were replaced (visible %v, a Preview coming %v)", w.pagesVisible, w.pagesComing != 0)
	}
}
