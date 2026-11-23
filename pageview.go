package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/pdf"
)

// The page view: a PDF's pages as Lexington drew them (pdf.Drawing),
// painted by Accolade itself — no PDF viewer, no browser frame, the same
// on the desktop, in the browser and on a phone (Chrome on Android has no
// PDF viewer in a frame: the preview was blank there). White pages with
// a shadow down a column; fit to the width, 100%, or zoomed in and out
// (the buttons, the keys, two fingers in the browser); "Page n of N" as
// it scrolls. Save PDF… and Open in Viewer on the desktop, Download PDF
// in the browser, hand on the PDF itself.

// pageZooms are the zoom steps, in points a unit's fraction of 100%.
var pageZooms = []float32{0.25, 0.33, 0.5, 0.67, 0.8, 1, 1.25, 1.5, 2, 3, 4}

// pointUnits is a point in Fyne's units at 100%: a unit is about a pixel
// at 96 dots an inch.
const pointUnits = 96.0 / 72

// maxPagePixels is the widest a page is painted: zoomed further, the
// image is stretched (a phone's texture and memory have limits).
const maxPagePixels = 3000

// pageGap is the space round and between the pages, in units.
const pageGap = 12

// pagesShownHook is told each time a page view has shown a drawing, the
// last thing its caller does (tests wait on it). Atomic: a preview's
// goroutine calls it while a later test may set it (the test driver runs
// fyne.Do on the calling goroutine).
var pagesShownHook atomic.Pointer[func(*pageView)]

// pagesShown tells the hook, if one is set, that v shows a drawing.
func pagesShown(v *pageView) {
	if f := pagesShownHook.Load(); f != nil {
		(*f)(v)
	}
}

// pageView shows a drawing's pages with its bar.
type pageView struct {
	widget.BaseWidget
	w        *MainWindow
	d        *pdf.Drawing
	r        *pdf.Rasterizer
	pdfPath  string // the PDF the pages are of, "" if there is none
	name     string // its file's name, for Save PDF…
	zoom     float32
	fit      bool // the width fits the view
	scale    float32
	narrow   bool   // laid out for a phone
	pinching bool   // two fingers are zooming: the pages stretch, painted again when they lift
	remake   func() // makes the pages again after a change in Page Setup, and Refresh (nil: they stay)
	// follow keeps a script's pages following its text (pagefollow.go;
	// nil: none, or a Markdown document's, which mdPages follows)
	follow *pageFollow
	// stale: the text changed since the pages were made (a dot on
	// Refresh); dimFrom: the pages from there may be out of date, dimmed
	// (-1: none); pdfStale: the pages shown were made again since the PDF
	// (Save PDF… makes it again first: freshPDF)
	stale    bool
	dimFrom  int
	pdfStale bool
	freshPDF func(then func())
	// painted counts the pages painted (the tests', the measures')
	painted int
	// links are the drawing's (pageview_links.go); hoverLink the one
	// under the pointer, focusLink the keyboard's (-1: none); tip the
	// tip shown for one
	links     []drawnLink
	hoverLink int
	focusLink int
	marks     linkMarks
	tip       string

	title *widget.Label
	head  *fyne.Container // the title, the count, Close
	count *widget.Label
	note  *widget.Label
	extra *fyne.Container // the caller's: a page setup line, Back
	// setupLine is the page setup line in extra, if it is one (Preview
	// PDF's: the Preview tab's Setup has it in the Ribbon look)
	setupLine *pageSetupLine
	area      *pageArea
	scroll    *container.Scroll
	column    *fyne.Container
	pages     []*pageImage
	top       *fyne.Container // the bar, the caller's line, the note: over the pages
	tools     *fyne.Container // the bar's buttons (the Menus look's; the Ribbon look has the Preview tab)
	save      *widget.Button
	open      *widget.Button
	refreshB  *widget.Button
	followC   *widget.Check
	closeB    *widget.Button
	onClose   func()
	box       fyne.CanvasObject

	mu        sync.Mutex
	rendering map[int]paintJob // a page being painted: the width in pixels asked, of its version
}

