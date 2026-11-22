package storyread

import (
	"slices"
	"testing"
)

// A heading's place comes from place evidence: a place word with the
// words that name it with it (market square, Pitchugin's timber-yard), a
// name after a preposition of place that ends the phrase, a place the
// writer gave a heading. A name that only owns or qualifies the
// preposition's noun is none ("in God's sight", "in Äußerung ihrer
// Sorgen"), nor is a name said to someone (present tense too: "Hryukin
// begins").
func TestPlaceEvidence(t *testing.T) {
	ps := passagesFrom(`# A Chameleon

Otchumyelov is walking across the market square. The open doors of the shops look out upon God's world.

"So you bite?" Otchumyelov hears. A dog runs out of Pitchugin's timber-yard.

"I was walking along here," Hryukin begins. "I swear it, as in God's sight."

"What is it?" Otchumyelov turns to Hryukin. The crowd laughs at Hryukin.

Prohor calls the dog, and walks away from the timber-yard with her.
`)
	places := func(r *Reading) []string {
		var out []string
		for _, p := range r.Places {
			out = append(out, p.Name)
		}
		return out
	}
	r := Read(ps, "en", Hints{})
	if pl := places(r); slices.Contains(pl, "God") || slices.Contains(pl, "Hryukin") {
		t.Errorf("places %q", pl)
	}
	if !slices.ContainsFunc(r.Characters, func(c Character) bool { return c.Name == "Hryukin" && c.Lines == 1 }) {
		t.Errorf("Hryukin is no speaker: %+v", r.Characters)
	}
	if h := r.Scenes[0].Heading("INT.", "EXT.", "INT./EXT."); h != "EXT. MARKET SQUARE" {
		t.Errorf("heading %q", h)
	}
	for i, p := range r.Passages {
		if p.Place.Place != "" {
			t.Errorf("passage %d names %+v without a name or a lesson", i, p.Place)
		}
	}
	// the writer's place: where a passage names it, the passage's own
	h := Hints{Places: map[string]string{"the timber-yard": "EXT. PITCHUGIN'S TIMBER-YARD"}}
	r = Read(ps, "en", h)
	var own []string
	for _, p := range r.Passages {
		own = append(own, p.Place.Given)
	}
	if want := []string{"", "", "EXT. PITCHUGIN'S TIMBER-YARD", "", "", "EXT. PITCHUGIN'S TIMBER-YARD"}; !slices.Equal(own, want) {
		t.Errorf("own places %q", own)
	}
	if h := r.Scenes[0].Heading("INT.", "EXT.", "INT./EXT."); h != "EXT. MARKET SQUARE" {
		t.Errorf("the scene's heading with the lesson: %q", h)
	}
	// an owner's name with the place word, where nothing comes before
	r = Read(passagesFrom("A dog runs out of Pitchugin's timber-yard, yelping.\n"), "en", Hints{})
	if h := r.Scenes[0].Heading("INT.", "EXT.", "INT./EXT."); h != "EXT. PITCHUGIN'S TIMBER-YARD" {
		t.Errorf("the owner's: %q", h)
	}
	// a place by name, after a preposition, keeps being one
	r = Read(passagesFrom("They came at last to the inn in Meryton, and in Meryton they stayed.\n\nMeryton was quiet.\n"), "en", Hints{})
	if pl := places(r); !slices.Contains(pl, "Meryton") {
		t.Errorf("Meryton: %q", pl)
	}
	// German: the noun's genitive goes on after it
	r = Read(passagesFrom("Und die Mama schien ernstlich willens, in Äußerung ihrer Sorgen und Ängste fortzufahren. Sie saß im Garten.\n"), "de", Hints{})
	if pl := places(r); len(pl) > 0 {
		t.Errorf("de places %q", pl)
	}
	if h := r.Scenes[0].Heading("INT.", "EXT.", "INT./EXT."); h != "EXT. GARTEN" {
		t.Errorf("de heading %q", h)
	}
}
