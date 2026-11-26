package cite

import (
	"strings"
	"testing"
)

// cslBib is the fixture the CSL tests cite from; their expectations
// were checked against Pandoc's --citeproc with the same .csl files
// (italics aside, and NLM's "In:," where Pandoc has "In,").
const cslBib = `
@article{doe99,
  author = {Doe, John and Roe, Richard},
  title = {A Study of Things},
  journal = {Journal of Studies},
  year = {1999},
  volume = {12},
  number = {3},
  pages = {101--115},
  doi = {10.1000/xyz}
}
@book{smith2001,
  author = {Smith, Anna Maria},
  title = {The Book of Words},
  publisher = {Oxford University Press},
  address = {Oxford},
  year = {2001},
  edition = {2}
}
@incollection{lee2010,
  author = {Lee, Kim and Park, Ji and Choi, Min and Kang, Su},
  title = {Chapters in Context},
  booktitle = {Handbook of Context},
  editor = {Brown, Bob},
  publisher = {Springer},
  address = {Berlin},
  year = {2010},
  pages = {45--67}
}
@book{smith2001b,
  author = {Smith, Anna Maria},
  title = {More Words},
  publisher = {Oxford University Press},
  address = {Oxford},
  year = {2001}
}
@online{web2020,
  author = {Green, Pat},
  title = {A Web Page},
  url = {https://example.org/page},
  year = {2020},
  urldate = {2021-03-04}
}
`

const cslDoc = `As shown [@doe99, p. 12] and later [see @smith2001; @smith2001b].

@lee2010 says so, and [@web2020; @doe99; @smith2001].

A note.[^1]

[^1]: In a note [@lee2010, chap. 2].

# References
`

func cslRender(t *testing.T, style, doc string) string {
	t.Helper()
	lib, err := Parse("lib.bib", []byte(cslBib))
	if err != nil {
		t.Fatal(err)
	}
	st := BuiltinStyle(style)
	if st == nil {
		t.Fatalf("no style %s", style)
	}
	return strings.ReplaceAll(RenderWith(doc, lib, nil, st), EntryMark, "")
}

func TestCSLStyles(t *testing.T) {
	for style, want := range map[string][]string{
		"apa": {
			`As shown (Doe & Roe, 1999, p. 12) and later (see Smith, 2001b, 2001a).`,
			`Lee et al. (2010) says so, and (Doe & Roe, 1999; Green, 2020; Smith, 2001b).`,
			`[^1]: In a note (Lee et al., 2010, Chapter 2).`,
			"# References\n\n\nDoe, J., & Roe, R. (1999). A study of things. *Journal of Studies*, *12*(3), 101–115. https://doi.org/10.1000/xyz\n\nGreen, P.",
			`Lee, K., Park, J., Choi, M., & Kang, S. (2010). Chapters in context. In B. Brown (Ed.), *Handbook of context* (pp. 45–67). Springer.`,
			"Smith, A. M. (2001a). *More words*. Oxford University Press.\n\nSmith, A. M. (2001b). *The book of words* (2nd ed.). Oxford University Press.",
		},
		"chicago-author-date": {
			`As shown (Doe and Roe 1999, 12) and later (see Smith 2001b, 2001a).`,
			`Lee et al. (2010) says so, and (Green 2020; Doe and Roe 1999; Smith 2001b).`,
			`Doe, John, and Richard Roe. 1999. “A Study of Things.” *Journal of Studies* 12 (3): 101–15. https://doi.org/10.1000/xyz.`,
			`Smith, Anna Maria. 2001b. *The Book of Words*. 2nd ed. Oxford University Press.`,
		},
		"ieee": {
			`As shown [1, p. 12] and later see [2], [3].`,
			`[4] says so, and [1], [2], [5].`,
			`[^1]: In a note [4, Ch. 2].`,
			`[1] J. Doe and R. Roe, “A study of things,” *Journal of Studies*, vol. 12, no. 3, pp. 101–115, 1999, doi: 10.1000/xyz.`,
			`[4] K. Lee, J. Park, M. Choi, and S. Kang, “Chapters in context,” in *Handbook of context*, B. Brown, Ed., Berlin: Springer, 2010, pp. 45–67.`,
			`[5] P. Green, “A web page.” Accessed: Mar. 04, 2021. [Online]. Available: https://example.org/page`,
		},
		"vancouver": {
			`As shown (1) and later (see 2,3).`,
			`(4) says so, and (1,2,5).`,
			`1. Doe J, Roe R. A study of things. Journal of Studies. 1999;12(3):101–15. doi:10.1000/xyz`,
			`2. Smith AM. The book of words. 2nd ed. Oxford: Oxford University Press; 2001.`,
			`5. Green P. A web page [Internet]. 2020 [cited 2021 Mar 4]. Available from: https://example.org/page`,
		},
	} {
		got := cslRender(t, style, cslDoc)
		for _, w := range want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: want\n%s\nin\n%s", style, w, got)
			}
		}
	}
}

