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

// paginationChoices are the dialog's choices; the last one picks parts.
var paginationChoices = []struct{ label, value string }{
	{"Script format's", ""},
	{"All (as screenplay software)", "all"},
	{"None (as before)", "none"},
	{"Some:", "some"},
}

// paginationWidgets makes the page breaks' choice and its parts.
func (ed *ExportDialog) paginationWidgets() fyne.CanvasObject {
	var labels []string
	for _, c := range paginationChoices {
		labels = append(labels, c.label)
	}
	ed.paginationSelect = widget.NewSelect(labels, func(string) { ed.enablePagination() })
	ed.paginationKeep = widget.NewCheck("Headings, names and parentheticals with what follows", nil)
	ed.paginationWidows = widget.NewCheck("Two lines of a paragraph on each side of a break", nil)
	ed.paginationMore = widget.NewCheck("Speeches broken with (MORE) and (CONT'D)", nil)
	ed.setPagination("")
	return container.NewVBox(labelled("Page breaks:", ed.paginationSelect),
		container.NewPadded(container.NewVBox(ed.paginationKeep, ed.paginationWidows, ed.paginationMore)))
}

// setPagination shows a setting's value.
func (ed *ExportDialog) setPagination(value string) {
	choice := paginationChoices[0].label
	p, err := rules.ParsePagination(value)
	switch {
	case value == "" || err != nil:
	case p == rules.AllPagination:
		choice = paginationChoices[1].label
	case p == rules.Pagination{}:
		choice = paginationChoices[2].label
	default:
		choice = paginationChoices[3].label
	}
	if value == "" || err != nil {
		p = rules.AllPagination // the parts, should some be chosen
	}
	ed.paginationKeep.SetChecked(p.Keep)
	ed.paginationWidows.SetChecked(p.Widows)
	ed.paginationMore.SetChecked(p.More)
	ed.paginationSelect.SetSelected(choice)
}

// chosenPagination is the value chosen, for Lexington and the setting.
func (ed *ExportDialog) chosenPagination() string {
	for _, c := range paginationChoices {
		if c.label == ed.paginationSelect.Selected && c.value != "some" {
			return c.value
		}
	}
	if ed.paginationSelect.Selected != paginationChoices[3].label {
		return ""
	}
	return rules.Pagination{Keep: ed.paginationKeep.Checked, Widows: ed.paginationWidows.Checked, More: ed.paginationMore.Checked}.String()
}

// enablePagination lets the parts be chosen with Some, the choice with a
// format Lexington makes.
func (ed *ExportDialog) enablePagination() {
	if ed.paginationMore == nil {
		return
	}
	on := !ed.includeTitlePageCheck.Disabled() // (as the other script options)
	some := on && ed.paginationSelect.Selected == paginationChoices[3].label
	for _, o := range []fyne.Disableable{ed.paginationKeep, ed.paginationWidows, ed.paginationMore} {
		if some {
			o.Enable()
		} else {
			o.Disable()
		}
	}
	if on {
		ed.paginationSelect.Enable()
	} else {
		ed.paginationSelect.Disable()
	}
}
