// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package opml

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func isScene(l string) bool {
	u := strings.ToUpper(strings.TrimSpace(l))
	return strings.HasPrefix(u, "INT.") || strings.HasPrefix(u, "EXT.")
}

func readFile(t *testing.T, name string) Doc {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if !Sniff(data) {
		t.Fatalf("%s: not seen as OPML", name)
	}
	d, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// A Workflowy export (one bullet holding the story, notes, completed
// items, its inline HTML, an empty bullet): the nesting as # and ##, a
// note as the synopsis touching its heading, the leaves as beats, a
// scene heading as a scene (its note its own synopsis).
func TestWorkflowy(t *testing.T) {
	got := Fountain(readFile(t, "testdata/workflowy.opml"), isScene)
	want := `# The Lighthouse

## Act One
= The keeper's daughter comes home.

= Mara finds her father's **letter** in the lamp room.

= She decides to stay the winter.

= #todo check the lamp's dates

## Act Two
= The storm, and what it takes.
= Nobody sleeps.

### The storm

= The storm tears the roof from the boathouse.

= Tomas goes missing on the water.
= Is he hiding?

ext. harbour - night
= Mara takes the old boat out alone.

## Act Three

= She finds *Tomas* on the rocks & brings him home.
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// An OmniOutliner export (a title, its columns as attributes, a DOCTYPE).
func TestOmniOutliner(t *testing.T) {
	d := readFile(t, "testdata/omnioutliner.opml")
	if d.Title != "The Lighthouse" {
		t.Errorf("title %q", d.Title)
	}
	got := Fountain(d, isScene)
	want := `# Act One
= The keeper's daughter comes home.

= Mara finds her father's letter in the lamp room.

INT. LAMP ROOM - NIGHT
= She reads it twice.

# Act Two
= The storm, and what it takes.

= The storm tears the roof from the boathouse.

= Tomas goes missing on the water.

# Act Three

= She finds Tomas on the rocks.
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// Accolade's own items (_kind) come back as they were, through Write and
// Read: a ## with no # above it, a note, a forced scene heading, a
// synopsis of more than one line, the title page.
func TestOwnKindsRoundTrip(t *testing.T) {
	d := Doc{Title: "T", Nodes: []*Node{
		{Kind: KindTitlePage, Text: "Title page", Note: "Title: T\nAuthor: A"},
		{Kind: KindNote, Text: "[[how to use it]]"},
		{Kind: KindSection, Text: "Part", Level: 2, Note: "What it does\n[[red]]", Kids: []*Node{
			{Kind: KindSynopsis, Text: "A beat", Note: "on two lines"},
			{Kind: KindScene, Text: ".THE VOID", Note: "Nothing.", Kids: []*Node{{Kind: KindSynopsis, Text: "Later."}}},
		}},
	}}
	var b bytes.Buffer
	if err := Write(&b, d); err != nil {
		t.Fatal(err)
	}
	back, err := Read(&b)
	if err != nil {
		t.Fatal(err)
	}
	want := "Title: T\nAuthor: A\n\n[[how to use it]]\n\n## Part\n= What it does\n[[red]]\n\n= A beat\n= on two lines\n\n.THE VOID\n= Nothing.\n\n= Later.\n"
	if got := Fountain(back, isScene); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
	if back.Title != "T" {
		t.Errorf("title %q", back.Title)
	}
}

// Indented text as outliners copy it: Workflowy's "- " bullets two
// spaces a level with its notes in quotes, tabs, a plain list.
func TestParseIndented(t *testing.T) {
	wf := "- Act One\n  \"The keeper's daughter comes home.\"\n  - Mara finds the letter.\n  - She stays.\n- Act Two\n  - INT. BOATHOUSE - NIGHT\n"
	got := Fountain(Doc{Nodes: ParseIndented(wf)}, isScene)
	want := "# Act One\n= The keeper's daughter comes home.\n\n= Mara finds the letter.\n\n= She stays.\n\n# Act Two\n\nINT. BOATHOUSE - NIGHT\n"
	if got != want {
		t.Errorf("Workflowy's text:\n%q\nwant\n%q", got, want)
	}
	tabs := "Act One\n\tMara finds the letter.\n\t\tA detail of it.\nAct Two\n\tThe storm.\n"
	got = Fountain(Doc{Nodes: ParseIndented(tabs)}, isScene)
	want = "# Act One\n\n## Mara finds the letter.\n\n= A detail of it.\n\n# Act Two\n\n= The storm.\n"
	if got != want {
		t.Errorf("tabs:\n%q\nwant\n%q", got, want)
	}
}

// Not an outline: an error, not an empty script.
func TestNotOPML(t *testing.T) {
	if Sniff([]byte("INT. HOUSE - DAY")) {
		t.Error("a script seen as OPML")
	}
	if _, err := Read(strings.NewReader("<html><body>no</body></html>")); err == nil {
		t.Error("HTML read as OPML")
	}
}
