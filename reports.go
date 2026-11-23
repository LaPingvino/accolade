package main

import (
	"bytes"
	"cmp"
	"fmt"
	"github.com/LaPingvino/lexington/pdf"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/lex"
	lexreport "github.com/LaPingvino/lexington/report"
)

// Production > Reports… (docs/PRODUCTION.md, Order step 2): the scene,
// location, character and cast reports, made from the script and stored
// nowhere. Lexington's report package makes them (pages and eighths as
// its PDF prints the script with the export's settings); here they are
// shown as a table to sort (stacked rows on a phone), saved as CSV, and
// printed as a web page (a PDF through Chromium, else the print dialog).

// reportKinds are the reports offered, with their names on screen.
var reportKinds = []struct{ kind, label string }{
	{lexreport.KindScenes, "Scenes"},
	{lexreport.KindLocations, "Locations"},
	{lexreport.KindCharacters, "Characters"},
	{lexreport.KindCast, "Cast"},
	{lexreport.KindSheets, "Breakdown sheets"},
	{lexreport.KindElements, "Elements"},
	{lexreport.KindOneLiner, "One-liner"},
	{lexreport.KindDOOD, "Day out of days"},
}

// scriptReport makes the reports of a script as the export would print
// it (job): its scenes numbered as a locked script numbers them, else
// counted on as Export's "Number the scenes" does, so each has a number;
// the characters' other names (the adaptation block's characters: line)
// merged into the cue's.
func scriptReport(content string, job exportJob) (r lexreport.Report, err error) {
	defer func() { // as exportAs: Lexington failing is an error, not the app's end
		if p := recover(); p != nil {
			err = fmt.Errorf("the reports failed: %v", p)
		}
	}()
	job.mergeOmitted = omittedRunsPrinted(content)
	s := job.script(content)
	if !sceneLockOf(content).locked {
		s = lex.NumberScenes(s, lex.NumberOptions{})
	}
	aliases := map[string]string{}
	l := learnedOf(content)
	for _, a := range l.Aliases {
		for _, n := range a.Names {
			aliases[n] = a.Cue
		}
	}
	scenes := job.scenes
	if scenes == nil {
		scenes = englishScenes
	}
	return cli.Report(s, job.options("pdf", ""), lexreport.Options{Scenes: scenes, Aliases: aliases})
}

// reportsDialog is Production > Reports….
type reportsDialog struct {
	w       *MainWindow
	pdfB    *widget.Button // PDF… (Print… for a table the browser prints)
	dialog  *dialog.CustomDialog
	rep     lexreport.Report
	bd      lexreport.Breakdown // the tags, for the breakdown's reports
	lost    int                 // the tags lost
	sched   lexreport.Schedule  // the shoot days, for the schedule's reports
	kind    *widget.Select
	sortBy  *widget.Select
	reverse *widget.Button
	desc    bool
	body    *fyne.Container // the table, or the stacked rows
	summary *widget.Label
	narrow  bool
	table   lexreport.Table // the report shown, sorted as shown
}

// showReports makes the reports (in the background: the script is laid
// out as printed) and shows them.
func (w *MainWindow) showReports() {
	if w.markdown {
		return
	}
	content, job := w.textEditor.Text(), w.layoutJob()
	f, bt, ps := w.breakdownNowFound()
	lost := 0
	for _, p := range ps {
		if p.lost {
			lost++
		}
	}
	w.setStatus("Making the reports...")
	type made struct {
		r   lexreport.Report
		err error
	}
	inBackground(func() made {
		r, err := scriptReport(content, job)
		return made{r, err}
	}, func(m made) {
		if m.err != nil {
			dialog.ShowError(fmt.Errorf("could not make the reports: %v", m.err), w.fyneWindow)
			return
		}
		w.setStatus("")
		rd := newReportsDialogWith(w, m.r, reportBreakdownFor(f, ps, bt, &m.r), lost)
		rd.sched = w.lexScheduleFor(&m.r)
		rd.kindChanged() // shown again with the days
		rd.dialog.Show()
		w.lastReports = rd
	})
}

