package cite

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const cslSample = `[
 {"id": "doe99", "type": "book", "title": "A <i>Fine</i> Book",
  "author": [{"family": "Doe", "given": "Jane"}, {"family": "Gogh", "given": "Vincent", "non-dropping-particle": "van"}],
  "issued": {"date-parts": [[1999, 3]]}, "publisher": "Press", "publisher-place": "Leiden"},
 {"id": 42, "type": "article-journal", "title": "Numbers", "author": [{"literal": "World Health Organization"}],
  "issued": {"date-parts": [["2004"]]}, "container-title": "Journal", "volume": 7, "issue": "2", "page": "1-9"},
 {"id": "raw", "title": "Raw date", "editor": [{"family": "Roe", "given": "R."}], "issued": {"raw": "circa 1850"}},
 {"title": "no id: left out"}
]`

func TestParseCSLJSON(t *testing.T) {
	es, err := ParseCSLJSON([]byte(cslSample))
	if err != nil {
		t.Fatal(err)
	}
	if got := Keys(es); !reflect.DeepEqual(got, []string{"42", "doe99", "raw"}) {
		t.Fatalf("keys %v", got)
	}
	lib := NewLibrary(es)
	d := lib.Get("doe99")
	if d.Title != "A Fine Book" || d.Year != "1999" || d.Place != "Leiden" || d.Authors[1].FamilyName() != "van Gogh" {
		t.Errorf("doe99 = %+v", d)
	}
	if got := d.AuthorShort(); got != "Doe and van Gogh" {
		t.Errorf("short %q", got)
	}
	n := lib.Get("42")
	if n.Volume != "7" || n.Year != "2004" || n.AuthorShort() != "World Health Organization" {
		t.Errorf("42 = %+v", n)
	}
	r := lib.Get("raw")
	if r.Year != "1850" || r.AuthorShort() != "Roe" {
		t.Errorf("raw = %+v", r)
	}
	if !strings.Contains(r.Tip(), "R. Roe (ed.) (1850)\nRaw date") {
		t.Errorf("tip %q", r.Tip())
	}
	if _, err := ParseCSLJSON([]byte(`{"name": "package.json"}`)); err == nil {
		t.Error("a JSON file that is no library read as one")
	}
	if es, err := ParseCSLJSON([]byte(`{"id": "one", "title": "Alone"}`)); err != nil || len(es) != 1 {
		t.Errorf("one item: %v %v", es, err)
	}
}

const bibSample = `
% a comment line with an @ in it: me@example.org
@string{ pup = "Princeton University Press" }
@String(acm = {ACM})
@comment{ @book{ignored, title = {No} } }
@preamble{ "\newcommand{\noop}[1]{}" }

@Book{knuth84,
  author    = {Donald E. Knuth},
  title     = "The {\TeX}book",
  publisher = pup # { (reprint)},
  year      = 1984,
  month     = jun,
  address   = {Reading, MA},
}

@article{goedel31,
  AUTHOR = {G{\"o}del, Kurt and Erd\H{o}s, P\'al and others},
  Title = {{\"U}ber formal unentscheidbare S{\"a}tze --- a test},
  journal = {Monatshefte f\"{u}r Mathematik},
  volume = {38}, number = 1, pages = {173--198},
  date = {1931-12},
}

@InProceedings{child,
  author = {Ludwig van Beethoven and de la Fontaine, Jean and {World Health Organization}},
  title = {A Chapter},
  crossref = {proc},
  pages = {1--10},
}

@proceedings{proc,
  editor = {Jane Doe and Smith, Jr, John},
  title = {Proceedings of Things},
  publisher = acm,
  year = {2001},
  location = {Lyon},
}

@misc{broken,
  title = {This one never closes
@book{after, title = {After the Broken One}, author = {Ana Ñúñez}, year = {2020}}
@online{web, title = {A Page}, url = {https://example.org/a_b}, urldate = {2024-01-01}, author = {Brontë, Charlotte}, date = {2015}}
`