// paintJob is a page being painted: its width, its version.
type paintJob struct{ px, ver int }

// pageImage is a page in the column: its shadow, its paper, the image
// painted of it (none while it is far out of sight).
type pageImage struct {
	shadow, paper, dim *canvas.Rectangle
	img                *canvas.Image
	px                 int // the image's width in pixels, 0 if none (or to paint again)
	ver                int // its drawing's version: a paint of an older one is dropped
}

func newPageImage() *pageImage {
	p := &pageImage{
		shadow: canvas.NewRectangle(color.NRGBA{0, 0, 0, 0x40}),
		paper:  canvas.NewRectangle(color.White),
		dim:    canvas.NewRectangle(color.NRGBA{0x80, 0x80, 0x80, 0x70}),
		img:    canvas.NewImageFromImage(nil),
	}
	p.dim.Hide()
	p.img.FillMode = canvas.ImageFillStretch
	p.img.ScaleMode = canvas.ImageScaleSmooth
	return p
}

func newPageView(w *MainWindow) *pageView {
	v := &pageView{w: w, fit: true, zoom: 1, scale: 1, rendering: map[int]paintJob{}, narrow: w != nil && w.narrow, dimFrom: -1,
		hoverLink: -1, focusLink: -1, marks: newLinkMarks()}
	v.ExtendBaseWidget(v)
	v.title = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.title.Truncation = fyne.TextTruncateEllipsis
	v.count = widget.NewLabel("")
	v.note = widget.NewLabel("")
	v.note.Wrapping = fyne.TextWrapWord
	v.note.Importance = widget.WarningImportance
	v.note.Hide()
	v.extra = container.NewVBox()
	v.column = container.New(&pagesLayout{v: v})
	v.scroll = container.NewScroll(v.column)
	v.scroll.OnScrolled = func(fyne.Position) { v.showInSight() }
	v.area = newPageArea(v)
	zoomOut := widget.NewButtonWithIcon("", theme.ZoomOutIcon(), func() { v.zoomBy(-1) })
	zoomOut.Importance = widget.LowImportance
	zoomIn := widget.NewButtonWithIcon("", theme.ZoomInIcon(), func() { v.zoomBy(1) })
	zoomIn.Importance = widget.LowImportance
	fit := widget.NewButtonWithIcon("Fit", theme.ViewFullScreenIcon(), func() { v.setFit() })
	fit.Importance = widget.LowImportance
	actual := widget.NewButton("100%", func() { v.setZoom(1) })
	actual.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(zoomOut, "Zoom out")
	fyne.SetAccessibleLabel(zoomIn, "Zoom in")
	fyne.SetAccessibleLabel(fit, "Fit the page's width")
	fyne.SetAccessibleLabel(actual, "Actual size")
	if inBrowser {
		v.save = widget.NewButtonWithIcon("Download PDF", theme.DownloadIcon(), v.download)
	} else {
		v.save = widget.NewButtonWithIcon("Save PDF…", theme.DocumentSaveIcon(), v.savePDF)
		v.open = widget.NewButtonWithIcon("Open in Viewer", theme.FileApplicationIcon(), v.openInViewer)
	}
	v.closeB = widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		if v.onClose != nil {
			v.onClose()
		}
	})
	v.refreshB = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refresh)
	v.refreshB.Importance = widget.LowImportance
	v.followC = widget.NewCheck("Follow Edits", func(on bool) {
		if w != nil && on != w.followEdits() {
			w.setFollowEdits(on)
		}
	})
	if w != nil {
		v.followC.Checked = w.followEdits()
	}
	files := container.NewHBox(v.save)
	if v.open != nil {
		files.Add(v.open)
	}
	if v.narrow { // a phone: the files' buttons as icons, beside the zoom's
		for _, o := range files.Objects {
			b := o.(*widget.Button)
			fyne.SetAccessibleLabel(b, b.Text)
			b.SetText("")
		}
	}
	right := container.NewHBox(v.count, v.closeB)
	v.tools = container.NewHBox(zoomOut, fit, actual, zoomIn, v.refreshB, v.followC, layout.NewSpacer(), files)
	if v.narrow { // a phone: Refresh an icon, Follow by the title
		fyne.SetAccessibleLabel(v.refreshB, v.refreshB.Text)
		fyne.SetAccessibleLabel(v.followC, v.followC.Text)
		v.refreshB.SetText("")
		v.followC.Text = "Follow"
		right = container.NewHBox(v.followC, v.count, v.closeB)
		v.tools = container.NewHBox(zoomOut, fit, actual, zoomIn, v.refreshB, layout.NewSpacer(), files)
	}
	v.head = container.NewBorder(nil, nil, nil, right, v.title)
	v.top = container.NewVBox(v.head, v.tools, v.extra, v.note)
	v.box = container.NewBorder(v.top, nil, nil, nil, v.area)
	fyne.SetAccessibleLabel(v.area, "Pages")
	return v
}

