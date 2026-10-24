package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Print… makes Lexington's HTML of the script (every script's letters in
// it, for the browser's fonts) and hands it to the browser to print.
func TestPrintScript(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	got := make(chan string, 1)
	was := printFile
	printFile = func(p string) { got <- p }
	t.Cleanup(func() { printFile = was })
	w.textEditor.SetText("INT. NHÀ - NGÀY\n\nTrời mưa. 日本語のせりふ.\n")
	w.printScript()
	select {
	case p := <-got:
		b, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(b), "Trời mưa. 日本語のせりふ.") {
			t.Errorf("printed %s: %v", p, err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("nothing to print")
	}
}
