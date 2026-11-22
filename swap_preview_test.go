package main

import "testing"

// A preview shown follows the document in the main editor (Joop: the preview
// "gets stuck when clicking swap"): the script's pages go when the source
// comes in and its Live Preview takes their place; swapping back closes the
// source's preview (as it did).
func TestSwapPreviewFollowsMain(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	v := previewAndWait(t, w)
	if !w.pagesVisible || w.previewView() != v {
		t.Fatal("the script's pages are not shown")
	}
	w.swapWithMain("source")
	if w.pagesVisible {
		t.Fatal("the script's pages still over the source swapped in")
	}
	if !w.previewVisible || !w.livePreviewShown() {
		t.Fatal("the source's Live Preview not shown in their place")
	}
	if got := w.previewPane.last; got != w.mainDoc.Text() {
		t.Errorf("the Live Preview shows %q, the main editor holds %q", got, w.mainDoc.Text())
	}
	w.swapWithMain("source") // back: the source's preview goes, as before
	if w.previewVisible {
		t.Error("the source's Live Preview still shown over the script")
	}
	// no preview before: none after, either way
	w.hidePages()
	w.swapWithMain("source")
	if w.pagesVisible || w.previewVisible {
		t.Error("a preview opened by the swap although none was shown")
	}
	w.swapWithMain("source")
	if w.pagesVisible || w.previewVisible {
		t.Error("a preview opened by swapping back although none was shown")
	}
}
