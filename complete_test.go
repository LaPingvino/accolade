package main

import (
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

const completeScript = `INT. KITCHEN - DAY

ANNA
Hello.

ANDREW (V.O.)
Hi.

ANNA
Again.

EXT. GARDEN - NIGHT #2#

Anna walks out.

`

func TestCompleteAt(t *testing.T) {
	for _, c := range []struct {
		typed string
		want  []string
	}{
		{"AN", []string{"ANNA", "ANDREW"}}, // Anna speaks more often
		{"AND", []string{"ANDREW"}},
		{"     ANN", []string{"ANNA"}}, // indented like a name
		{"INT. K", []string{"INT. KITCHEN - DAY"}},
		{"ext", []string{"EXT. GARDEN - NIGHT"}},
		{"Anna", nil}, // not upper case: action
		{"ANNA", nil}, // complete already
		{"ZED", nil},
	} {
		text := completeScript + c.typed
		got := completeAt(text, len([]rune(text)))
		if c.want == nil {
			if got != nil {
				t.Errorf("%q: completions %v", c.typed, got.candidates)
			}
			continue
		}
		if got == nil || strings.Join(got.candidates, "|") != strings.Join(c.want, "|") {
			t.Errorf("%q: got %+v, want %v", c.typed, got, c.want)
		}
	}
	// not after a blank line: dialogue in capitals, nothing to complete
	text := completeScript + "BRAM\nAN"
	if got := completeAt(text, len([]rune(text))); got != nil {
		t.Errorf("dialogue completed: %v", got.candidates)
	}
}

func TestTabCycles(t *testing.T) {
	e := editor.New(completeScript + "    AN")
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false) })
	var k completer
	if !k.tab(e) || !strings.HasSuffix(e.Text(), "\n    ANNA") {
		t.Fatalf("first Tab: %q", e.Text()[len(completeScript):])
	}
	k.tab(e)
	if !strings.HasSuffix(e.Text(), "\n    ANDREW") {
		t.Errorf("second Tab: %q", e.Text()[len(completeScript):])
	}
	k.tab(e)
	if !strings.HasSuffix(e.Text(), "\n    ANNA") {
		t.Errorf("third Tab goes round: %q", e.Text()[len(completeScript):])
	}
	e2 := editor.New("Some action")
	e2.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false) })
	if (&completer{}).tab(e2) {
		t.Error("Tab completed action")
	}
	if h := completionHint(completeScript+"AN", len([]rune(completeScript))+2); h != "Tab: ANNA · ANDREW" {
		t.Errorf("hint %q", h)
	}
}
