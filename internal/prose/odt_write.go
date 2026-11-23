package prose

import (
	"archive/zip"
	"fmt"
	"io"
	"sort"
	"strings"
)

// WriteODT writes the document as an OpenDocument text (LibreOffice,
// Word too): LibreOffice's own styles (Text body, Heading 1–6,
// Quotations, Preformatted Text), real lists and footnotes.
func (d *Doc) WriteODT(out io.Writer) error {
	w := &odtWriter{textStyles: map[string]string{}, titlePage: len(d.TitlePage) > 0, pics: newEmbedder(d)}
	for i, line := range d.TitlePage {
		style := "P_title_line"
		switch {
		case i == 0:
			style = "P_title"
		case i == len(d.TitlePage)-1:
			style = "P_title_end"
		}
		if len(d.TitlePage) == 1 {
			style = "P_title_only"
		}
		b := &w.b
		b.WriteString(`<text:p text:style-name="` + style + `">` + xmlEsc(line) + `</text:p>`)
	}
	w.blocks(&w.b, d.Blocks, "")
	z := &zipWriter{z: zip.NewWriter(out)}
	z.part("mimetype", "application/vnd.oasis.opendocument.text", true)
	manifest := odtManifest
	for _, em := range w.pics.list {
		manifest = strings.Replace(manifest, `</manifest:manifest>`, fmt.Sprintf(`<manifest:file-entry manifest:full-path="Pictures/image%d.%s" manifest:media-type="image/%s"/></manifest:manifest>`, em.n, em.ext, em.ext), 1)
	}
	z.part("META-INF/manifest.xml", manifest, false)
	for _, em := range w.pics.list {
		z.file(fmt.Sprintf("Pictures/image%d.%s", em.n, em.ext), em.data)
	}
	z.part("meta.xml", xmlDecl+`<office:document-meta `+odtNS+` office:version="1.3"><office:meta><dc:title>`+xmlEsc(d.Title)+
		`</dc:title><meta:generator>Accolade</meta:generator></office:meta></office:document-meta>`, false)
	z.part("styles.xml", odtStyles(d.Paper), false)
	z.part("content.xml", xmlDecl+`<office:document-content `+odtNS+` office:version="1.3"><office:automatic-styles>`+
		w.automatic()+`</office:automatic-styles><office:body><office:text>`+w.b.String()+`</office:text></office:body></office:document-content>`, false)
	if z.err != nil {
		return z.err
	}
	return z.z.Close()
}

// odtStyles is the style sheet with paper's page size (A4's for "").
func odtStyles(paper string) string {
	if paper == "" {
		return odtStyleSheet
	}
	wd, ht := paperSize(paper)
	return strings.Replace(odtStyleSheet, `fo:page-width="8.27in" fo:page-height="11.69in"`,
		fmt.Sprintf(`fo:page-width="%.2fin" fo:page-height="%.2fin"`, wd, ht), 1)
}

const odtNS = `xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
	`xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" ` +
	`xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" ` +
	`xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" ` +
	`xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" ` +
	`xmlns:xlink="http://www.w3.org/1999/xlink" ` +
	`xmlns:dc="http://purl.org/dc/elements/1.1/" ` +
	`xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" ` +
	`xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0" ` +
	`xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0"`

type odtWriter struct {
	b          strings.Builder
	textStyles map[string]string // T_bi: its properties
	notes      int
	tables     int
	titlePage  bool      // the title page's styles are wanted
	pics       *embedder // the pictures
}

// blocks writes blocks; pstyle is the style of their paragraphs ("" Text
// body).
func (w *odtWriter) blocks(b *strings.Builder, bs []*Block, pstyle string) {
	for _, bl := range bs {
		w.block(b, bl, pstyle)
	}
}

