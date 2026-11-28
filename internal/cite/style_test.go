// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package cite

import (
	"strings"
	"testing"
)

// styleBib is a library with one entry of each common type, two works
// of one author in one year, one with four authors, one with none.
const styleBib = `
@book{doe20a, author = {Doe, Jane}, title = {Winter Light}, year = 2020, publisher = {Harbor Press}, address = {Boston}, edition = {2}}
@book{doe20b, author = {Doe, Jane}, title = {Autumn Rain}, year = 2020, publisher = {Harbor Press}, address = {Boston}}
@book{doe18, author = {Doe, Jane}, title = {Early Work}, year = 2018, publisher = {Harbor Press}}
@article{roe01, author = {Roe, Richard and Poe, Ann}, title = {On Tides}, journal = {Journal of the Sea}, volume = 12, number = 3, pages = {45--67}, year = 2001, doi = {10.1000/tides}}
@incollection{moe15, author = {Moe, Mary}, title = {A Chapter}, booktitle = {The Big Book}, editor = {Smith, Sam and Jones, Jo}, pages = {101-120}, publisher = {Univ. Press}, address = {Leiden}, year = 2015}
@phdthesis{lee10, author = {Lee, Li}, title = {Deep Waters}, school = {University of Utrecht}, year = 2010}
@online{web19, author = {Kim, Kay}, title = {A Page}, howpublished = {Example Site}, url = {https://example.org/page}, year = 2019}
@techreport{rep05, author = {{World Health Organization}}, title = {Health Report}, institution = {WHO}, address = {Geneva}, year = 2005}
@misc{misc99, author = {Fox, Fay}, title = {Some Data}, year = 1999}
@book{four12, author = {Ash, A. and Birch, B. and Cedar, C. and Dogwood, D.}, title = {Trees}, year = 2012, publisher = {Wood}}
@book{anon, title = {Anonymous Classic: A Subtitle}, year = 1800}
@book{nodate, author = {Nobody, Ned}, title = {Timeless}}
`

func styleLib(t *testing.T) *Library {
	t.Helper()
	es, err := ParseBibTeX([]byte(styleBib))
	if err != nil {
		t.Fatal(err)
	}
	return NewLibrary(es)
}

// inText is the first paragraph Render makes of md: its citations.
func inText(lib *Library, md string) string {
	out := Render(md, lib, nil)
	first, _, _ := strings.Cut(out, "\n\n")
	return first
}

func TestStyleInText(t *testing.T) {
	lib := styleLib(t)
	for _, c := range []struct{ in, want string }{
		{"[@roe01]", "(Roe and Poe 2001)"},
		{"[@roe01, p. 12]", "(Roe and Poe 2001, 12)"},
		{"[@roe01, pp. 12-15]", "(Roe and Poe 2001, 12–15)"},
		{"[@moe15, chap. 2]", "(Moe 2015, chap. 2)"},
		{"[see @roe01; @moe15]", "(see Roe and Poe 2001; Moe 2015)"},
		{"[see @roe01, p. 3; also @moe15, 7, emphasis added]", "(see Roe and Poe 2001, 3; also Moe 2015, 7, emphasis added)"},
		{"[-@roe01]", "(2001)"},
		{"[-@roe01, p. 9]", "(2001, 9)"},
		{"@roe01 says so.", "Roe and Poe (2001) says so."},
		{"@roe01 [p. 2] says so.", "Roe and Poe (2001, 2) says so."},
		{"@moe15 [chap. 4] too", "Moe (2015, chap. 4) too"},
		{"[@doe18; @doe20a]", "(Doe 2018, 2020)"},                  // one 2020 cited: no letter
		{"[@doe18; @doe20a; @doe20b]", "(Doe 2018, 2020b, 2020a)"}, // Autumn (doe20b) before Winter: 2020a
		{"[@doe18, 4; @doe20b, 9; @doe20a]", "(Doe 2018, 4; 2020a, 9; 2020b)"},
		{"[@doe20a] and [@doe20b]", "(Doe 2020b) and (Doe 2020a)"},
		{"[@four12]", "(Ash et al. 2012)"},
		{"[@anon]", "(*Anonymous Classic* 1800)"},
		{"[@nodate]", "(Nobody n.d.)"},
		{"[@rep05]", "(World Health Organization 2005)"},
		{"[@nobody]", "(nobody?)"},
		{"[@roe01; @nobody]", "(Roe and Poe 2001; nobody?)"},
		{"@nobody says", "(nobody?) says"},
		{"Mail me@example.org, `[@roe01]` stays.", "Mail me@example.org, `[@roe01]` stays."},
		{"*In emphasis* [@lee10]", "*In emphasis* (Lee 2010)"},
	} {
		if got := inText(lib, c.in); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.in, got, c.want)
		}
	}
}