func (v *pageView) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(v.box)
}

// show shows a drawing: the pages of pdfPath (name its file's name), the
// place in them and the zoom kept if it is made again, the pages that
// look as before not painted again.
func (v *pageView) show(d pdf.Drawing, title, pdfPath, name string) {
	v.title.SetText(title)
	v.pdfPath, v.name = pdfPath, name
	v.pdfStale = false
	for _, b := range []*widget.Button{v.save, v.open} {
		if b != nil {
			b.Disable()
			if pdfPath != "" {
				b.Enable()
			}
		}
	}
	v.setStale(false)
	v.setDrawing(d, -1)
}

// setDrawing shows d's pages, those from dimFrom on dimmed (-1: none):
// the place in them (the page, and how far down it) and the zoom kept;
// a page that looks as it did keeps its painting, the others are painted
// again (shown as they were until then), only those in sight.
func (v *pageView) setDrawing(d pdf.Drawing, dimFrom int) {
	again := v.d != nil && len(d.Pages) > 0
	page, frac := v.anchor()
	old := v.d
	if v.r == nil || old == nil || !sameFonts(old.Fonts, d.Fonts) {
		v.r = pdf.NewRasterizer(&d)
		old = nil // painted with other fonts
	} else if !sameImages(old.Images, d.Images) {
		// the rasterizer paints its drawing's pictures: one that came (a
		// title page's, a storyboard frame's) or changed needs the new
		// drawing's, and the pages showing it painted again
		v.r = pdf.NewRasterizer(&d)
		old = nil
	}
	v.d = &d
	for i, p := range d.Pages {
		if i == len(v.pages) {
			v.pages = append(v.pages, newPageImage())
		}
		pi := v.pages[i]
		if old != nil && i < len(old.Pages) && old.Pages[i].Equal(p) {
			continue
		}
		pi.ver++
		pi.px = 0 // painted again when in sight
		if old == nil || i >= len(old.Pages) || old.Pages[i].Width != p.Width || old.Pages[i].Height != p.Height {
			pi.img.Image = nil
			pi.img.Refresh()
		}
	}
	v.pages = v.pages[:len(d.Pages)]
	v.column.Objects = v.column.Objects[:0]
	for _, p := range v.pages {
		v.column.Objects = append(v.column.Objects, p.shadow, p.paper, p.img, p.dim)
	}
	v.column.Objects = append(v.column.Objects, v.marks.lit, v.marks.ring)
	// the links as now drawn (a page that looks the same may link
	// elsewhere: a heading moved)
	v.links, v.focusLink = drawingLinks(v.d), -1
	v.hover(-1)
	v.mu.Lock()
	v.rendering = map[int]paintJob{}
	v.mu.Unlock()
	v.dim(dimFrom)
	v.relayout()
	if again {
		v.toAnchor(page, frac)
	} else {
		v.scroll.ScrollToTop()
	}
	v.showInSight()
}