func TestParseBibTeX(t *testing.T) {
	es, err := ParseBibTeX([]byte(bibSample))
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Errorf("the broken entry is not named: %v", err)
	}
	lib := NewLibrary(es)
	if got := Keys(lib.Entries); !reflect.DeepEqual(got, []string{"after", "child", "goedel31", "knuth84", "proc", "web"}) {
		t.Fatalf("keys %v", got)
	}
	k := lib.Get("knuth84")
	if k.Title != "The TeXbook" || k.Publisher != "Princeton University Press (reprint)" || k.Year != "1984" || k.Type != "book" || k.Place != "Reading, MA" {
		t.Errorf("knuth84 = %+v", k)
	}
	if k.Authors[0].Family != "Knuth" || k.Authors[0].Given != "Donald E." {
		t.Errorf("knuth's name %+v", k.Authors[0])
	}
	g := lib.Get("goedel31")
	if g.Title != "Über formal unentscheidbare Sätze — a test" || g.Container != "Monatshefte für Mathematik" || g.Pages != "173–198" || g.Year != "1931" || g.Issue != "1" {
		t.Errorf("goedel31 = %+v", g)
	}
	if n := g.Authors; len(n) != 3 || n[0].Family != "Gödel" || n[1].Family != "Erdős" || n[1].Given != "Pál" || n[2].Literal != "others" {
		t.Errorf("goedel31's names %+v", n)
	}
	if got := g.AuthorShort(); got != "Gödel et al." && got != "Gödel, Erdős, and others" {
		t.Errorf("short %q", got)
	}
	c := lib.Get("child")
	if c.Container != "Proceedings of Things" || c.Publisher != "ACM" || c.Year != "2001" || c.Place != "Lyon" || c.Type != "paper-conference" {
		t.Errorf("child (crossref) = %+v", c)
	}
	if n := c.Authors; len(n) != 3 || n[0].Particle != "van" || n[0].Family != "Beethoven" || n[1].Particle != "de la" || n[1].Family != "Fontaine" || n[1].Given != "Jean" || n[2].Literal != "World Health Organization" {
		t.Errorf("child's names %+v", n)
	}
	if n := c.Editors; len(n) != 2 || n[1].Family != "Smith" || n[1].Suffix != "Jr" || n[1].Given != "John" {
		t.Errorf("child's editors (inherited) %+v", n)
	}
	if a := lib.Get("after"); a.Authors[0].Family != "Ñúñez" {
		t.Errorf("after = %+v", a)
	}
	w := lib.Get("web")
	if w.URL != "https://example.org/a_b" || w.Year != "2015" || w.Type != "webpage" || w.Authors[0].Family != "Brontë" {
		t.Errorf("web = %+v", w)
	}
	if lib.Get("KNUTH84") != k {
		t.Error("a key in another case is not found")
	}
}

func TestDecodeLaTeX(t *testing.T) {
	for in, want := range map[string]string{
		`Schr\"odinger`:             "Schrödinger",
		`\'{e}t\'e`:                 "été",
		`{\c c}a`:                   "ça",
		`\v{S}koda`:                 "Škoda",
		`Stra{\ss}e`:                "Straße",
		`\o{}re \AA{}ngstr\"om`:     "øre Ångström",
		`\emph{Very} \textbf{bold}`: "Very bold",
		`10\% of \$5 \& more`:       "10% of $5 & more",
		"``quoted''":                "“quoted”",
		`a~b`:                       "a b",
		`na\"{\i}ve`:                "naïve",
		`  many   spaces `:          "many spaces",
	} {
		if got := decodeLaTeX(in); got != want {
			t.Errorf("%s: %q, want %q", in, got, want)
		}
	}
}

func TestMatch(t *testing.T) {
	es, _ := ParseBibTeX([]byte(bibSample))
	lib := NewLibrary(es)
	keys := func(es []*Entry) []string {
		var out []string
		for _, e := range es {
			out = append(out, e.Key)
		}
		return out
	}
	if got := keys(lib.Match("", 3)); len(got) != 3 || got[0] != "after" {
		t.Errorf("nothing typed: %v", got)
	}
	if got := keys(lib.Match("go", 0)); len(got) < 1 || got[0] != "goedel31" {
		t.Errorf("go: %v", got)
	}
	if got := keys(lib.Match("godel", 0)); len(got) != 1 || got[0] != "goedel31" { // the author, without the accent
		t.Errorf("godel: %v", got)
	}
	if got := keys(lib.Match("texbook", 0)); len(got) != 1 || got[0] != "knuth84" {
		t.Errorf("texbook (title): %v", got)
	}
	if got := keys(lib.Match("zzz", 0)); len(got) != 0 {
		t.Errorf("zzz: %v", got)
	}
}

