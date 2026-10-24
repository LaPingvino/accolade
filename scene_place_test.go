package main

import "testing"

func TestPlaceWords(t *testing.T) {
	for lang, want := range map[string][3]string{
		"en": {"INT.", "EXT.", "INT./EXT."},
		"nl": {"BIN.", "BUI.", "BIN./BUI."},
		"ru": {"ИНТ.", "НАТ.", "ИНТ/НАТ."},
	} {
		in, out, both, ok := placeWords(sceneHeaders(lang))
		if !ok || [3]string{in, out, both} != want {
			t.Errorf("%s: %q %q %q", lang, in, out, both)
		}
	}
	scenes := sceneHeaders("en")
	for _, c := range [][3]string{
		{"", "INT.", "INT. "},
		{"INT. KITCHEN - DAY", "EXT.", "EXT. KITCHEN - DAY"},
		{"INT./EXT. CAR - NIGHT", "INT.", "INT. CAR - NIGHT"},
		{".THE ROOF", "EXT.", "EXT. THE ROOF"},
		{"kitchen", "INT.", "INT. kitchen"},
	} {
		if got := withPlace(c[0], c[1], scenes); got != c[2] {
			t.Errorf("withPlace(%q, %q) = %q", c[0], c[1], got)
		}
	}
}