func (w *odtWriter) block(b *strings.Builder, bl *Block, pstyle string) {
	switch bl.Kind {
	case Para:
		if img := bl.Picture(); img != nil {
			if em := w.pics.get(img); em != nil {
				pw, ph := w.pics.blockSize(em)
				b.WriteString(`<text:p text:style-name="P_picture">` + odtFrame(em, pw, ph, img.Alt) + `</text:p>`)
				if c := w.pics.doc.caption(img); c != "" {
					b.WriteString(`<text:p text:style-name="P_caption">` + odtText(c) + `</text:p>`)
				}
				return
			}
		}
		if bl.Hanging && pstyle == "" { // a bibliography's entry
			w.para(b, "P_hanging", bl.Spans)
			return
		}
		w.para(b, cmpOr(pstyle, "Text_20_body"), bl.Spans)
	case Heading:
		lv := min(max(bl.Level, 1), 6)
		fmt.Fprintf(b, `<text:h text:style-name="Heading_20_%d" text:outline-level="%d">`, lv, lv)
		w.spans(b, bl.Spans)
		b.WriteString(`</text:h>`)
	case Quote:
		w.blocks(b, bl.Children, "Quotations")
	case List:
		style := "L_bullet"
		if bl.Ordered {
			style = "L_number"
		}
		b.WriteString(`<text:list text:style-name="` + style + `">`)
		for i, it := range bl.Children {
			if i == 0 && bl.Ordered && bl.Start > 1 {
				fmt.Fprintf(b, `<text:list-item text:start-value="%d">`, bl.Start)
			} else {
				b.WriteString(`<text:list-item>`)
			}
			if len(it.Children) == 0 {
				b.WriteString(`<text:p text:style-name="List_20_Paragraph"/>`)
			}
			w.blocks(b, it.Children, cmpOr(pstyle, "List_20_Paragraph"))
			b.WriteString(`</text:list-item>`)
		}
		b.WriteString(`</text:list>`)
	case Code:
		for _, l := range strings.Split(strings.TrimRight(bl.Text, "\n"), "\n") {
			b.WriteString(`<text:p text:style-name="Preformatted_20_Text">` + odtText(l) + `</text:p>`)
		}
	case Rule:
		b.WriteString(`<text:p text:style-name="P_break">* * *</text:p>`)
	case Table:
		w.table(b, bl)
	}
}

func (w *odtWriter) para(b *strings.Builder, style string, spans []Span) {
	b.WriteString(`<text:p text:style-name="` + style + `">`)
	w.spans(b, spans)
	b.WriteString(`</text:p>`)
}

func (w *odtWriter) spans(b *strings.Builder, spans []Span) {
	for _, s := range spans {
		switch {
		case s.Break:
			b.WriteString(`<text:line-break/>`)
			continue
		case s.Note != nil:
			w.notes++
			fmt.Fprintf(b, `<text:note text:id="ftn%d" text:note-class="footnote"><text:note-citation>%d</text:note-citation><text:note-body>`, w.notes, w.notes)
			w.blocks(b, s.Note, "Footnote")
			b.WriteString(`</text:note-body></text:note>`)
			continue
		case s.Image != nil:
			if em := w.pics.get(s.Image); em != nil {
				pw, ph := inlineSize(em)
				b.WriteString(odtFrame(em, pw, ph, s.Image.Alt))
				continue
			}
			s.Text, s.Italic = altText(s.Image), true
		case s.Comment != "" || s.Text == "":
			continue
		}
		text := odtText(s.Text)
		if name := w.textStyle(s); name != "" {
			text = `<text:span text:style-name="` + name + `">` + text + `</text:span>`
		}
		if s.Link != "" {
			text = `<text:a xlink:type="simple" xlink:href="` + xmlAttr(s.Link) + `" text:style-name="Internet_20_link">` + text + `</text:a>`
		}
		b.WriteString(text)
	}
}