// entryOf is the bibliography's entry for key in what Render makes of
// a document citing it alone.
func entryOf(t *testing.T, lib *Library, key string) string {
	t.Helper()
	out := Render("[@"+key+"]", lib, nil)
	_, refs, ok := strings.Cut(out, "# References\n\n")
	if !ok {
		t.Fatalf("%s: no references in %q", key, out)
	}
	return strings.TrimPrefix(strings.TrimSpace(refs), EntryMark)
}

func TestStyleBibliographyTypes(t *testing.T) {
	lib := styleLib(t)
	for _, c := range []struct{ key, want string }{
		{"doe20a", "Doe, Jane. 2020. *Winter Light*. 2nd ed. Boston: Harbor Press."},
		{"roe01", "Roe, Richard, and Ann Poe. 2001. “On Tides.” *Journal of the Sea* 12 (3): 45–67. <https://doi.org/10.1000/tides>."},
		{"moe15", "Moe, Mary. 2015. “A Chapter.” In *The Big Book*, edited by Sam Smith and Jo Jones, 101–120. Leiden: Univ. Press."},
		{"lee10", "Lee, Li. 2010. “Deep Waters.” PhD diss., University of Utrecht."},
		{"web19", "Kim, Kay. 2019. “A Page.” Example Site. <https://example.org/page>."},
		{"rep05", "World Health Organization. 2005. *Health Report*. Geneva: WHO."},
		{"misc99", "Fox, Fay. 1999. *Some Data*."},
		{"four12", "Ash, A., B. Birch, C. Cedar, and D. Dogwood. 2012. *Trees*. Wood."},
		{"anon", "*Anonymous Classic: A Subtitle*. 1800."},
		{"nodate", "Nobody, Ned. n.d. *Timeless*."},
	} {
		if got := entryOf(t, lib, c.key); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.key, got, c.want)
		}
	}
}

func TestStyleEditorsAndCSL(t *testing.T) {
	es, err := ParseCSLJSON([]byte(`[
 {"id": "ed", "type": "book", "title": "Collected", "editor": [{"family": "Roe", "given": "R."}, {"family": "Poe", "given": "P."}], "issued": {"date-parts": [[1990]]}, "publisher": "P"},
 {"id": "th", "type": "thesis", "genre": "Master's thesis", "title": "Small Waters", "author": [{"family": "Ng", "given": "N."}], "publisher": "Leiden University", "issued": {"date-parts": [[2011]]}},
 {"id": "vg", "type": "book", "title": "Letters", "author": [{"family": "Gogh", "given": "Vincent", "non-dropping-particle": "van"}], "issued": {"date-parts": [[1890]]}},
 {"id": "q", "type": "article-journal", "title": "Why Tides?", "author": [{"family": "Wu", "given": "W."}], "container-title": "Sea", "page": "3-4", "issued": {"date-parts": [[2002]]}}
]`))
	if err != nil {
		t.Fatal(err)
	}
	lib := NewLibrary(es)
	for _, c := range []struct{ key, want string }{
		{"ed", "Roe, R., and P. Poe, eds. 1990. *Collected*. P."},
		{"th", "Ng, N. 2011. “Small Waters.” Master's thesis, Leiden University."},
		{"q", "Wu, W. 2002. “Why Tides?” *Sea*, 3–4."},
		{"vg", "Gogh, Vincent van. 1890. *Letters*."}, // listed and sorted under G
	} {
		if got := entryOf(t, lib, c.key); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.key, got, c.want)
		}
	}
	if got := inText(lib, "@vg"); got != "van Gogh (1890)" {
		t.Errorf("a particle in the text: %s", got)
	}
	if got := inText(lib, "[@ed]"); got != "(Roe and Poe 1990)" {
		t.Errorf("editors in the text: %s", got)
	}
}