// showReportsOf is Reports… showing a report first (the pane's
// Breakdown Sheets…).
func (w *MainWindow) showReportsOf(kind string) {
	if w.settings != nil {
		w.settings.SetString("report-kind", kind)
	}
	w.showReports()
}

// tableOf is a report as a table: the breakdown's made with the tags.
func (rd *reportsDialog) tableOf(kind string) (lexreport.Table, error) {
	switch kind {
	case lexreport.KindSheets:
		return rd.bd.SheetTable(rd.rep.Sheets(rd.bd)), nil
	case lexreport.KindElements:
		return rd.rep.ElementTable(rd.bd), nil
	case lexreport.KindOneLiner:
		return rd.rep.OneLinerTable(rd.bd, rd.sched, lexreport.StripOptions{Lang: rd.w.language}), nil
	case lexreport.KindDOOD:
		return rd.rep.DOODTable(rd.bd, rd.sched), nil
	}
	return rd.rep.Table(kind)
}

// shownOf are a table's columns shown: a breakdown's without the
// categories nothing is tagged in (the CSV has them all).
func shownOf(t lexreport.Table) []int {
	shown := t.Shown()
	if t.Kind != lexreport.KindSheets {
		return shown
	}
	return slices.DeleteFunc(shown, func(i int) bool {
		if i < 10 { // the scene's own columns
			return false
		}
		for _, row := range t.Rows {
			if row[i].String() != "" {
				return false
			}
		}
		return true
	})
}

func newReportsDialog(w *MainWindow, r lexreport.Report) *reportsDialog {
	return newReportsDialogWith(w, r, lexreport.Breakdown{}, 0)
}

func newReportsDialogWith(w *MainWindow, r lexreport.Report, bd lexreport.Breakdown, lost int) *reportsDialog {
	rd := &reportsDialog{w: w, rep: r, bd: bd, lost: lost}
	size := fitDialog(w.fyneWindow, 900+dialogMargin, 620+dialogMargin).SubtractWidthHeight(dialogMargin, dialogMargin)
	rd.narrow = size.Width < 600
	var labels []string
	for _, k := range reportKinds {
		labels = append(labels, k.label)
	}
	rd.summary = widget.NewLabel("")
	rd.summary.Wrapping = fyne.TextWrapWord
	rd.summary.Importance = widget.LowImportance
	rd.body = container.NewStack()
	rd.sortBy = widget.NewSelect(nil, func(string) { rd.show() })
	rd.reverse = widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
		rd.desc = !rd.desc
		rd.show()
	})
	fyne.SetAccessibleLabel(rd.reverse, "Reverse the order")
	rd.kind = widget.NewSelect(labels, func(string) { rd.kindChanged() })
	csvButton := widget.NewButtonWithIcon("CSV…", theme.DocumentSaveIcon(), rd.saveCSV)
	pdfButton := widget.NewButtonWithIcon("PDF…", theme.DocumentPrintIcon(), rd.savePDF)
	rd.pdfB = pdfButton
	closeButton := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() { rd.dialog.Hide() })
	closeButton.Importance = widget.HighImportance

	var top fyne.CanvasObject
	sortRow := labelledWith("Sort by:", rd.sortBy, rd.reverse)
	if rd.narrow {
		top = container.NewVBox(labelled("Report:", rd.kind), sortRow)
	} else {
		top = container.NewGridWithColumns(2, labelled("Report:", rd.kind), sortRow)
	}
	buttons := container.NewHBox(csvButton, pdfButton, layout.NewSpacer(), closeButton)
	content := container.NewBorder(container.NewVBox(top, rd.summary),
		container.NewVBox(widget.NewSeparator(), buttons), nil, nil, rd.body)
	sized := container.NewStack(content)
	rd.dialog = dialog.NewCustomWithoutButtons("Reports", sized, w.fyneWindow)
	rd.dialog.SetOnEscape(func() { rd.dialog.Hide() })
	rd.dialog.Resize(size.AddWidthHeight(dialogMargin, dialogMargin))
	kind := w.settings.GetString("report-kind")
	label := labels[0]
	for _, k := range reportKinds {
		if k.kind == kind {
			label = k.label
		}
	}
	rd.kind.SetSelected(label)
	return rd
}

