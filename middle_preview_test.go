package main

import (
	"testing"
	"time"
)

// A preview shown follows the middle tab in front (Joop: the preview "gets
// stuck when clicking swap"): from each kind of tab with its preview to
// each other kind, the preview shown is of the tab now in front: the
// script's pages, a document's Live Preview of its own text, nothing over
// a view (the cards, the outline), and the next editor tab gets its own
// again. With no preview shown, none comes.
func TestMiddlePreviewFollowsShownTab(t *testing.T) {
	kinds := []string{"", "source", "notes", "cards", "outline"}
	name := map[string]string{"": "script", "source": "source", "notes": "notes", "cards": "cards", "outline": "outline"}
	editorTab := func(id string) bool { return id == "" || id == "source" || id == "notes" }
	for _, from := range kinds {
		if !editorTab(from) {
			continue // a view has no preview to start from
		}
		for _, to := range kinds {
			if to == from {
				continue
			}
			t.Run(name[from]+"→"+name[to], func(t *testing.T) {
				w, _ := sourceWindow(t, sourceScript+"\n/* Notes\nThe notes.\n*/\n", sourceStory)
				for _, id := range []string{"source", "notes", "cards", "outline"} {
					w.openMiddle(id)
				}
				showWithPreview(t, w, from)
				switchTo(t, w, to)
				previewIsOf(t, w, to)
				if !editorTab(to) { // through a view: the next editor tab has its preview
					back := "source"
					switchTo(t, w, back)
					previewIsOf(t, w, back)
				}
			})
		}
	}
	// no preview before: none after, whichever way
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.openMiddle("source")
	w.openMiddle("cards")
	for _, id := range []string{"", "source", "cards", "source", ""} {
		w.showMiddle(id)
		if w.pagesVisible || w.previewVisible {
			t.Fatalf("a preview opened by showing %q although none was shown", id)
		}
	}
}

// showWithPreview puts tab id in front with its preview: a screenplay's
// pages, a document's Live Preview.
func showWithPreview(t *testing.T, w *MainWindow, id string) {
	t.Helper()
	if id == "" {
		w.showMiddle("")
		previewAndWait(t, w)
	} else {
		w.showMiddle(id)
		w.preview()
	}
	previewIsOf(t, w, id)
}

// switchTo puts tab id in front, waiting for the script's pages when it
// is the script's tab coming in with a preview to follow.
func switchTo(t *testing.T, w *MainWindow, id string) {
	t.Helper()
	if id != "" {
		w.showMiddle(id)
		return
	}
	shown := make(chan *pageView, 4)
	onPagesShown(t, func(v *pageView) { shown <- v })
	w.showMiddle("")
	select {
	case <-shown:
	case <-time.After(60 * time.Second):
		t.Fatal("the script's pages never came")
	}
}

// previewIsOf checks that the preview shown is of tab id, and none of
// another document.
func previewIsOf(t *testing.T, w *MainWindow, id string) {
	t.Helper()
	switch {
	case id == "":
		if !w.pagesVisible || w.previewVisible || w.previewView() != w.pages {
			t.Errorf("the script in front: pages %v, Live Preview %v", w.pagesVisible, w.previewVisible)
		}
	case w.mid.docs[id] != nil:
		d := w.mid.docs[id]
		if w.pagesVisible || !w.livePreviewShown() || w.previewPane.last != d.Text() {
			t.Errorf("%s in front: pages %v, its Live Preview %v, of its text %v", id, w.pagesVisible, w.livePreviewShown(),
				w.previewPane != nil && w.previewPane.last == d.Text())
		}
	default:
		if w.pagesVisible || w.previewVisible {
			t.Errorf("%s in front: a preview over it (pages %v, Live Preview %v)", id, w.pagesVisible, w.previewVisible)
		}
	}
}
