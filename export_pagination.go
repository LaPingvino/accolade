package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// Page breaks as screenplay software makes them (Lexington's pagination):
// the script format's own way, all of it, none, or some of its parts.
// The "export-pagination" setting is Lexington's value: "" (the format's),
// "all", "none" or parts joined with "+" ("keep+more").

// paginationChoices are Page Setup's choices; the last one picks parts.
var paginationChoices = []struct{ label, value string }{
	{"Script format's", ""},
	{"All (as screenplay software)", "all"},
	{"None (as before)", "none"},
	{"Some:", "some"},
}

// paginationWidgets makes the page breaks' choice and its parts.
func (p *pageSetupDlg) paginationWidgets() fyne.CanvasObject {
	var labels []string
	for _, c := range paginationChoices {
		labels = append(labels, c.label)
	}
	p.paginationSelect = widget.NewSelect(labels, func(string) { p.enable() })
	p.paginationKeep = widget.NewCheck("Headings and names with what follows", nil)
	p.paginationWidows = widget.NewCheck("Two lines each side of a break", nil)
	p.paginationMore = widget.NewCheck("Speeches split with (MORE), (CONT'D)", nil)
	p.setPagination("")
	return container.NewVBox(labelled("Page breaks:", p.paginationSelect),
		container.NewPadded(container.NewVBox(p.paginationKeep, p.paginationWidows, p.paginationMore)))
}

// setPagination shows a setting's value.
func (p *pageSetupDlg) setPagination(value string) {
	choice := paginationChoices[0].label
	pg, err := rules.ParsePagination(value)
	switch {
	case value == "" || err != nil:
	case pg == rules.AllPagination:
		choice = paginationChoices[1].label
	case pg == rules.Pagination{}:
		choice = paginationChoices[2].label
	default:
		choice = paginationChoices[3].label
	}
	if value == "" || err != nil {
		pg = rules.AllPagination // the parts, should some be chosen
	}
	p.paginationKeep.SetChecked(pg.Keep)
	p.paginationWidows.SetChecked(pg.Widows)
	p.paginationMore.SetChecked(pg.More)
	p.paginationSelect.SetSelected(choice)
}

// chosenPagination is the value chosen, for Lexington and the setting.
func (p *pageSetupDlg) chosenPagination() string {
	for _, c := range paginationChoices {
		if c.label == p.paginationSelect.Selected && c.value != "some" {
			return c.value
		}
	}
	if p.paginationSelect.Selected != paginationChoices[3].label {
		return ""
	}
	return rules.Pagination{Keep: p.paginationKeep.Checked, Widows: p.paginationWidows.Checked, More: p.paginationMore.Checked}.String()
}

// paginationLabel is the choice a setting's value shows.
func paginationLabel(value string) string {
	pg, err := rules.ParsePagination(value)
	switch {
	case value == "" || err != nil:
		return paginationChoices[0].label
	case pg == rules.AllPagination:
		return paginationChoices[1].label
	case pg == rules.Pagination{}:
		return paginationChoices[2].label
	}
	return paginationChoices[3].label + " " + value
}