// A numeric style numbers a footnote's citation where its mark is,
// collapses runs (1–3) and gives the bibliography in that order.
func TestCSLNumbering(t *testing.T) {
	doc := "First.[^n] Then [@doe99; @smith2001; @smith2001b; @web2020].\n\n[^n]: See [@lee2010].\n"
	got := cslRender(t, "vancouver", doc)
	for _, w := range []string{`Then (2–5).`, `[^n]: See (1).`, `1. Lee K`} {
		if !strings.Contains(got, w) {
			t.Errorf("want %q in\n%s", w, got)
		}
	}
	got = cslRender(t, "vancouver", "Some long text [@doe99] a.[^1] b.[^2]\n\n[^2]: [@web2020]\n[^1]: [@smith2001]\n")
	for _, w := range []string{`[^1]: (2)`, `[^2]: (3)`} {
		if !strings.Contains(got, w) {
			t.Errorf("two marks on a line: want %q in\n%s", w, got)
		}
	}
	if got := cslRender(t, "ieee", "[@nope] and [-@doe99]"); !strings.Contains(got, `(nope?) and [1]`) {
		t.Errorf("a missing key, the author left out:\n%s", got)
	}
}

// CSL-JSON's dates and other variables reach the style.
func TestCSLJSONVariables(t *testing.T) {
	lib, err := Parse("lib.json", []byte(`[{"id":"w","type":"webpage","title":"Page","author":[{"family":"Green","given":"Pat"}],
		"issued":{"date-parts":[[2020,5,6]]},"accessed":{"date-parts":[[2021,3,4]]},"URL":"https://example.org","container-title":"Site"}]`))
	if err != nil {
		t.Fatal(err)
	}
	got := RenderWith("[@w]", lib, nil, BuiltinStyle("apa"))
	if !strings.Contains(got, "Green, P. (2020, May 6). *Page*. Site. https://example.org") {
		t.Errorf("got\n%s", got)
	}
}

func TestBuiltinStyleNames(t *testing.T) {
	for _, n := range []string{"apa", "APA.csl", "styles/ieee.csl", "vancouver", "nlm-citation-sequence.csl", "chicago-author-date"} {
		if BuiltinStyle(n) == nil {
			t.Errorf("%s: not found", n)
		}
	}
	if BuiltinCSL("harvard-x") != nil || StyleOf("", nil) != nil || StyleOf("nowhere.csl", nil) != nil {
		t.Error("a style that is not there")
	}
	if _, err := ParseStyle([]byte("<html/>")); err == nil {
		t.Error("not a style")
	}
}

// cslBib2 has what the first lacks: no author, editors only, no year,
// particles, an institution, eight authors, a thesis, a report's number.
const cslBib2 = `
@book{noauth, title = {Anonymous Classic Works}, publisher = {Penguin}, address = {London}, year = {1990}}
@book{edonly, editor = {Miller, Sue and Jones, Tom}, title = {Edited Volume on Things}, publisher = {MIT Press}, address = {Cambridge, MA}, year = {2015}}
@book{noyear, author = {Black, Jack}, title = {Undated Musings}, publisher = {Self}}
@article{vander, author = {van der Berg, Jan and Doe, Jane}, title = {Particles in Names?}, journal = {Name Studies}, year = {2018}, volume = {3}, pages = {1--9}}
@techreport{inst, author = {{World Health Organization}}, title = {Global Report... and More}, institution = {WHO}, address = {Geneva}, year = {2019}, number = {TR-42}}
@phdthesis{thesis, author = {White, Walter}, title = {On Chemistry}, school = {University of New Mexico}, year = {2008}}
@article{many, author = {A, Aa and B, Bb and C, Cc and D, Dd and E, Ee and F, Ff and G, Gg and H, Hh}, title = {Eight Authors}, journal = {Big Science}, year = {2020}, volume = {1}, pages = {100--200}}
`

func TestCSLHarderEntries(t *testing.T) {
	lib, err := Parse("lib2.bib", []byte(cslBib2))
	if err != nil {
		t.Fatal(err)
	}
	doc := "Cites [@noauth; @edonly; @noyear; @vander; @inst; @thesis; @many].\n\n# References\n"
	for style, want := range map[string][]string{
		"apa": {
			"Cites (A et al., 2020; *Anonymous Classic Works*, 1990; Berg & Doe, 2018; Black, n.d.; Miller & Jones, 2015; White, 2008; World Health Organization, 2019).",
			"A, A., B, B., C, C., D, D., E, E., F, F., G, G., & H, H. (2020). Eight authors. *Big Science*, *1*, 100–200.",
			"*Anonymous classic works*. (1990). Penguin.", "Berg, J. van der, & Doe, J. (2018).", "Black, J. (n.d.).",
			"Miller, S., & Jones, T. (Eds.). (2015).", "White, W. (2008). *On chemistry* [PhD thesis]. University of New Mexico.",
			"(TR-42). WHO.",
		},
		"chicago-author-date": {
			"Cites (*Anonymous Classic Works* 1990; Miller and Jones 2015; Black, n.d.; Berg and Doe 2018; World Health Organization 2019; White 2008; A et al. 2020).",
			"A, Aa, Bb B, Cc C, et al. 2020.", "Berg, Jan van der, and Jane Doe. 2018.", "Miller, Sue, and Tom Jones, eds. 2015.",
			"“On Chemistry.” PhD thesis, University of New Mexico.",
		},
		"ieee": {"[2] S. Miller and T. Jones, Eds., *Edited volume on things*. Cambridge, MA: MIT Press, 2015.",
			"[4] J. van der Berg and J. Doe, “Particles in names?” *Name Studies*", "[7] A. A *et al.*, “Eight authors,”"},
		"vancouver": {"Cites (1–7).", "4. Berg J van der, Doe J.", "7. A A, B B, C C, D D, E E, F F, et al. Eight authors."},
	} {
		got := strings.ReplaceAll(RenderWith(doc, lib, nil, BuiltinStyle(style)), EntryMark, "")
		for _, w := range want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: want\n%s\nin\n%s", style, w, got)
			}
		}
	}
}