func TestFrontOf(t *testing.T) {
	for text, want := range map[string]Front{
		"---\ntitle: X\nbibliography: refs.bib\ncsl: chicago.csl\n---\nText": {Bibliography: "refs.bib", CSL: "chicago.csl"},
		"---\nbibliography: \"my refs.json\"\n---\n":                         {Bibliography: "my refs.json"},
		"---\nbibliography: [a.bib, b.json]\n---\n":                          {Bibliography: "a.bib"},
		"---\nbibliography:\n  - first.bib\n  - second.bib\ntitle: T\n---\n": {Bibliography: "first.bib"},
		"---\nnocite: |\n  @a, @b\n  @c\n---\n":                              {Nocite: []string{"a", "b", "c"}},
		"---\nnocite: \"@*\"\n---\n":                                         {Nocite: []string{"*"}},
		"No front matter\nbibliography: x.bib\n":                             {},
		"---\nbibliography: unclosed.bib\n":                                  {},
	} {
		if got := FrontOf(text); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %+v, want %+v", text, got, want)
		}
	}
}

// fakeFiles reads sources from a map (bundle entries as "bundle:name").
func fakeFiles(m map[string]string) func(Source) ([]byte, error) {
	return func(s Source) ([]byte, error) {
		n := s.Name
		if s.Bundle {
			n = "bundle:" + n
		}
		if d, ok := m[n]; ok {
			return []byte(d), nil
		}
		return nil, os.ErrNotExist
	}
}

func TestResolveOrder(t *testing.T) {
	root := t.TempDir() // an absolute folder on every system
	p := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	docs, home := p("docs"), p("home/me")
	bib := "@book{k, title={T}}"
	csl := `[{"id": "c", "title": "C"}]`
	files := fakeFiles(map[string]string{
		p("docs/refs.bib"):              bib,
		p("docs/novel.json"):            csl,
		p("docs/novel.bib"):             bib,
		p("docs/zotero.bib"):            bib,
		p("docs/package.json"):          `{"name": "x"}`,
		p("home/me/library.json"):       csl,
		"bundle:assets/library.bib":     bib,
		p("docs/b.textbundle/refs.bib"): bib,
	})
	all := []string{"refs.bib", "novel.json", "novel.bib", "zotero.bib", "package.json", "novel.md"}
	for _, tc := range []struct {
		name string
		p    Place
		want string // Source.Name, "" for none
		why  string
		err  bool
	}{
		{"front matter first", Place{Front: "refs.bib", Dir: docs, Stem: "novel", DirFiles: all, Main: "~/library.json", Home: home}, p("docs/refs.bib"), WhyFront, false},
		{"front matter missing is an error, not a fallback", Place{Front: "gone.bib", Dir: docs, DirFiles: all, Main: p("home/me/library.json")}, p("docs/gone.bib"), WhyFront, true},
		{"the document's own name, CSL-JSON first", Place{Dir: docs, Stem: "novel", DirFiles: all}, p("docs/novel.json"), WhyBeside, false},
		{"other .bib files before .json", Place{Dir: docs, Stem: "other", DirFiles: []string{"package.json", "zotero.bib"}}, p("docs/zotero.bib"), WhyBeside, false},
		{"a .json that is no library is passed over", Place{Dir: docs, Stem: "other", DirFiles: []string{"package.json"}, Main: "~/library.json", Home: home}, p("home/me/library.json"), WhyMain, false},
		{"the bundle's assets", Place{Dir: docs, Stem: "b", DirFiles: all, Bundle: []string{"info.json", "text.md", "assets/", "assets/library.bib"}}, "assets/library.bib", WhyBundle, false},
		{"the front matter's file in the bundle", Place{Front: "library.bib", Dir: docs, Bundle: []string{"assets/library.bib"}}, "assets/library.bib", WhyFront, false},
		{"nothing at all", Place{}, "", "", false},
		{"not saved: the main library", Place{Main: p("home/me/library.json")}, p("home/me/library.json"), WhyMain, false},
	} {
		s, lib, err := Resolve(tc.p, files)
		if s.Name != tc.want || s.Why != tc.why || (err != nil) != tc.err {
			t.Errorf("%s: %+v %v, want %s (%s) err %v", tc.name, s, err, tc.want, tc.why, tc.err)
		}
		if err == nil && tc.want != "" && (lib == nil || len(lib.Entries) == 0) {
			t.Errorf("%s: no entries", tc.name)
		}
	}
}

// TestParseHalfWritten: a file read while it is being written again
// says so (the caller keeps the library it had).
func TestParseHalfWritten(t *testing.T) {
	if _, err := Parse("x.json", []byte(`[{"id": "a", "tit`)); err == nil {
		t.Error("half a CSL-JSON file read")
	}
	if _, err := Parse("x.bib", []byte(`@book{a, title = {Hal`)); err == nil {
		t.Error("half a BibTeX file read")
	}
	if _, err := Parse("x.txt", nil); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Errorf("a .txt: %v", err)
	}
}
