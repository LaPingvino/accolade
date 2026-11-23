# Markdown Syntax

For Markdown documents (File > New Markdown Document, or any .md file) and for a story used as a script's source.

## Blocks

- **Heading** — `# Chapter One`, `## Part`, `### Scene`
- **Paragraph** — lines of text, a blank line between paragraphs
- **Line break** — two spaces at the end of a line
- **Quote** — `> She said nothing.`
- **List** — `- milk` or `* milk`
- **Numbered list** — `1. Wake up`
- **Nested item** — indent two spaces: `  - inner`
- **Break** — `* * *` or `---` on a line of its own
- **Code** — three backticks above and below the lines
- **Table** — `| Name | Role |`, then `|---|---|`, then rows

## Inline

- **Italic** — `*word*` or `_word_` (Ctrl+I)
- **Bold** — `**word**` (Ctrl+B)
- **Strikethrough** — `~~word~~`
- **Code** — `` `word` ``
- **Link** — `[text](https://example.com)`
- **Footnote** — `word[^1]`, and below: `[^1]: The note.`

## The Format menu

Each writes plain Markdown on the selection or the cursor's lines; again takes it away where that makes sense; one undo step each.

- **Heading 1** … **Heading 6** — Ctrl+1 … Ctrl+6; **Normal Text** — Ctrl+0
- **Bold** — Ctrl+B; **Italic** — Ctrl+I; **Strikethrough**; **Inline Code**
- **Link…** — Ctrl+L: asks the address, the selection is its text; **Image…** — the same, `![…](…)`, or **Choose a Picture…**: copied into the bundle's `assets/` (or a `Name.assets/` folder beside the document) and linked from there
- **Footnote** — the next `[^n]` at the cursor, its note at the end
- **Go to Note or Reference** — from a `[^n]` to its note, from a note back to its mark; the pointer over a mark shows its note
- **Renumber Notes** — 1, 2, 3… in the order the text refers to them, the notes at the end put in that order
- **Bulleted List**, **Numbered List**, **Task List**, **Quote** — on the lines (a list replaces another)
- **Code Block** — fences the lines in ``` ```; **Horizontal Rule**; **Table…** — rows × columns

## Accolade

- **Notes** — the notepad is kept at the end in `<!-- Notes` … `-->`
- **Front matter** — a `---` block at the top is left out of the preview and the word count
- **Command palette** — Ctrl+K or F1: every command by name
- **Preview** — Ctrl+Shift+P shows the document formatted beside the editor; its Pages are the PDF's, links clickable (Tab goes through them)
- **Notes in the PDF** — at the foot of the page by default; Page Setup: after each chapter or after the text, numbered through or again in each chapter
- **Lists go on** — Enter at the end of an item starts the next; Enter on an empty item ends the list
- **Outline** — the headings (Ctrl+Shift+O)
- **Document type** — View > Document Type switches between Screenplay and Markdown

## As a source

- **Headings** — become sections with Adaptation > Source Headings as Sections
- **Passages** — each paragraph, quote and list item is one passage to adapt
- **Breaks** — `* * *` starts a new scene when Accolade reads the story; it is no passage

More: [Adapting a Story](adapting.md).

## Citations

Pandoc's citations, as Zettlr writes them: `[@doe99]`, `[@doe99, p. 12]`, `[see @doe99; @roe01, ch. 2]`, `[-@doe99]` (the author left out), and `@doe99` in the text (`@doe99 [p. 12]`). An address like `me@example.org` is no citation; `@{a key, with spaces}` is one.

- **The library** — the front matter's `bibliography: refs.bib` (the first, if it lists more), else a `.bib` or CSL-JSON `.json` in the bundle's `assets/` or beside the document (the document's own name first: `essay.json`, `essay.bib`), else **Preferences > Editor > Citations: Main library**. CSL-JSON (Zotero's Better BibTeX: *Better CSL JSON*, kept updated) or BibTeX/BibLaTeX. A file exported again is read again; one caught half-written keeps the library read before.
- **@** — typing a key lists the library's entries (key, author, year, title): ↑↓ choose, Tab or Enter puts the key in, Esc closes; a tap on a phone. Keys, authors and titles match, without accents.
- **The pointer over a citation** shows its entry (a tap on a phone); a key the library lacks says so.
- **View > References** — what the document cites, in order, and the front matter's `nocite:`; a tap goes to the first citation.
- **Export with pandoc** (EPUB, Word, ODT, LaTeX…) — cites with `--citeproc` and the library, in the front matter's `csl:` style (a `.csl` file beside the document or in its bundle, or one Pandoc knows), Chicago author-date without one.
- **Accolade's own citations** — the Live Preview (Text and Pages), the PDF, Word, ODT, web page and plain text Accolade writes itself cite in one built-in author-date style after Chicago's: `[@doe99]` is (Doe 1999), `[@doe99, p. 12]` (Doe 1999, 12), `[see @a; @b]` (see Doe 1999; Roe 2001), `[-@doe99]` (1999), `@doe99` Doe (1999) and `@doe99 [p. 12]` Doe (1999, 12); two works by the same authors together (Doe 1999, 2001); four authors or more Doe et al.; the same authors and year twice 1999a, 1999b; a key the library lacks (key?). A **References** list follows the text: under a heading of its own, or at the end of your `# References` (or Bibliography, Works Cited) section, or in a `::: {#refs}` … `:::` block. It holds what is cited and the front matter's `nocite:` (`@*`: the whole library), sorted by author and year, with a hanging indent (in the Text preview as plain paragraphs), titles of books and journals in italics. Books, articles, chapters, theses, web pages, reports and the rest each have their form. Without a library the citations stay as written. Pandoc's exports use `--citeproc` and your `csl:` instead.

## Pictures

`![alt text](assets/harbour.png "Caption")` — a path beside the document or in its bundle's `assets/` (a name not found is looked for in `assets/` too), or a `data:` URI. PNG, JPEG, GIF and WebP show; an SVG shows as its alt text for now. A picture on the web is never fetched — not in the preview, not in an export: its alt text and link stand in its place.

- Alone in its paragraph: a picture of its own, as wide as the text at most (never larger than it is), its title under it as a caption (Page Setup: **Alt text as caption** for those without a title); in a line of text: as high as the line.
- The Live Preview (Text and Pages), the PDF, Word and ODT show it; the exported web page keeps the tag; Print puts the pictures in the page it prints.
- A picture that cannot be shown is a quiet box with its alt text and where it was.
- Importing a Word, ODT or web page keeps its pictures: copied into `Name.assets/` beside the file, linked from the Markdown.