// sameFonts reports whether two drawings' fonts are the same files.
// sameImages tells whether two drawings hold the same pictures (the
// same images, by name).
func sameImages(a, b map[string]image.Image) bool {
	if len(a) != len(b) {
		return false
	}
	for k, x := range a {
		if y, ok := b[k]; !ok || x != y {
			return false
		}
	}
	return true
}

func sameFonts(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, x := range a {
		y, ok := b[k]
		if !ok || len(x) != len(y) || len(x) > 0 && &x[0] != &y[0] {
			return false
		}
	}
	return true
}

// anchor is the page at the view's top and how far down it the top is
// (0 to 1).
func (v *pageView) anchor() (int, float32) {
	if v.d == nil || len(v.d.Pages) == 0 {
		return 0, 0
	}
	tops, _ := v.pageTops()
	y := v.scroll.Offset.Y
	for i := len(tops) - 1; i >= 0; i-- {
		if tops[i]-pageGap <= y {
			h := float32(v.d.Pages[i].Height)*v.scale + pageGap
			return i, (y - (tops[i] - pageGap)) / h
		}
	}
	return 0, 0
}

// toAnchor scrolls to an anchor: page i, frac down it.
func (v *pageView) toAnchor(i int, frac float32) {
	tops, _ := v.pageTops()
	if len(tops) == 0 {
		return
	}
	i = min(i, len(tops)-1)
	h := float32(v.d.Pages[i].Height)*v.scale + pageGap
	v.scroll.Offset.Y = max(0, min(tops[i]-pageGap+frac*h, v.column.MinSize().Height-v.scroll.Size().Height))
	v.scroll.Refresh()
}

// dim dims the pages from page from on (-1: none): they may be out of
// date until the pages are all made again.
func (v *pageView) dim(from int) {
	v.dimFrom = from
	for i, p := range v.pages {
		setVisible(p.dim, from >= 0 && i >= from)
	}
}

// setStale says whether the text changed since the pages were made (a
// dot on Refresh).
func (v *pageView) setStale(on bool) {
	if v.stale == on {
		return
	}
	v.stale = on
	v.refreshB.Importance = widget.LowImportance
	if on {
		v.refreshB.Importance = widget.HighImportance
	}
	v.refreshB.Refresh()
	if v.w != nil && v.w.ribbon != nil {
		v.w.ribbon.followPreview()
	}
}

// refresh makes the pages again: the place and zoom kept (Refresh, F5).
func (v *pageView) refresh() {
	switch {
	case v.follow != nil:
		v.follow.refresh()
	case v.remake != nil:
		v.remake()
	}
}

// setNote shows a line over the pages ("" none).
func (v *pageView) setNote(s string) {
	v.note.SetText(s)
	setVisible(v.note, s != "")
	v.fitTop()
}

// widest is the widest page's width, in points.
func (v *pageView) widest() float32 {
	w := float32(0)
	if v.d != nil {
		for _, p := range v.d.Pages {
			w = max(w, float32(p.Width))
		}
	}
	return w
}

// relayout sets the scale (fitting the view's width) and lays the column
// out again.
func (v *pageView) relayout() {
	if v.fit {
		if w := v.widest(); w > 0 && v.scroll.Size().Width > 0 {
			v.scale = max((v.scroll.Size().Width-2*pageGap)/w, 0.05)
		}
	} else {
		v.scale = v.zoom * pointUnits
	}
	v.column.Refresh()
	v.scroll.Refresh()
	if v.w != nil && v.w.ribbon != nil {
		v.w.ribbon.followPreview() // Fit checked, or not
	}
}

func (v *pageView) setFit() {
	v.fit = true
	v.relayout()
	v.showInSight()
}

