package cluster

import (
	"bytes"
	"testing"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	lexfont "github.com/LaPingvino/lexington/font"
)

// advance is how many cells Courier Badi (data, one of its faces) draws s
// in, shaped right to left as Arabic.
func advance(t *testing.T, data []byte, s string) int {
	t.Helper()
	face, err := font.ParseTTF(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	rs := []rune(s)
	out := (&shaping.HarfbuzzShaper{}).Shape(shaping.Input{Text: rs, RunEnd: len(rs),
		Direction: di.DirectionRTL, Face: face, Size: fixed.I(1000), Script: language.Arabic,
		Language: language.NewLanguage("ar")})
	return (out.Advance.Round() + 300) / 600
}

// Width counts what the font draws: a lam-alef one cell, also with vowel
// marks on the lam (Courier Badi 1.109), a lam and an alef the font keeps
// apart (a tatweel or a joiner between, alef wasla) two; in each of
// Courier Badi's faces.
func TestWidthIsTheFonts(t *testing.T) {
	faces := map[string][]byte{"Regular": lexfont.CourierBadiRegular, "Bold": lexfont.CourierBadiBold,
		"Italic": lexfont.CourierBadiItalic, "BoldItalic": lexfont.CourierBadiBoldItalic}
	cases := []struct {
		s    string
		want int
	}{
		{"لا", 1}, {"لآ", 1}, {"لأ", 1}, {"لإ", 1}, {"إلا", 2}, {"بلا", 2}, {"السلام", 5},
		{"لاَ", 1}, {"تلاشي", 4}, {"لأن", 2},
		{"لَا", 1}, {"لّا", 1}, {"لٰا", 1}, {"لّٰا", 1}, {"فَلَّا", 2}, {"السَّلَامُ", 5},
		{"لَـا", 3}, {"لـا", 3}, {"ل\u200dا", 2}, {"ل\u200cا", 2}, {"لٱ", 2}, {"ڵا", 2}, {"سل", 2},
	}
	for name, data := range faces {
		for _, c := range cases {
			if got := Width(c.s); got != c.want {
				t.Errorf("Width(%q) = %d, want %d", c.s, got, c.want)
			}
			if got := advance(t, data, c.s); got != c.want {
				t.Errorf("%s draws %q in %d cells, want %d", name, c.s, got, c.want)
			}
		}
	}
}

// A lam-alef is one cluster: no boundary at its alef, its end after it.
func TestLamAlefCluster(t *testing.T) {
	rs := []rune("إلاَ ب")
	if Boundary(rs, 2) || !Boundary(rs, 1) || Boundary(rs, 3) || !Boundary(rs, 4) {
		t.Errorf("boundaries in %q: %v %v %v %v", string(rs), Boundary(rs, 1), Boundary(rs, 2), Boundary(rs, 3), Boundary(rs, 4))
	}
	if End(rs, 1) != 4 {
		t.Errorf("End(1) = %d, want 4", End(rs, 1))
	}
	if !Joins('ل', 'ا') || Joins('ل', 'ٱ') || Joins('ب', 'ا') || Joins('ـ', 'ا') || Joins('\u200c', 'ا') {
		t.Error("Joins")
	}
	if !Joins('َ', 'ا') { // a quick filter: Boundary decides
		t.Error("Joins: an alef after a mark")
	}
}

// With vowel marks on its lam, a lam-alef is still one cluster: no
// boundary at its marks or its alef; after a vowelled beh, or a tatweel,
// the alef starts its own.
func TestLamAlefMarksCluster(t *testing.T) {
	rs := []rune("فَلَّا بَا لَـا")
	//            0 1 2 3 4 5 6 7 8 9 10 11 12 13 14
	want := map[int]bool{1: false, 2: true, 3: false, 4: false, 5: false, 6: true, 7: true, 8: false, 9: true,
		10: true, 11: true, 12: false, 13: true, 14: true}
	for i, w := range want {
		if got := Boundary(rs, i); got != w {
			t.Errorf("Boundary(%d) (%q) = %v, want %v", i, string(rs[i]), got, w)
		}
	}
	if End(rs, 2) != 6 {
		t.Errorf("End(2) = %d, want 6 (لَّا)", End(rs, 2))
	}
	if c := Cells(rs[2:6]); c[0]+c[1]+c[2]+c[3] != 1 || c[0] != 1 {
		t.Errorf("لَّا's cells %v, want its lam the one", c)
	}
}