// textStyle is the automatic style for a span's format, made the first
// time it is needed ("" for plain text).
func (w *odtWriter) textStyle(s Span) string {
	name, props := "T", ""
	if s.Bold {
		name += "b"
		props += ` fo:font-weight="bold" style:font-weight-asian="bold" style:font-weight-complex="bold"`
	}
	if s.Italic {
		name += "i"
		props += ` fo:font-style="italic" style:font-style-asian="italic" style:font-style-complex="italic"`
	}
	if s.Strike {
		name += "s"
		props += ` style:text-line-through-style="solid" style:text-line-through-type="single"`
	}
	if s.Code {
		name += "c"
		props += ` style:font-name="Liberation Mono" fo:font-family="'Liberation Mono'" style:font-family-generic="modern" style:font-pitch="fixed"`
	}
	if props == "" {
		return ""
	}
	w.textStyles[name] = props
	return name
}

func (w *odtWriter) automatic() string {
	var names []string
	for n := range w.textStyles {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		b.WriteString(`<style:style style:name="` + n + `" style:family="text"><style:text-properties` + w.textStyles[n] + `/></style:style>`)
	}
	b.WriteString(`<style:style style:name="P_hanging" style:family="paragraph" style:parent-style-name="Text_20_body">` +
		`<style:paragraph-properties fo:margin-left="0.5in" fo:text-indent="-0.5in"/></style:style>`)
	b.WriteString(`<style:style style:name="P_break" style:family="paragraph" style:parent-style-name="Text_20_body">` +
		`<style:paragraph-properties fo:text-align="center"/></style:style>`)
	if len(w.pics.list) > 0 {
		b.WriteString(`<style:style style:name="P_picture" style:family="paragraph" style:parent-style-name="Text_20_body">` +
			`<style:paragraph-properties fo:text-align="center" fo:keep-with-next="always"/></style:style>` +
			`<style:style style:name="P_caption" style:family="paragraph" style:parent-style-name="Text_20_body">` +
			`<style:paragraph-properties fo:text-align="center"/><style:text-properties fo:font-style="italic" fo:font-size="10pt"/></style:style>` +
			`<style:style style:name="Fr_pic" style:family="graphic"><style:graphic-properties style:vertical-pos="middle" style:vertical-rel="text" fo:border="none"/></style:style>`)
	}
	if w.titlePage { // centred, the title a third down; the page breaks after the last line
		title := `<style:paragraph-properties fo:text-align="center" fo:margin-top="3in" fo:margin-bottom="0.33in"`
		b.WriteString(`<style:style style:name="P_title" style:family="paragraph" style:parent-style-name="Standard">` +
			title + `/><style:text-properties fo:font-size="20pt"/></style:style>` +
			`<style:style style:name="P_title_only" style:family="paragraph" style:parent-style-name="Standard">` +
			title + ` fo:break-after="page"/><style:text-properties fo:font-size="20pt"/></style:style>` +
			`<style:style style:name="P_title_line" style:family="paragraph" style:parent-style-name="Standard">` +
			`<style:paragraph-properties fo:text-align="center"/></style:style>` +
			`<style:style style:name="P_title_end" style:family="paragraph" style:parent-style-name="Standard">` +
			`<style:paragraph-properties fo:text-align="center" fo:break-after="page"/></style:style>`)
	}
	b.WriteString(`<style:style style:name="Tbl" style:family="table"><style:table-properties style:width="6.5in" table:align="margins"/></style:style>` +
		`<style:style style:name="Tbl_cell" style:family="table-cell"><style:table-cell-properties fo:padding="0.04in" fo:border="0.5pt solid #000000"/></style:style>`)
	return b.String()
}

