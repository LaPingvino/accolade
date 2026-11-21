package prose

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
)

// WriteDOCX writes the document as a Word document: Word's own styles
// (Normal, Heading 1–6, Quote, List Paragraph), real lists and
// footnotes, so it is restyled in Word or LibreOffice as any other.
func (d *Doc) WriteDOCX(out io.Writer) error {
	w := &docxWriter{nextNum: 2}
	for i, line := range d.TitlePage {
		w.b.WriteString(`<w:p><w:pPr><w:jc w:val="center"/>`)
		if i == 0 { // the title, a third down the page
			w.b.WriteString(`<w:spacing w:before="4320" w:after="480"/>`)
		}
		w.b.WriteString(`</w:pPr>`)
		size := 24
		if i == 0 {
			size = 40
		}
		fmt.Fprintf(&w.b, `<w:r><w:rPr><w:sz w:val="%d"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, size, xmlEsc(line))
	}
	if len(d.TitlePage) > 0 {
		w.b.WriteString(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
	}
	w.blocks(d.Blocks, docxCtx{})
	z := &zipWriter{z: zip.NewWriter(out)}
	z.part("[Content_Types].xml", docxTypes, false)
	z.part("_rels/.rels", docxRootRels, false)
	z.part("docProps/core.xml", docxCoreProps(d.Title), false)
	z.part("word/_rels/document.xml.rels", w.rels(), false)
	z.part("word/styles.xml", docxStyleSheet, false)
	z.part("word/settings.xml", docxSettings, false)
	z.part("word/numbering.xml", w.numbering(), false)
	z.part("word/footnotes.xml", w.footnotes(), false)
	z.part("word/document.xml", xmlDecl+`<w:document `+wNS+`><w:body>`+w.b.String()+
		`<w:sectPr>`+docxPage(d.Paper)+`<w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`+
		`</w:body></w:document>`, false)
	if z.err != nil {
		return z.err
	}
	return z.z.Close()
}

// docxPage is the page size of paper, in twips (A4's for "").
func docxPage(paper string) string {
	if paper == "" {
		return `<w:pgSz w:w="11906" w:h="16838"/>`
	}
	twips := map[string][2]int{"letter": {12240, 15840}, "a4": {11906, 16838}, "a5": {8391, 11906}}
	t, ok := twips[strings.ToLower(paper)]
	if !ok {
		t = twips["a4"]
	}
	return fmt.Sprintf(`<w:pgSz w:w="%d" w:h="%d"/>`, t[0], t[1])
}

const xmlDecl = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"

const wNS = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"`

type docxWriter struct {
	b       strings.Builder
	links   []string // the hyperlinks' targets: rId100, rId101, ...
	notes   []string // the footnotes' paragraphs
	nums    []string // numbered lists: a <w:num> each, restarting
	nextNum int
	inNote  bool // in a footnote: its links would need rels of their own
}

// docxCtx is where a block is: in quotes, in list items.
type docxCtx struct {
	quote int
	list  []docxList // the lists it is in, the innermost last
	first bool       // the first paragraph of a list item: it has the number
	style string     // a style for its paragraphs (the footnotes')
}

type docxList struct{ num int }

func (w *docxWriter) blocks(bs []*Block, c docxCtx) {
	for i, b := range bs {
		cc := c
		cc.first = c.first && i == 0
		w.block(b, cc)
	}
}

func (w *docxWriter) block(b *Block, c docxCtx) {
	switch b.Kind {
	case Para:
		w.para(c.style, c, b.Spans)
	case Heading:
		w.para(fmt.Sprintf("Heading%d", min(max(b.Level, 1), 6)), c, b.Spans)
	case Quote:
		c.quote++
		w.blocks(b.Children, c)
	case List:
		num := 1 // the bullets'
		if b.Ordered {
			num = w.nextNum
			w.nextNum++
			start := max(b.Start, 1)
			w.nums = append(w.nums, fmt.Sprintf(`<w:num w:numId="%d"><w:abstractNumId w:val="2"/>`+
				`<w:lvlOverride w:ilvl="%d"><w:startOverride w:val="%d"/></w:lvlOverride></w:num>`, num, len(c.list), start))
		}
		for _, it := range b.Children {
			cc := c
			cc.list = append(append([]docxList(nil), c.list...), docxList{num})
			cc.first = true
			if len(it.Children) == 0 {
				w.para("", cc, nil)
			}
			w.blocks(it.Children, cc)
		}
	case Code:
		for _, l := range strings.Split(strings.TrimRight(b.Text, "\n"), "\n") {
			w.para("SourceCode", c, []Span{{Text: l, Code: true}})
		}
	case Rule:
		w.b.WriteString(`<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:t>* * *</w:t></w:r></w:p>`)
	case Table:
		w.table(b, c)
	}
}

// para writes a paragraph in a style ("" Normal) where c says.
func (w *docxWriter) para(style string, c docxCtx, spans []Span) {
	var pPr strings.Builder
	switch {
	case style != "":
		pPr.WriteString(`<w:pStyle w:val="` + style + `"/>`)
	case len(c.list) > 0:
		pPr.WriteString(`<w:pStyle w:val="ListParagraph"/>`)
	case c.quote > 0:
		pPr.WriteString(`<w:pStyle w:val="Quote"/>`)
	}
	depth := len(c.list)
	if depth > 0 && c.first {
		fmt.Fprintf(&pPr, `<w:numPr><w:ilvl w:val="%d"/><w:numId w:val="%d"/></w:numPr>`, min(depth-1, 8), c.list[depth-1].num)
	}
	if left := 720*c.quote + 720*depth; left > 0 && (c.quote > 0 || !c.first) {
		fmt.Fprintf(&pPr, `<w:ind w:left="%d"/>`, left)
	}
	w.b.WriteString(`<w:p>`)
	if pPr.Len() > 0 {
		w.b.WriteString(`<w:pPr>` + pPr.String() + `</w:pPr>`)
	}
	w.runs(&w.b, spans)
	w.b.WriteString(`</w:p>`)
}

func (w *docxWriter) runs(b *strings.Builder, spans []Span) {
	for _, s := range spans {
		switch {
		case s.Break:
			b.WriteString(`<w:r><w:br/></w:r>`)
			continue
		case s.Note != nil:
			id := len(w.notes) + 1
			w.notes = append(w.notes, "") // its place, before notes in it
			var nb docxWriter
			nb.nextNum, nb.inNote = w.nextNum, true
			nb.b.WriteString(`<w:footnote w:id="` + fmt.Sprint(id) + `">`)
			first := true
			for _, bl := range s.Note {
				if bl.Kind != Para || !first {
					nb.block(bl, docxCtx{style: "FootnoteText"})
					continue
				}
				nb.b.WriteString(`<w:p><w:pPr><w:pStyle w:val="FootnoteText"/></w:pPr>` +
					`<w:r><w:rPr><w:rStyle w:val="FootnoteReference"/></w:rPr><w:footnoteRef/></w:r><w:r><w:t xml:space="preserve"> </w:t></w:r>`)
				nb.runs(&nb.b, bl.Spans)
				nb.b.WriteString(`</w:p>`)
				first = false
			}
			nb.b.WriteString(`</w:footnote>`)
			w.notes[id-1] = nb.b.String()
			w.notes = append(w.notes, nb.notes...)
			w.nextNum, w.nums = nb.nextNum, append(w.nums, nb.nums...)
			fmt.Fprintf(b, `<w:r><w:rPr><w:rStyle w:val="FootnoteReference"/></w:rPr><w:footnoteReference w:id="%d"/></w:r>`, id)
			continue
		case s.Comment != "" || s.Text == "":
			continue
		}
		if w.inNote {
			s.Link = ""
		}
		var rPr strings.Builder
		if s.Link != "" {
			rPr.WriteString(`<w:rStyle w:val="Hyperlink"/>`)
		}
		if s.Code {
			rPr.WriteString(`<w:rFonts w:ascii="Courier New" w:hAnsi="Courier New" w:cs="Courier New"/>`)
		}
		if s.Bold {
			rPr.WriteString(`<w:b/>`)
		}
		if s.Italic {
			rPr.WriteString(`<w:i/>`)
		}
		if s.Strike {
			rPr.WriteString(`<w:strike/>`)
		}
		run := `<w:r>`
		if rPr.Len() > 0 {
			run += `<w:rPr>` + rPr.String() + `</w:rPr>`
		}
		run += `<w:t xml:space="preserve">` + xmlEsc(strings.ReplaceAll(s.Text, "\n", " ")) + `</w:t></w:r>`
		if s.Link != "" {
			w.links = append(w.links, s.Link)
			run = fmt.Sprintf(`<w:hyperlink r:id="rId%d">%s</w:hyperlink>`, 99+len(w.links), run)
		}
		b.WriteString(run)
	}
}

func (w *docxWriter) table(t *Block, c docxCtx) {
	cols := 0
	for _, r := range t.Rows {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		return
	}
	w.b.WriteString(`<w:tbl><w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="5000" w:type="pct"/>` +
		`<w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/>` +
		`<w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders></w:tblPr><w:tblGrid>`)
	for range cols {
		fmt.Fprintf(&w.b, `<w:gridCol w:w="%d"/>`, 9000/cols)
	}
	w.b.WriteString(`</w:tblGrid>`)
	for i, r := range t.Rows {
		w.b.WriteString(`<w:tr>`)
		for j := range cols {
			var spans []Span
			if j < len(r) {
				spans = r[j]
			}
			if i == 0 { // the header, in bold
				spans = append([]Span(nil), spans...)
				for k := range spans {
					spans[k].Bold = true
				}
			}
			w.b.WriteString(`<w:tc><w:p>`)
			w.runs(&w.b, spans)
			w.b.WriteString(`</w:p></w:tc>`)
		}
		w.b.WriteString(`</w:tr>`)
	}
	w.b.WriteString(`</w:tbl><w:p/>`)
}

func (w *docxWriter) rels() string {
	var b strings.Builder
	b.WriteString(xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/settings" Target="settings.xml"/>` +
		`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>` +
		`<Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footnotes" Target="footnotes.xml"/>`)
	for i, l := range w.links {
		fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="%s" TargetMode="External"/>`,
			100+i, xmlAttr(l))
	}
	b.WriteString(`</Relationships>`)
	return b.String()
}

func xmlAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func (w *docxWriter) numbering() string {
	level := func(i int, format, text string) string {
		font := ""
		if format == "bullet" {
			font = `<w:rPr><w:rFonts w:ascii="Symbol" w:hAnsi="Symbol" w:hint="default"/></w:rPr>`
			if i%3 != 0 {
				font = ""
			}
		}
		return fmt.Sprintf(`<w:lvl w:ilvl="%d"><w:start w:val="1"/><w:numFmt w:val="%s"/><w:lvlText w:val="%s"/>`+
			`<w:lvlJc w:val="left"/><w:pPr><w:ind w:left="%d" w:hanging="360"/></w:pPr>%s</w:lvl>`, i, format, text, 720*(i+1), font)
	}
	var b strings.Builder
	b.WriteString(xmlDecl + `<w:numbering ` + wNS + `><w:abstractNum w:abstractNumId="1"><w:multiLevelType w:val="hybridMultilevel"/>`)
	bullets := []string{"", "o", "▪"}
	for i := range 9 {
		b.WriteString(level(i, "bullet", bullets[i%3]))
	}
	b.WriteString(`</w:abstractNum><w:abstractNum w:abstractNumId="2"><w:multiLevelType w:val="hybridMultilevel"/>`)
	formats := []string{"decimal", "lowerLetter", "lowerRoman"}
	for i := range 9 {
		b.WriteString(level(i, formats[i%3], fmt.Sprintf("%%%d.", i+1)))
	}
	b.WriteString(`</w:abstractNum><w:num w:numId="1"><w:abstractNumId w:val="1"/></w:num>`)
	for _, n := range w.nums {
		b.WriteString(n)
	}
	b.WriteString(`</w:numbering>`)
	return b.String()
}

func (w *docxWriter) footnotes() string {
	return xmlDecl + `<w:footnotes ` + wNS + `>` +
		`<w:footnote w:type="separator" w:id="-1"><w:p><w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/></w:pPr><w:r><w:separator/></w:r></w:p></w:footnote>` +
		`<w:footnote w:type="continuationSeparator" w:id="0"><w:p><w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/></w:pPr><w:r><w:continuationSeparator/></w:r></w:p></w:footnote>` +
		strings.Join(w.notes, "") + `</w:footnotes>`
}

func docxCoreProps(title string) string {
	return xmlDecl + `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" ` +
		`xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>` + xmlEsc(title) + `</dc:title><dc:creator>Accolade</dc:creator></cp:coreProperties>`
}

const docxTypes = xmlDecl + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
	`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
	`<Default Extension="xml" ContentType="application/xml"/>` +
	`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
	`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
	`<Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/>` +
	`<Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>` +
	`<Override PartName="/word/footnotes.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footnotes+xml"/>` +
	`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>` +
	`</Types>`

const docxRootRels = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
	`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>` +
	`</Relationships>`

const docxSettings = xmlDecl + `<w:settings ` + wNS + `><w:footnotePr><w:footnote w:id="-1"/><w:footnote w:id="0"/></w:footnotePr></w:settings>`

func docxHeadingStyle(n int, size int) string {
	return fmt.Sprintf(`<w:style w:type="paragraph" w:styleId="Heading%d"><w:name w:val="heading %d"/><w:basedOn w:val="Normal"/>`+
		`<w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:keepLines/><w:spacing w:before="%d" w:after="120"/><w:outlineLvl w:val="%d"/></w:pPr>`+
		`<w:rPr><w:rFonts w:ascii="Liberation Sans" w:hAnsi="Liberation Sans" w:cs="Arial"/><w:b/><w:sz w:val="%d"/></w:rPr></w:style>`,
		n, n, 360-40*n, n-1, size)
}

var docxStyleSheet = xmlDecl + `<w:styles ` + wNS + `>` +
	`<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Liberation Serif" w:hAnsi="Liberation Serif" w:eastAsia="Times New Roman" w:cs="Times New Roman"/>` +
	`<w:sz w:val="24"/><w:szCs w:val="24"/><w:lang w:val="en-GB"/></w:rPr></w:rPrDefault>` +
	`<w:pPrDefault><w:pPr><w:spacing w:after="160" w:line="276" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>` +
	`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>` +
	docxHeadingStyle(1, 36) + docxHeadingStyle(2, 30) + docxHeadingStyle(3, 26) + docxHeadingStyle(4, 24) + docxHeadingStyle(5, 24) + docxHeadingStyle(6, 22) +
	`<w:style w:type="paragraph" w:styleId="Quote"><w:name w:val="Quote"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>` +
	`<w:pPr><w:ind w:left="720" w:right="720"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="ListParagraph"><w:name w:val="List Paragraph"/><w:basedOn w:val="Normal"/><w:qFormat/>` +
	`<w:pPr><w:spacing w:after="60"/><w:ind w:left="720"/><w:contextualSpacing/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="SourceCode"><w:name w:val="Source Code"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/></w:pPr><w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New" w:cs="Courier New"/><w:sz w:val="20"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="FootnoteText"><w:name w:val="footnote text"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/></w:pPr><w:rPr><w:sz w:val="20"/></w:rPr></w:style>` +
	`<w:style w:type="character" w:styleId="FootnoteReference"><w:name w:val="footnote reference"/><w:rPr><w:vertAlign w:val="superscript"/></w:rPr></w:style>` +
	`<w:style w:type="character" w:styleId="Hyperlink"><w:name w:val="Hyperlink"/><w:rPr><w:color w:val="0563C1"/><w:u w:val="single"/></w:rPr></w:style>` +
	`<w:style w:type="table" w:styleId="TableGrid"><w:name w:val="Table Grid"/></w:style>` +
	`</w:styles>`