// kindOf is the selected report's kind.
func (rd *reportsDialog) kindOf() string {
	for _, k := range reportKinds {
		if k.label == rd.kind.Selected {
			return k.kind
		}
	}
	return lexreport.KindScenes
}

// kindChanged offers the report's columns to sort by, and shows it in
// its own order.
func (rd *reportsDialog) kindChanged() {
	kind := rd.kindOf()
	if rd.pdfB != nil { // Lexington's PDF, shown in the page view; a table the browser prints
		switch label := "PDF…"; kind {
		case lexreport.KindSheets, lexreport.KindOneLiner, lexreport.KindDOOD:
			rd.pdfB.SetText(label)
		default:
			if pdfByPrinting() {
				label = "Print…"
			}
			rd.pdfB.SetText(label)
		}
	}
	if rd.w.settings.GetString("report-kind") != kind {
		rd.w.settings.SetString("report-kind", kind)
		_ = rd.w.settings.Save()
	}
	t, _ := rd.tableOf(kind)
	options := []string{ownOrder(kind)}
	for _, i := range shownOf(t) {
		options = append(options, t.Columns[i].Title)
	}
	rd.desc = false
	rd.sortBy.Options = options
	rd.sortBy.SetSelected(options[0]) // shows it
}

// ownOrder is a report's own order, the first way to sort it.
func ownOrder(kind string) string {
	switch kind {
	case lexreport.KindCast:
		return "Cast number"
	case lexreport.KindElements:
		return "Category order"
	case lexreport.KindOneLiner:
		return "Shooting order"
	case lexreport.KindDOOD:
		return "Cast number"
	}
	return "Script order"
}

// sortedTable is the report sorted by the column titled by (its own
// order for any other), reversed if desc.
func sortedTable(t lexreport.Table, by string, desc bool) lexreport.Table {
	col := -1
	for i, c := range t.Columns {
		if c.Title == by && !c.Data {
			col = i
		}
	}
	rows := slices.Clone(t.Rows)
	if col >= 0 {
		slices.SortStableFunc(rows, func(a, b []lexreport.Cell) int {
			return cmp.Or(cmp.Compare(a[col].Value, b[col].Value),
				strings.Compare(strings.ToLower(a[col].String()), strings.ToLower(b[col].String())))
		})
	}
	if desc {
		slices.Reverse(rows)
	}
	t.Rows = rows
	return t
}

// show shows the report as chosen and sorted.
func (rd *reportsDialog) show() {
	kind := rd.kindOf()
	t, err := rd.tableOf(kind)
	if err != nil {
		return
	}
	rd.table = sortedTable(t, rd.sortBy.Selected, rd.desc)
	if rd.desc {
		rd.reverse.SetIcon(theme.MoveUpIcon())
	} else {
		rd.reverse.SetIcon(theme.MoveDownIcon())
	}
	sum := reportSummary(rd.rep, kind, rd.w.layoutJob().page)
	if kind == lexreport.KindSheets || kind == lexreport.KindElements {
		sum = breakdownSummary(rd.bd, rd.lost) + " " + sum
	}
	if kind == lexreport.KindOneLiner || kind == lexreport.KindDOOD {
		sum = scheduleSummary(rd.rep, rd.sched, kind)
	}
	rd.summary.SetText(sum)
	if rd.narrow {
		rd.body.Objects = []fyne.CanvasObject{stackedRows(rd.table)}
	} else {
		rd.body.Objects = []fyne.CanvasObject{rd.tableView()}
	}
	rd.body.Refresh()
}