// setZoom zooms to z (1 is 100%), keeping the middle of the view where it
// is on the page.
func (v *pageView) setZoom(z float32) {
	z = min(max(z, pageZooms[0]), pageZooms[len(pageZooms)-1])
	old := v.scale
	h := v.scroll.Size().Height
	mid := v.scroll.Offset.Y + h/2
	midX := v.scroll.Offset.X + v.scroll.Size().Width/2
	v.fit, v.zoom = false, z
	v.relayout()
	if old > 0 {
		r := v.scale / old
		v.scroll.Offset.Y = max(0, mid*r-h/2)
		v.scroll.Offset.X = max(0, midX*r-v.scroll.Size().Width/2)
		v.scroll.Refresh()
	}
	v.showInSight()
}

// zoomBy goes a step in or out from the zoom shown.
func (v *pageView) zoomBy(step int) {
	now := v.scale / pointUnits
	if step > 0 {
		for _, z := range pageZooms {
			if z > now*1.01 {
				v.setZoom(z)
				return
			}
		}
		v.setZoom(pageZooms[len(pageZooms)-1])
		return
	}
	for i := len(pageZooms) - 1; i >= 0; i-- {
		if pageZooms[i] < now*0.99 {
			v.setZoom(pageZooms[i])
			return
		}
	}
	v.setZoom(pageZooms[0])
}

// pinch zooms by a factor from the zoom at the pinch's start (two
// fingers in the browser, browser_js.go): the pages painted stretch, and
// are painted again at their size when the fingers lift (pinchEnd).
func (v *pageView) pinch(start, factor float32) {
	v.pinching = true
	v.setZoom(start * factor / pointUnits)
}

func (v *pageView) pinchEnd() {
	if v.pinching {
		v.pinching = false
		v.showInSight()
	}
}

// pageTops are the pages' tops in the column, and its height.
func (v *pageView) pageTops() ([]float32, float32) {
	tops := make([]float32, len(v.pages))
	y := float32(pageGap)
	if v.d == nil {
		return tops, 0
	}
	for i, p := range v.d.Pages {
		tops[i] = y
		y += float32(p.Height)*v.scale + pageGap
	}
	return tops, y
}

// showInSight paints the pages in sight (and a screen about them) at the
// size shown, lets those far away go, and says which page is in sight.
func (v *pageView) showInSight() {
	if v.d == nil || len(v.pages) == 0 {
		v.count.SetText("")
		return
	}
	tops, _ := v.pageTops()
	h := v.scroll.Size().Height
	if h <= 0 {
		h = 800
	}
	from, to := v.scroll.Offset.Y-h, v.scroll.Offset.Y+2*h
	scale := float32(1)
	if c := fyne.CurrentApp().Driver().CanvasForObject(v); c != nil {
		scale = c.Scale()
	}
	current := 0
	mid := v.scroll.Offset.Y + h/3
	for i, p := range v.d.Pages {
		ph := float32(p.Height) * v.scale
		if tops[i] <= mid {
			current = i
		}
		pi := v.pages[i]
		if tops[i]+ph < from || tops[i] > to {
			if pi.px != 0 {
				pi.img.Image, pi.px = nil, 0
				pi.img.Refresh()
			}
			continue
		}
		px := min(int(math.Round(float64(float32(p.Width)*v.scale*scale))), maxPagePixels)
		if px > 0 && pi.px != px && (!v.pinching || pi.px == 0) {
			v.paint(i, px)
		}
	}
	if c := fmt.Sprintf("Page %d of %d", current+1, len(v.d.Pages)); c != v.count.Text {
		v.count.SetText(c)
		v.head.Refresh() // its width
		if v.w != nil && v.w.ribbon != nil {
			v.w.ribbon.followPreview()
		}
	}
}

