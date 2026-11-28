// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package storyread

import "testing"

// The writer's notes (step 7): another name of a character merges its
// lines and its count under the character, a word that is no one is no
// speaker, and a place takes the heading given it.
func TestHints(t *testing.T) {
	ps := passagesFrom(`# One

The kitchen door opened. “Where were you?” said Annie.

“Out,” said Tom.

“No,” said Anna, and Annie laughed.

“Wait,” said Ferro.
`)
	plain := Read(ps, "en", Hints{})
	speakers := func(r *Reading) []string {
		var out []string
		for _, p := range r.Passages {
			for _, l := range p.Lines {
				out = append(out, l.Speaker)
			}
		}
		return out
	}
	if got := speakers(plain); len(got) != 4 || got[0] != "Annie" || got[3] != "Ferro" {
		t.Fatalf("without hints: %q", got)
	}
	h := Hints{Aliases: map[string]string{"annie": "ANNA", "Anna": "ANNA"}, NotCharacters: []string{"ferro"},
		Places: map[string]string{"the kitchen": "INT. KOVAČ KITCHEN"}}
	r := Read(ps, "en", h)
	got := speakers(r)
	want := []string{"ANNA", "Tom", "ANNA", "?"} // Ferro's: no one's, or the turns'
	for i := range want {
		if i >= len(got) || got[i] != want[i] && (want[i] != "?" || got[i] == "Ferro") {
			t.Fatalf("with hints: %q, want %q", got, want)
		}
	}
	n := 0
	for _, c := range r.Characters {
		switch c.Name {
		case "Annie", "Anna", "Ferro":
			t.Errorf("still a character: %+v", c)
		case "ANNA":
			n++
			if c.Mentions < 3 || c.Lines != 2 {
				t.Errorf("merged: %+v", c)
			}
		}
	}
	if n != 1 {
		t.Errorf("ANNA %d times in %+v", n, r.Characters)
	}
	sc := r.Scenes[r.Passages[1].Scene]
	if sc.Place != "KITCHEN" || sc.Given != "INT. KOVAČ KITCHEN" || sc.Heading("INT.", "EXT.", "INT./EXT.") != "INT. KOVAČ KITCHEN" {
		t.Errorf("place: %+v", sc)
	}
}