// reportSummary is the line above a report: its counts and how its pages
// are measured.
func reportSummary(r lexreport.Report, kind, page string) string {
	omitted := 0
	for _, s := range r.Scenes {
		if s.Omitted {
			omitted++
		}
	}
	scenes := plural(len(r.Scenes)-omitted, "scene")
	if omitted > 0 {
		scenes += fmt.Sprintf(" (and %d omitted)", omitted)
	}
	var parts []string
	switch kind {
	case lexreport.KindScenes:
		parts = append(parts, scenes, "in all "+r.Length().String()+" pages")
	case lexreport.KindLocations:
		parts = append(parts, plural(len(r.Locations), "location"), scenes)
	default:
		parts = append(parts, plural(len(r.Characters), "speaking character"), scenes)
	}
	paper := "the script format's paper"
	switch strings.ToLower(page) {
	case "letter":
		paper = "Letter"
	case "a4":
		paper = "A4"
	case "a5":
		paper = "A5"
	}
	return strings.Join(parts, " · ") + ". Pages and lengths (in eighths of a page) as Export prints the PDF, on " +
		paper + "; a character is in the scenes they speak in."
}

// tableView is the report as a table: a header to sort by (tapped again,
// reversed), each column as wide as its widest cell (within reason).
func (rd *reportsDialog) tableView() fyne.CanvasObject {
	t := rd.table
	shown := shownOf(t)
	tab := widget.NewTable(
		func() (int, int) { return len(t.Rows), len(shown) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, o fyne.CanvasObject) {
			l := o.(*widget.Label)
			c := t.Rows[id.Row][shown[id.Col]]
			// alignment and truncation first: setting the fields draws nothing,
			// SetText does — the other way round a reused cell kept the last
			// cell's look until something else redrew it (a click)
			l.Alignment = fyne.TextAlignLeading
			if t.Columns[shown[id.Col]].Numeric {
				l.Alignment = fyne.TextAlignTrailing
			}
			l.Truncation = fyne.TextTruncateEllipsis
			if l.Text == c.String() {
				l.Refresh() // same text: SetText would not redraw the new alignment
			}
			l.SetText(c.String())
		})
	tab.ShowHeaderRow = true
	tab.CreateHeader = func() fyne.CanvasObject {
		b := widget.NewButton("", nil)
		b.Importance = widget.LowImportance
		b.Alignment = widget.ButtonAlignLeading
		return b
	}
	tab.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		b := o.(*widget.Button)
		title := t.Columns[shown[id.Col]].Title
		b.SetText(title)
		b.SetIcon(nil)
		if rd.sortBy.Selected == title {
			b.SetIcon(theme.MoveDownIcon())
			if rd.desc {
				b.SetIcon(theme.MoveUpIcon())
			}
		}
		b.OnTapped = func() {
			if rd.sortBy.Selected == title {
				rd.desc = !rd.desc
				rd.show()
				return
			}
			rd.desc = false
			rd.sortBy.SetSelected(title)
		}
	}
	cellPad := theme.InnerPadding()*2 + theme.Padding()*2 // a label's, and the table's between cells
	for c, i := range shown {
		width := fyne.MeasureText(t.Columns[i].Title, theme.TextSize(), fyne.TextStyle{Bold: true}).Width + cellPad +
			theme.IconInlineSize() + theme.Padding()
		for _, row := range t.Rows {
			width = max(width, fyne.MeasureText(row[i].String(), theme.TextSize(), fyne.TextStyle{}).Width+cellPad)
		}
		tab.SetColumnWidth(c, min(width, 340))
	}
	return tab
}