func TestStyleBibliographyOrderAndNocite(t *testing.T) {
	lib := styleLib(t)
	out := Render("# Essay\n\nFirst [@roe01], then [@doe20b; @doe20a].\n", lib, []string{"misc99", "nobody"})
	_, refs, ok := strings.Cut(out, "## References\n\n") // one level-1 heading is the title: under it
	if !ok {
		t.Fatalf("no References heading:\n%s", out)
	}
	for _, p := range strings.Split(strings.TrimSpace(refs), "\n\n") {
		if !strings.HasPrefix(p, EntryMark) {
			t.Errorf("entry without the mark: %q", p)
		}
	}
	at := -1
	for _, w := range []string{"Doe, Jane. 2020a.", "Doe, Jane. 2020b.", "Fox, Fay. 1999.", "Roe, Richard, and Ann Poe. 2001."} {
		i := strings.Index(refs, w)
		if i <= at {
			t.Errorf("%q out of order (or missing):\n%s", w, refs)
		}
		at = i
	}
	// 2020a is the first in the bibliography's order (by title: Autumn before Winter)
	if !strings.Contains(out, "(Doe 2020a, 2020b)") && !strings.Contains(out, "(Doe 2020b, 2020a)") {
		t.Errorf("collapsed years: %s", out)
	}
	if !strings.Contains(refs, "2020a. *Autumn Rain*") || !strings.Contains(refs, "2020b. *Winter Light*") {
		t.Errorf("disambiguation by title:\n%s", refs)
	}
	if strings.Contains(refs, "nobody") {
		t.Errorf("a key the library lacks is listed:\n%s", refs)
	}

	// @* lists the whole library
	all := Render("Nothing cited.\n", lib, []string{"*"})
	if n := strings.Count(all, EntryMark); n != len(lib.Entries) {
		t.Errorf("nocite @*: %d entries, want %d", n, len(lib.Entries))
	}
}

func TestStyleBibliographyPlace(t *testing.T) {
	lib := styleLib(t)
	// a {#refs} div: there, its fences gone
	out := Render("Text [@misc99].\n\n::: {#refs}\n:::\n\n## Appendix\n\nMore.\n", lib, nil)
	if strings.Contains(out, ":::") || strings.Contains(out, "# References") {
		t.Errorf("refs div:\n%s", out)
	}
	if i, j := strings.Index(out, EntryMark), strings.Index(out, "## Appendix"); i < 0 || i > j {
		t.Errorf("refs not in the div's place:\n%s", out)
	}
	// a References heading: at its section's end, before the next one
	out = Render("# A\n\nText [@misc99].\n\n# References\n\nSee below.\n\n# Index\n\nx\n", lib, nil)
	if i, j, k := strings.Index(out, "See below."), strings.Index(out, EntryMark), strings.Index(out, "# Index"); !(i < j && j < k) {
		t.Errorf("refs not in the References section:\n%s", out)
	}
	if strings.Count(out, "References") != 1 {
		t.Errorf("a second References heading:\n%s", out)
	}
	// setext Bibliography
	out = Render("Text [@misc99].\n\nBibliography\n------------\n", lib, nil)
	if strings.Contains(out, "# References") || !strings.HasSuffix(strings.TrimSpace(out), "*Some Data*.") {
		t.Errorf("setext bibliography heading:\n%s", out)
	}
	// several chapters: References as one of them
	out = Render("# One\n\n[@misc99]\n\n# Two\n\nx\n", lib, nil)
	if !strings.Contains(out, "\n# References\n") {
		t.Errorf("chapter-level heading:\n%s", out)
	}
	// code is no citation; no library, no change
	md := "```\n[@misc99]\n```\n"
	if out := Render(md, lib, nil); out != md {
		t.Errorf("code cited:\n%s", out)
	}
	if out := Render("[@misc99]", nil, nil); out != "[@misc99]" {
		t.Errorf("no library: %s", out)
	}
	// only keys the library lacks: no bibliography
	if out := Render("[@nobody]", lib, nil); out != "(nobody?)" {
		t.Errorf("missing only: %q", out)
	}
}

func TestStyleEscapes(t *testing.T) {
	lib := NewLibrary([]*Entry{{Key: "x", Type: "book", Title: "Stars *and* [Bars]", Authors: []Name{{Family: "O_Brien", Given: "Pat"}}, Year: "2000"}})
	out := Render("[@x]", lib, nil)
	if !strings.HasPrefix(out, `(O\_Brien 2000)`) || !strings.Contains(out, `*Stars \*and\* \[Bars\]*`) {
		t.Errorf("escapes:\n%s", out)
	}
}