func (w *odtWriter) table(b *strings.Builder, t *Block) {
	cols := 0
	for _, r := range t.Rows {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		return
	}
	w.tables++
	fmt.Fprintf(b, `<table:table table:name="Table%d" table:style-name="Tbl"><table:table-column table:number-columns-repeated="%d"/>`, w.tables, cols)
	for i, r := range t.Rows {
		if i == 0 {
			b.WriteString(`<table:table-header-rows>`)
		}
		b.WriteString(`<table:table-row>`)
		for j := range cols {
			b.WriteString(`<table:table-cell table:style-name="Tbl_cell" office:value-type="string">`)
			style := "Table_20_Contents"
			if i == 0 {
				style = "Table_20_Heading"
			}
			var spans []Span
			if j < len(r) {
				spans = r[j]
			}
			w.para(b, style, spans)
			b.WriteString(`</table:table-cell>`)
		}
		b.WriteString(`</table:table-row>`)
		if i == 0 {
			b.WriteString(`</table:table-header-rows>`)
		}
	}
	b.WriteString(`</table:table>`)
}

// odtText is text for ODT: spaces after the first as text:s, tabs as
// text:tab.
func odtText(s string) string {
	var b strings.Builder
	spaces := 0
	flush := func() {
		if spaces == 1 {
			b.WriteString(" ")
		} else if spaces > 1 {
			fmt.Fprintf(&b, ` <text:s text:c="%d"/>`, spaces-1)
		}
		spaces = 0
	}
	for _, r := range s {
		switch r {
		case ' ':
			spaces++
			continue
		case '\t':
			flush()
			b.WriteString(`<text:tab/>`)
			continue
		case '\n', '\r':
			flush()
			b.WriteString(" ")
			continue
		}
		flush()
		b.WriteString(xmlEsc(string(r)))
	}
	flush()
	return b.String()
}

const odtManifest = xmlDecl + `<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.3">` +
	`<manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.text" manifest:version="1.3"/>` +
	`<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>` +
	`<manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>` +
	`<manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>` +
	`</manifest:manifest>`

func odtHeading(n int, size string) string {
	return fmt.Sprintf(`<style:style style:name="Heading_20_%d" style:display-name="Heading %d" style:family="paragraph" style:parent-style-name="Heading" `+
		`style:next-style-name="Text_20_body" style:default-outline-level="%d" style:class="text">`+
		`<style:text-properties fo:font-size="%s" fo:font-weight="bold" style:font-weight-asian="bold" style:font-weight-complex="bold"/></style:style>`, n, n, n, size)
}

func odtListLevels(number bool) string {
	var b strings.Builder
	bullets := []string{"•", "◦", "▪"}
	formats := []string{"1", "a", "i"}
	for i := 1; i <= 10; i++ {
		props := fmt.Sprintf(`<style:list-level-properties text:list-level-position-and-space-mode="label-alignment">`+
			`<style:list-level-label-alignment text:label-followed-by="listtab" text:list-tab-stop-position="%.2fin" fo:text-indent="-0.25in" fo:margin-left="%.2fin"/>`+
			`</style:list-level-properties>`, 0.5*float64(i), 0.5*float64(i))
		if number {
			fmt.Fprintf(&b, `<text:list-level-style-number text:level="%d" style:num-suffix="." style:num-format="%s">%s</text:list-level-style-number>`,
				i, formats[(i-1)%3], props)
		} else {
			fmt.Fprintf(&b, `<text:list-level-style-bullet text:level="%d" text:bullet-char="%s">%s</text:list-level-style-bullet>`,
				i, bullets[(i-1)%3], props)
		}
	}
	return b.String()
}