// paint paints page i px pixels wide, in the background.
func (v *pageView) paint(i, px int) {
	pi := v.pages[i]
	job := paintJob{px, pi.ver}
	v.mu.Lock()
	if v.rendering[i] == job {
		v.mu.Unlock()
		return
	}
	v.rendering[i] = job
	v.mu.Unlock()
	r, p := v.r, v.d.Pages[i]
	set := func(img image.Image) {
		v.mu.Lock()
		wanted := v.rendering[i] == job
		if wanted {
			delete(v.rendering, i)
		}
		v.mu.Unlock()
		if !wanted || i >= len(v.pages) || v.pages[i] != pi || pi.ver != job.ver {
			return
		}
		pi.img.Image, pi.px = img, px
		pi.img.Refresh()
		v.painted++
	}
	inBackground(func() image.Image { return r.Paint(p, float64(px)/p.Width) }, set)
}

// image is page i as painted, nil if it is not (tests look).
func (v *pageView) image(i int) image.Image {
	if i < len(v.pages) {
		return v.pages[i].img.Image
	}
	return nil
}

// withFreshPDF runs then once the PDF is of the pages shown (made again
// first, if they were made again since).
func (v *pageView) withFreshPDF(then func()) {
	if v.pdfStale && v.freshPDF != nil {
		v.freshPDF(then)
		return
	}
	then()
}

func (v *pageView) savePDF() {
	if v.pdfStale && v.freshPDF != nil {
		v.withFreshPDF(v.savePDF)
		return
	}
	if v.pdfPath == "" {
		return
	}
	v.w.saveNamedFile(v.name, func(p string) error {
		data, err := os.ReadFile(v.pdfPath)
		if err == nil {
			err = writeFileAtomic(p, data)
		}
		if err == nil {
			v.w.setStatus("PDF saved as " + filepath.Base(p))
		}
		return err
	})
}

func (v *pageView) download() {
	if v.pdfStale && v.freshPDF != nil {
		v.withFreshPDF(v.download)
		return
	}
	if v.pdfPath == "" {
		return
	}
	v.w.saveNamedFile(v.name, func(p string) error {
		if p == v.pdfPath {
			return nil
		}
		data, err := os.ReadFile(v.pdfPath)
		if err == nil {
			err = os.WriteFile(p, data, 0o644)
		}
		return err
	})
}

func (v *pageView) openInViewer() {
	if v.pdfStale && v.freshPDF != nil {
		v.withFreshPDF(v.openInViewer)
		return
	}
	if v.pdfPath != "" {
		openInSystem(v.pdfPath)
	}
}

// pagesLayout lays the pages down a column, centred, each with its
// shadow.
type pagesLayout struct{ v *pageView }

func (l *pagesLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	_, h := l.v.pageTops()
	w := float32(0)
	if !l.v.fit {
		w = l.v.widest()*l.v.scale + 2*pageGap
	}
	return fyne.NewSize(w, h)
}

func (l *pagesLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	v := l.v
	if v.d == nil {
		return
	}
	tops, _ := v.pageTops()
	for i, p := range v.d.Pages {
		if 4*i+3 >= len(objects) {
			return
		}
		pw, ph := float32(p.Width)*v.scale, float32(p.Height)*v.scale
		x := max((size.Width-pw)/2, pageGap)
		at := fyne.NewPos(x, tops[i])
		s := fyne.NewSize(pw, ph)
		objects[4*i].Move(at.AddXY(3, 3))
		objects[4*i].Resize(s)
		for k := 1; k < 4; k++ { // the paper, the painting, the dimming
			objects[4*i+k].Move(at)
			objects[4*i+k].Resize(s)
		}
	}
	v.markLinks()
}

// pageArea is the pages' scroller, which takes the focus: the keys move
// through the pages and zoom, and a frame shows it has the focus.
type pageArea struct {
	widget.BaseWidget
	v       *pageView
	frame   *canvas.Rectangle
	focused bool
	width   float32
	shift   bool // held: Tab goes back through the links
}

func newPageArea(v *pageView) *pageArea {
	a := &pageArea{v: v}
	a.frame = canvas.NewRectangle(color.Transparent)
	a.frame.StrokeWidth = 2
	a.ExtendBaseWidget(a)
	return a
}