// stackedRows is the report for a phone: a row a card, its first columns
// (the scene's number, place and time; the name) on its first line, the
// rest as "Title value" below.
func stackedRows(t lexreport.Table) fyne.CanvasObject {
	shown := shownOf(t)
	head := map[string]int{lexreport.KindScenes: 4, lexreport.KindLocations: 2, lexreport.KindCharacters: 1, lexreport.KindCast: 2,
		lexreport.KindSheets: 4, lexreport.KindElements: 2}[t.Kind]
	box := container.NewVBox()
	for _, row := range t.Rows {
		var first, rest []string
		for n, i := range shown {
			v := row[i].String()
			if v == "" {
				continue
			}
			if n < head {
				first = append(first, v)
			} else {
				rest = append(rest, t.Columns[i].Title+" "+v)
			}
		}
		title := widget.NewLabelWithStyle(strings.Join(first, " · "), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		title.Wrapping = fyne.TextWrapWord
		more := widget.NewLabel(strings.Join(rest, " · "))
		more.Wrapping = fyne.TextWrapWord
		box.Add(container.NewVBox(title, more))
		box.Add(widget.NewSeparator())
	}
	if len(t.Rows) == 0 {
		box.Add(widget.NewLabel("Nothing to report yet."))
	}
	return container.NewVScroll(box)
}

// reportFileName is a report's file name: the script's, then the report.
func (rd *reportsDialog) reportFileName(ext string) string {
	base := strings.TrimSuffix(rd.w.scriptName(), filepath.Ext(rd.w.scriptName()))
	return fmt.Sprintf("%s - %s.%s", base, rd.kind.Selected, ext)
}

// saveCSV saves the report shown, in the order shown, as CSV (its layout
// in docs/PRODUCTION.md): downloaded in the browser.
func (rd *reportsDialog) saveCSV() {
	var b bytes.Buffer
	if err := rd.table.WriteCSV(&b); err != nil {
		dialog.ShowError(err, rd.w.fyneWindow)
		return
	}
	rd.saveFile(rd.reportFileName("csv"), func(p string) error { return writeFileAtomic(p, b.Bytes()) })
}

// saveFile saves the report's file (saveNamedFile), and says so (a PDF
// says so itself, once made).
func (rd *reportsDialog) saveFile(name string, write func(p string) error) {
	rd.w.saveNamedFile(name, func(p string) error {
		if err := write(p); err != nil {
			return err
		}
		if !inBrowser && !strings.HasSuffix(p, ".pdf") {
			rd.w.setStatus("Report saved as " + filepath.Base(p))
		}
		return nil
	})
}

// saveNamedFile asks where to save name (the browser downloads it), and
// writes it there; write says what was saved.
func (w *MainWindow) saveNamedFile(name string, write func(p string) error) {
	if inBrowser {
		p := filepath.Join("/files", name)
		err := os.MkdirAll("/files", 0o755)
		if err == nil {
			err = write(p)
		}
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		offerDownload(p)
		return
	}
	d := dialog.NewFileSave(func(c fyne.URIWriteCloser, err error) {
		if err != nil || c == nil {
			return
		}
		p := c.URI().Path()
		c.Close()
		if err := write(p); err != nil {
			dialog.ShowError(fmt.Errorf("could not save %s: %v", filepath.Base(p), err), w.fyneWindow)
		}
	}, w.fyneWindow)
	d.SetFileName(name)
	if dir := cmp.Or(w.settings.GetString("export-directory"), defaultExportDir()); dir != "" {
		if l, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
			d.SetLocation(l)
		}
	}
	d.Show()
}

// reportPage is the report shown as a web page to print.
func (rd *reportsDialog) reportPage() ([]byte, error) {
	var b bytes.Buffer
	name := strings.TrimSuffix(rd.w.scriptName(), filepath.Ext(rd.w.scriptName()))
	err := lexreport.WriteHTML(&b, name+" – "+rd.kind.Selected, rd.summary.Text, rd.table)
	return b.Bytes(), err
}

