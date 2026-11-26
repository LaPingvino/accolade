package main

import "fmt"

// Real-film ranges (LOOP.md, Beats (B)): where films put a structure's
// key beats, from published measurements only, each with its source; a
// beat no study measures keeps its conventional range and says so. The
// numbers are the papers' own (or computed from their tables, as noted);
// none is Accolade's (the rule that makes a range of them is: two σ,
// or the lowest to the highest film, and a page of grace: off()). They are % of the running time, Accolade's are %
// of the pages: about the same thing (a page runs about a minute, a
// little under; Follows, "Does one page of a screenplay really equal one
// minute of screen time?", stephenfollows.com, 2026: 2,520 films, about 55 seconds a page). And the
// studies time the event, Accolade the section's start: a section
// usually starts with its event. docs/help/structure.md says the same.

// filmSource is a published measurement of where films put their beats.
type filmSource struct {
	short string // as the ranges' notes say it: "Thompson 1999, 73 films"
	cite  string // the full reference, as Help gives it
}

var (
	// Kristin Thompson's four large parts in 73 four-act films, as
	// Cutting tabulates them: setup 24.9% of the running time (σ 2.8),
	// complication 26.5% (σ 3.7), development 25.1% (σ 3.5), climax 23.5%
	// (σ 3.8); epilogs, in 83% of them, 1.8% (σ 1.6) within the climax.
	thompsonFilms = filmSource{"Thompson 1999, 73 films",
		"Kristin Thompson, Storytelling in the New Hollywood (Harvard University Press, 1999): 73 four-act films; " +
			"her figures as tabulated in James E. Cutting, \"Narrative theory and the dynamics of popular movies\", " +
			"Psychonomic Bulletin & Review 23(6), 1713–1743 (2016), doi:10.3758/s13423-016-1051-4"}
	// Eoin O'Brien's seven three-act plot points timed in the ten
	// highest-rated films on IMDb (September 2025), one film per row in
	// his Table 3; the medians and ranges are computed from that table
	// (its Table 1's means and SDs do not all agree with it).
	obrienFilms = filmSource{"O'Brien 2026, 10 films",
		"Eoin O'Brien, \"Narrative Structure in Highly Rated Hollywood Cinema: An Empirical Study of Three-Act Plot Points\", " +
			"Proceedings of the RSU International Research Conference 2026 (Rangsit University), 160–168: " +
			"the ten highest-rated films on IMDb, each plot point timed in the release cut; median and range computed from its Table 3"}
)

// filmSources are the sources, as Help lists them.
var filmSources = []filmSource{thompsonFilms, obrienFilms}

// filmMeasure is where films put one beat, and the range Accolade
// checks it against: two σ either side of the mean (about 19 films in
// 20) where the study gives a σ, the lowest to the highest film where
// it gives every film.
type filmMeasure struct {
	as     string // the beat as the study names it: "the end of the setup"
	typ    string // where films put it: "25%, σ 2.8", "median 50%, 49–53%"
	lo, hi float64
	src    filmSource
}

// sigmaMeasure is a beat at mean ± sd in % (the range two σ either side).
func sigmaMeasure(as string, mean, sd float64, src filmSource) *filmMeasure {
	return &filmMeasure{as: as, typ: fmt.Sprintf("%g%%, σ %g", mean, sd),
		lo: max(0, mean-2*sd), hi: min(100, mean+2*sd), src: src}
}

// spanMeasure is a beat with a median and the lowest and highest films.
func spanMeasure(as string, median, lo, hi float64, src filmSource) *filmMeasure {
	return &filmMeasure{as: as, typ: fmt.Sprintf("median %g%%, %g–%g%%", median, lo, hi), lo: lo, hi: hi, src: src}
}

// The measured beats. A boundary whose part runs from the start or to the
// end has that part's σ (the setup's end, the climax's start); the
// midpoint, between two parts, has none in Thompson's figures (24.9 +
// 26.5 = 51.4%), so O'Brien's is used.
var (
	filmActOneEnd = sigmaMeasure("the end of the setup (the act-one break)", 24.9, 2.8, thompsonFilms)
	filmActTwoEnd = sigmaMeasure("the start of the climax (the act-two break)", 100-23.5, 3.8, thompsonFilms)
	filmEpilog    = sigmaMeasure("the epilog (in 83% of the films)", 100-1.8, 1.6, thompsonFilms)
	filmInciting  = spanMeasure("the inciting incident", 10.5, 4, 17, obrienFilms)
	filmMidpoint  = spanMeasure("the midpoint", 50, 49, 53, obrienFilms)
	filmLowPoint  = spanMeasure("the low point (crisis)", 69, 66, 74, obrienFilms)
	filmClimax    = spanMeasure("the climax", 91.5, 82, 94, obrienFilms)
)

// beatRange is where a structure's section usually starts, in % of the
// pages; film, if not nil, is the measurement it is, else it is the
// convention the method and the manuals give.
type beatRange struct {
	lo, hi float64
	film   *filmMeasure
}

// conv is a conventional range.
func conv(lo, hi float64) beatRange { return beatRange{lo: lo, hi: hi} }

// films is a measured range.
func films(m *filmMeasure) beatRange { return beatRange{lo: m.lo, hi: m.hi, film: m} }

// kind is the range's in a word: measured or convention.
func (r beatRange) kind() string {
	if r.film != nil {
		return "measured"
	}
	return "convention"
}

// source says where a range is from: the measurement and its source, or
// that it is the convention.
func (r beatRange) source() string {
	if r.film == nil {
		return "the convention, not measured on films"
	}
	return fmt.Sprintf("films: %s, as %s; %s", r.film.typ, r.film.as, r.film.src.short)
}

// measuredCount is how many of t's ranges are measured, of how many.
func (t structureTemplate) measuredCount() (measured, all int) {
	for _, r := range t.ranges {
		if r.film != nil {
			measured++
		}
	}
	return measured, len(t.ranges)
}

// templateNamed is the structure called name.
func templateNamed(name string) (structureTemplate, bool) {
	for _, t := range structureTemplates {
		if t.name == name {
			return t, true
		}
	}
	return structureTemplate{}, false
}

// rangesNote says how a structure's ranges are made: "3 measured on
// films, 5 convention".
func rangesNote(structure string) string {
	t, ok := templateNamed(structure)
	if !ok {
		return ""
	}
	m, n := t.measuredCount()
	switch m {
	case 0:
		return "all convention, none measured on films"
	case n:
		return "all measured on films (Help > Structure)"
	}
	return fmt.Sprintf("%d measured on films (Help > Structure), %d convention", m, n-m)
}
