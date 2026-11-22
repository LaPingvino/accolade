package main

import (
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/fountain"
)

func typesOf(text, lang string) string {
	var out []string
	for _, l := range fountain.ParseIn(lexSettings.Language(lang), strings.NewReader(text)) {
		out = append(out, l.Type)
	}
	return strings.Join(out, " ")
}

// Every configured language's script, as standard Fountain, reads as the
// same elements with the standard words, and comes back as written
// (Lexington tests every pair; here Accolade's settings and wrappers).
func TestDialectFountainRoundTrip(t *testing.T) {
	std := standardLanguage()
	for _, lang := range scriptLanguages() {
		var b strings.Builder
		b.WriteString("Title: A Test\n\n")
		for _, w := range sceneHeaders(lang) {
			b.WriteString(w + " PLACE - DAY\n\nSomething happens.\n\n")
		}
		for _, w := range sceneHeaders(std) {
			b.WriteString(w + ". kitchen is written on the wall.\n\n" + w + ". BOB\nHello.\n\n")
		}
		native := b.String()
		forced := standardFountain(native, lang)
		if back := dialectFountain(forced, lang); back != native {
			t.Errorf("%s: round trip changed the text:\n%s\n---\n%s", lang, native, back)
		}
		if a, b := typesOf(native, lang), typesOf(forced, std); a != b {
			t.Errorf("%s: standard Fountain reads other elements:\n%s\n%s", lang, a, b)
		}
		if lang == std && forced != native {
			t.Errorf("the standard language's text changed")
		}
	}
}

func TestDialectFountainMarks(t *testing.T) {
	native := "EN. DOMO - TAGO\n\nINT. is how English starts a heading.\n\nINT. BOB\nSaluton.\n\n!EN. needed by Esperanto too\n\n.INT. A HEADING ON PURPOSE\n"
	want := ".EN. DOMO - TAGO\n\n!INT. is how English starts a heading.\n\n@INT. BOB\nSaluton.\n\n!EN. needed by Esperanto too\n\n.INT. A HEADING ON PURPOSE\n"
	if forced := standardFountain(native, "eo"); forced != want {
		t.Fatalf("forced:\n%s", forced)
	}
	if back := dialectFountain(want, "eo"); back != native {
		t.Fatalf("back:\n%s", back)
	}
}