// savePDF makes a PDF of the report shown: Chromium writes it, or the
// print dialog does (Save as PDF), as a Markdown document's export.
func (rd *reportsDialog) savePDF() {
	switch k := rd.kindOf(); k {
	case lexreport.KindSheets, lexreport.KindOneLiner, lexreport.KindDOOD: // Lexington's PDF: its pages shown
		rd.showReportPDF(k)
		return
	}
	page, err := rd.reportPage()
	if err != nil {
		dialog.ShowError(err, rd.w.fyneWindow)
		return
	}
	dir, err := os.MkdirTemp("", "accolade-report-")
	if err != nil {
		dialog.ShowError(err, rd.w.fyneWindow)
		return
	}
	htmlPath := filepath.Join(dir, "report.html")
	if err := os.WriteFile(htmlPath, page, 0o644); err != nil {
		dialog.ShowError(err, rd.w.fyneWindow)
		return
	}
	if pdfByPrinting() {
		rd.w.setStatus("Print: choose a printer, or Save as PDF")
		printFile(htmlPath)
		return
	}
	browser, w := localBrowser(), rd.w
	rd.saveFile(rd.reportFileName("pdf"), func(p string) error {
		w.setStatus("Making the PDF...")
		inBackground(func() error { // Chromium takes a moment: not on the window's thread
			defer os.RemoveAll(dir)
			return writeViaTemp(p, func(tmp string) error { return browserPDF(browser, htmlPath, tmp) })
		}, func(err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not make the PDF: %v", err), w.fyneWindow)
				return
			}
			w.setStatus("Report saved as " + filepath.Base(p))
		})
		return nil
	})
}

// showReportPDF shows the breakdown sheets (a page a scene in the
// classic layout), the one-liner or the Day Out of Days as Lexington
// writes their PDF, in the page view: Save PDF… (Download PDF in the
// browser) there; the dialog steps aside, Back to Reports brings it
// again.
func (rd *reportsDialog) showReportPDF(kind string) {
	w := rd.w
	var b bytes.Buffer
	paper := w.layoutJob().page
	var d pdf.Drawing
	var err error
	switch kind {
	case lexreport.KindSheets:
		name := strings.TrimSuffix(w.scriptName(), filepath.Ext(w.scriptName()))
		d, err = lexreport.DrawSheetsPDF(&b, name, rd.rep.Sheets(rd.bd), paper)
	case lexreport.KindOneLiner:
		d, err = lexreport.DrawOneLinerPDF(&b, w.scheduleTitle(), rd.rep, rd.bd, rd.sched, lexreport.StripOptions{Lang: w.language}, paper)
	default:
		d, err = lexreport.DrawDOODPDF(&b, w.scheduleTitle(), rd.rep, rd.bd, rd.sched, paper)
	}
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	dir, err := previewTemp()
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	name := rd.reportFileName("pdf")
	out := filepath.Join(dir, name)
	if err := os.WriteFile(out, b.Bytes(), 0o644); err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	rd.dialog.Hide()
	v := w.showPages(d, rd.kind.Selected, out, name, backButton("Back to Reports", func() {
		w.hidePages()
		rd.dialog.Show()
	}))
	pagesShown(v)
}

// breakdownSummary is the line about the tags above the breakdown's
// reports.
func breakdownSummary(bd lexreport.Breakdown, lost int) string {
	n := 0
	for _, es := range bd.Elements {
		n += len(es)
	}
	s := "Breakdown: " + plural(n, "tag") + " in " + plural(len(bd.Elements), "scene")
	if lost > 0 {
		s += ", " + plural(lost, "lost tag") + " left out (the Production pane lists them)"
	}
	return s + ". The cast is who speaks, and who is tagged as a cast member."
}

// scheduleSummary is the line above the schedule's reports.
func scheduleSummary(r lexreport.Report, s lexreport.Schedule, kind string) string {
	n := 0
	for _, d := range s.Days {
		n += len(d.Scenes)
	}
	if len(s.Days) == 0 {
		return "No shoot days yet: the stripboard (Production > Stripboard) puts scenes on days."
	}
	sum := fmt.Sprintf("%s, %s on them.", plural(len(s.Days), "shoot day"), plural(n, "scene"))
	if kind == lexreport.KindDOOD {
		sum += " SW start work, W work, WF work finish, SWF start-work-finish, H hold (Movie Magic Scheduling's codes)."
	}
	return sum
}