var odtStyleSheet = xmlDecl + `<office:document-styles ` + odtNS + ` office:version="1.3">` +
	`<office:font-face-decls xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0">` +
	`<style:font-face style:name="Liberation Serif" svg:font-family="'Liberation Serif'" style:font-family-generic="roman" style:font-pitch="variable"/>` +
	`<style:font-face style:name="Liberation Sans" svg:font-family="'Liberation Sans'" style:font-family-generic="swiss" style:font-pitch="variable"/>` +
	`<style:font-face style:name="Liberation Mono" svg:font-family="'Liberation Mono'" style:font-family-generic="modern" style:font-pitch="fixed"/>` +
	`</office:font-face-decls><office:styles>` +
	`<style:default-style style:family="paragraph"><style:text-properties style:font-name="Liberation Serif" fo:font-size="12pt" fo:language="en" fo:country="GB"/></style:default-style>` +
	`<style:style style:name="Standard" style:family="paragraph" style:class="text"/>` +
	`<style:style style:name="Text_20_body" style:display-name="Text body" style:family="paragraph" style:parent-style-name="Standard" style:class="text">` +
	`<style:paragraph-properties fo:margin-top="0in" fo:margin-bottom="0.1in" fo:line-height="115%"/></style:style>` +
	`<style:style style:name="Heading" style:family="paragraph" style:parent-style-name="Standard" style:next-style-name="Text_20_body" style:class="text">` +
	`<style:paragraph-properties fo:margin-top="0.17in" fo:margin-bottom="0.08in" fo:keep-with-next="always"/>` +
	`<style:text-properties style:font-name="Liberation Sans" fo:font-size="14pt"/></style:style>` +
	odtHeading(1, "18pt") + odtHeading(2, "15pt") + odtHeading(3, "13pt") + odtHeading(4, "12pt") + odtHeading(5, "12pt") + odtHeading(6, "11pt") +
	`<style:style style:name="Quotations" style:family="paragraph" style:parent-style-name="Standard" style:class="html">` +
	`<style:paragraph-properties fo:margin-left="0.39in" fo:margin-right="0.39in" fo:margin-top="0in" fo:margin-bottom="0.1in"/></style:style>` +
	`<style:style style:name="List_20_Paragraph" style:display-name="List Paragraph" style:family="paragraph" style:parent-style-name="Text_20_body" style:class="list">` +
	`<style:paragraph-properties fo:margin-bottom="0.04in"/></style:style>` +
	`<style:style style:name="Preformatted_20_Text" style:display-name="Preformatted Text" style:family="paragraph" style:parent-style-name="Standard" style:class="html">` +
	`<style:paragraph-properties fo:margin-top="0in" fo:margin-bottom="0in"/><style:text-properties style:font-name="Liberation Mono" fo:font-size="10pt"/></style:style>` +
	`<style:style style:name="Footnote" style:family="paragraph" style:parent-style-name="Standard" style:class="extra">` +
	`<style:paragraph-properties fo:margin-left="0.24in" fo:text-indent="-0.24in"/><style:text-properties fo:font-size="10pt"/></style:style>` +
	`<style:style style:name="Table_20_Contents" style:display-name="Table Contents" style:family="paragraph" style:parent-style-name="Standard" style:class="extra"/>` +
	`<style:style style:name="Table_20_Heading" style:display-name="Table Heading" style:family="paragraph" style:parent-style-name="Table_20_Contents" style:class="extra">` +
	`<style:text-properties fo:font-weight="bold"/></style:style>` +
	`<style:style style:name="Internet_20_link" style:display-name="Internet link" style:family="text">` +
	`<style:text-properties fo:color="#000080" style:text-underline-style="solid" style:text-underline-width="auto" style:text-underline-color="font-color"/></style:style>` +
	`<text:list-style style:name="L_bullet" style:display-name="List Bullet">` + odtListLevels(false) + `</text:list-style>` +
	`<text:list-style style:name="L_number" style:display-name="List Number">` + odtListLevels(true) + `</text:list-style>` +
	`<text:notes-configuration text:note-class="footnote" style:num-format="1" text:start-value="0" text:footnotes-position="page" text:start-numbering-at="document"/>` +
	`</office:styles><office:automatic-styles><style:page-layout style:name="pm1">` +
	`<style:page-layout-properties fo:page-width="8.27in" fo:page-height="11.69in" fo:margin-top="1in" fo:margin-bottom="1in" fo:margin-left="1in" fo:margin-right="1in"/>` +
	`</style:page-layout></office:automatic-styles><office:master-styles><style:master-page style:name="Standard" style:page-layout-name="pm1"/></office:master-styles>` +
	`</office:document-styles>`
