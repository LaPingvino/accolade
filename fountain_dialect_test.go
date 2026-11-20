package main

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/fountain"
)

// dialectSample is a script in lang using every one of its heading words,
// with English headings it does not know as action and as a character.
func dialectSample(lang string) string {
	var b strings.Builder
	b.WriteString("Title: La Domo\nAuthor: Iu\n\n")
	for _, w := range sceneHeaders(lang) {
		last, _ := utf8.DecodeLastRuneInString(w)
		switch {
		case last == '.':
			b.WriteString(w + " PLACE - DAY\n\n")
		case unicode.IsLetter(last) || unicode.IsDigit(last):
			b.WriteString(w + ". PLACE - DAY\n\n")
		default:
			b.WriteString(w + "PLACE\n\n")
		}
		b.WriteString("Something happens.\n\n")
	}
	for _, w := range sceneHeaders("en") {
		if sceneIn(w+". X", lang) {
			continue
		}
		b.WriteString(w + ". KITCHEN is written on the wall.\n\n")    // action
		b.WriteString(strings.ToUpper(w) + ". BOB\nHello there.\n\n") // a character
	}
	b.WriteString("[[a note]] ")
	b.WriteString(sceneHeaders(lang)[0])
	if !strings.HasSuffix(sceneHeaders(lang)[0], ".") {
		b.WriteString(".")
	}
	b.WriteString(" GARDEN\n\n/* Notes\nINT. not a heading in the boneyard\n*/\n")
	return b.String()
}

func typesOf(text, lang string) []string {
	var out []string
	for _, l := range fountain.Parse(sceneHeaders(lang), strings.NewReader(text)) {
		out = append(out, l.Type)
	}
	return out
}

func TestDialectFountainRoundTrip(t *testing.T) {
	for _, lang := range scriptLanguages() {
		native := dialectSample(lang)
		forced := englishFountain(native, lang)
		if back := dialectFountain(forced, lang); back != native {
			t.Errorf("%s: round trip changed the text:\n%s\n---\n%s", lang, native, back)
		}
		mine, english := typesOf(native, lang), typesOf(forced, "en")
		if strings.Join(mine, " ") != strings.Join(english, " ") {
			t.Errorf("%s: standard Fountain reads other elements:\n%v\n%v\n%s", lang, mine, english, forced)
		}
		if lang == "en" && forced != native {
			t.Errorf("English changed")
		}
	}
}

func TestDialectFountainMarks(t *testing.T) {
	native := "EN. DOMO - TAGO\n\nINT. is how English starts a heading.\n\nINT. BOB\nSaluton.\n\n!EN. needed by Esperanto too\n\n.INT. A HEADING ON PURPOSE\n"
	forced := englishFountain(native, "eo")
	want := ".EN. DOMO - TAGO\n\n!INT. is how English starts a heading.\n\n@INT. BOB\nSaluton.\n\n!EN. needed by Esperanto too\n\n.INT. A HEADING ON PURPOSE\n"
	if forced != want {
		t.Fatalf("forced:\n%s\nwant:\n%s", forced, want)
	}
	if back := dialectFountain(forced, "eo"); back != native {
		t.Fatalf("back:\n%s", back)
	}
	// another app's text: a mark it needs stays
	other := "!INT. KITCHEN\nBOB\n"
	if back := dialectFountain(other, "eo"); !strings.HasPrefix(back, "!INT.") {
		t.Errorf("a needed mark went: %q", back)
	}
}