func (a *pageArea) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	return &pageAreaRenderer{a: a, bg: bg, objects: []fyne.CanvasObject{bg, a.v.scroll, a.frame}}
}

type pageAreaRenderer struct {
	a       *pageArea
	bg      *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *pageAreaRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.a.v.scroll.Resize(size)
	r.a.frame.Resize(size)
	if size.Width != r.a.width { // the width to fit changed
		r.a.width = size.Width
		r.a.v.relayout()
		r.a.v.showInSight()
	}
}

func (r *pageAreaRenderer) MinSize() fyne.Size { return fyne.NewSize(120, 120) }

func (r *pageAreaRenderer) Refresh() {
	r.bg.FillColor = theme.Color(theme.ColorNameInputBackground)
	r.a.frame.StrokeColor = color.Transparent
	if r.a.focused {
		r.a.frame.StrokeColor = theme.Color(theme.ColorNameFocus)
	}
	r.bg.Refresh()
	r.a.frame.Refresh()
}

func (r *pageAreaRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *pageAreaRenderer) Destroy()                     {}

func (a *pageArea) FocusGained() { a.focused = true; a.Refresh(); a.v.markLinks() }
func (a *pageArea) FocusLost() {
	a.focused, a.shift = false, false
	a.v.focusLink = -1
	a.Refresh()
	a.v.markLinks()
}
func (a *pageArea) TypedRune(r rune) {
	switch r {
	case '+', '=':
		a.v.zoomBy(1)
	case '-':
		a.v.zoomBy(-1)
	case '0':
		a.v.setFit()
	case '1':
		a.v.setZoom(1)
	case ' ':
		a.scrollBy(a.v.scroll.Size().Height * 0.9)
	}
}

func (a *pageArea) TypedKey(e *fyne.KeyEvent) {
	v := a.v
	h := v.scroll.Size().Height
	switch e.Name {
	case fyne.KeyDown:
		a.scrollBy(48)
	case fyne.KeyUp:
		a.scrollBy(-48)
	case fyne.KeyPageDown:
		a.scrollBy(h * 0.9)
	case fyne.KeyPageUp:
		a.scrollBy(-h * 0.9)
	case fyne.KeyHome:
		v.scroll.ScrollToTop()
		v.showInSight()
	case fyne.KeyEnd:
		v.scroll.ScrollToBottom()
		v.showInSight()
	case fyne.KeyRight:
		a.toPage(1)
	case fyne.KeyLeft:
		a.toPage(-1)
	case fyne.KeyEscape:
		if v.onClose != nil {
			v.onClose()
		}
	case fyne.KeyTab:
		if i := v.nextLink(a.shift); i >= 0 {
			v.focusOn(i)
		}
	case fyne.KeyReturn, fyne.KeyEnter:
		v.followLink(v.focusLink)
	}
}

func (a *pageArea) scrollBy(dy float32) {
	s := a.v.scroll
	s.Offset.Y = min(max(0, s.Offset.Y+dy), max(0, a.v.column.MinSize().Height-s.Size().Height))
	s.Refresh()
	a.v.showInSight()
}

// toPage goes to the next page's top (or the previous one's).
func (a *pageArea) toPage(step int) {
	tops, _ := a.v.pageTops()
	y := a.v.scroll.Offset.Y + 1
	target := float32(0)
	if step > 0 {
		target = y
		for _, t := range tops {
			if t-pageGap > y {
				target = t - pageGap
				break
			}
		}
	} else {
		for _, t := range tops {
			if t-pageGap < y-2 {
				target = t - pageGap
			}
		}
	}
	a.scrollBy(target - a.v.scroll.Offset.Y)
}

// Tapped gives the pages the focus (the keys then move through them),
// and follows a link tapped.
func (a *pageArea) Tapped(e *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(a); c != nil {
		c.Focus(a)
	}
	if i := a.v.linkAt(e.Position); i >= 0 {
		a.v.focusLink = i
		a.v.followLink(i)
		a.v.markLinks()
	}
}
