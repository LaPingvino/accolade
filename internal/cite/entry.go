// Package cite reads a document's bibliography (the library its
// citations name): CSL-JSON, as Zotero's Better BibTeX exports it, and
// BibTeX/BibLaTeX through a small parser of its own. It finds which file
// a document cites from (Resolve), matches what is typed after @ to its
// entries (Library.Match) and says what an entry is in a line (Entry.Tip).
//
// Nothing here draws; the editor, its @ list and the exports use it.
package cite

import (
	"strings"
)

// Name is one author or editor: a family and given name, or a literal
// one (an organisation: {World Health Organization}).
type Name struct {
	Family, Given string
	Particle      string // von, van der, de: before the family name
	Suffix        string // Jr., III
	Literal       string // the whole name as is (no family/given)
}

// FamilyName is the name a citation shows: "van Gogh", "WHO".
func (n Name) FamilyName() string {
	if n.Literal != "" {
		return n.Literal
	}
	return strings.TrimSpace(n.Particle + " " + n.Family)
}

// String is the name as written in a list: "Vincent van Gogh".
func (n Name) String() string {
	if n.Literal != "" {
		return n.Literal
	}
	s := strings.TrimSpace(n.Given + " " + n.FamilyName())
	if n.Suffix != "" {
		s += ", " + n.Suffix
	}
	return s
}

// Inverted is the name family first, as a bibliography lists the first
// author: "Gogh, Vincent van" is CSL's way; this keeps the particle
// with the family name: "van Gogh, Vincent".
func (n Name) Inverted() string {
	if n.Literal != "" {
		return n.Literal
	}
	s := n.FamilyName()
	if n.Given != "" {
		s += ", " + n.Given
	}
	if n.Suffix != "" {
		s += ", " + n.Suffix
	}
	return s
}

// Entry is one work of the library, with the fields a citation and a
// short author-date reference need. Type is CSL's (book, article-journal,
// chapter, paper-conference, thesis, report, webpage, ...).
type Entry struct {
	Key       string
	Type      string
	Title     string
	Authors   []Name
	Editors   []Name
	Year      string // "1999", "n.d." never: "" when there is none
	Container string // the journal, the book a chapter is in, the proceedings
	Publisher string
	Place     string
	Volume    string
	Issue     string
	Pages     string
	Edition   string
	Genre     string // a thesis's kind (PhD diss.), a report's
	DOI       string
	URL       string
	Note      string
	// More are the other CSL variables by CSL's names (number,
	// collection-title, ISBN, title-short …), MoreNames the other name
	// variables (translator, container-author …) and Dates the dates
	// (issued, accessed …) as year, month, day (0: not given): what the
	// CSL styles read beyond the fields above.
	More      map[string]string
	MoreNames map[string][]Name
	Dates     map[string][3]int
}

// Creators are who the work is cited by: its authors, else its editors.
func (e *Entry) Creators() []Name {
	if len(e.Authors) > 0 {
		return e.Authors
	}
	return e.Editors
}

// AuthorShort is who the work is by in a citation: "Doe", "Doe and
// Roe", "Doe et al."; the title's start when there is no one.
func (e *Entry) AuthorShort() string {
	ns := e.Creators()
	switch len(ns) {
	case 0:
		return ""
	case 1:
		return ns[0].FamilyName()
	case 2:
		return ns[0].FamilyName() + " and " + ns[1].FamilyName()
	case 3:
		return ns[0].FamilyName() + ", " + ns[1].FamilyName() + ", and " + ns[2].FamilyName()
	}
	return ns[0].FamilyName() + " et al."
}

// Line is the entry in one line, for the @ list and the hover: author,
// year, title ("Doe 1999 · A Title").
func (e *Entry) Line() string {
	var parts []string
	who := e.AuthorShort()
	if e.Year != "" {
		who = strings.TrimSpace(who + " " + e.Year)
	}
	if who != "" {
		parts = append(parts, who)
	}
	if e.Title != "" {
		parts = append(parts, e.Title)
	}
	return strings.Join(parts, " · ")
}

// Tip is the entry as the hover over a citation shows it: who, when,
// the title, where it appeared.
func (e *Entry) Tip() string {
	var b strings.Builder
	ns := e.Creators()
	var names []string
	for i, n := range ns {
		if i == 6 && len(ns) > 7 {
			names = append(names, "et al.")
			break
		}
		names = append(names, n.String())
	}
	who := strings.Join(names, ", ")
	if len(e.Authors) == 0 && len(e.Editors) > 0 {
		who += " (ed.)"
	}
	b.WriteString(strings.TrimSpace(who))
	if e.Year != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString("(" + e.Year + ")")
	}
	if e.Title != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(e.Title)
	}
	var where []string
	if e.Container != "" {
		c := e.Container
		if e.Volume != "" {
			c += " " + e.Volume
			if e.Issue != "" {
				c += " (" + e.Issue + ")"
			}
		}
		if e.Pages != "" {
			c += ": " + e.Pages
		}
		where = append(where, "In "+c)
	}
	pub := e.Publisher
	if e.Place != "" && pub != "" {
		pub = e.Place + ": " + pub
	}
	if pub != "" {
		where = append(where, pub)
	}
	if len(where) > 0 {
		b.WriteString("\n" + strings.Join(where, ". "))
	}
	if e.DOI != "" {
		b.WriteString("\nhttps://doi.org/" + strings.TrimPrefix(e.DOI, "https://doi.org/"))
	} else if e.URL != "" {
		b.WriteString("\n" + e.URL)
	}
	return b.String()
}

// setMore sets a CSL variable of More ("" leaves it out).
func (e *Entry) setMore(name, v string) {
	if v = strings.TrimSpace(v); v == "" {
		return
	}
	if e.More == nil {
		e.More = map[string]string{}
	}
	e.More[name] = v
}

// setDate sets a date of Dates (a zero year leaves it out).
func (e *Entry) setDate(name string, d [3]int) {
	if d[0] == 0 {
		return
	}
	if e.Dates == nil {
		e.Dates = map[string][3]int{}
	}
	e.Dates[name] = d
}

// setNames sets a name variable of MoreNames.
func (e *Entry) setNames(name string, ns []Name) {
	if len(ns) == 0 {
		return
	}
	if e.MoreNames == nil {
		e.MoreNames = map[string][]Name{}
	}
	e.MoreNames[name] = ns
}
