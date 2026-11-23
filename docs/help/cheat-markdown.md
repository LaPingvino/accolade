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

## Pictures

`![alt text](assets/harbour.png "Caption")` — a path beside the document or in its bundle's `assets/` (a name not found is looked for in `assets/` too), or a `data:` URI. PNG, JPEG, GIF and WebP show; an SVG shows as its alt text for now. A picture on the web is never fetched — not in the preview, not in an export: its alt text and link stand in its place.

- Alone in its paragraph: a picture of its own, as wide as the text at most (never larger than it is), its title under it as a caption (Page Setup: **Alt text as caption** for those without a title); in a line of text: as high as the line.
- The Live Preview (Text and Pages), the PDF, Word and ODT show it; the exported web page keeps the tag; Print puts the pictures in the page it prints.
- A picture that cannot be shown is a quiet box with its alt text and where it was.
- Importing a Word, ODT or web page keeps its pictures: copied into `Name.assets/` beside the file, linked from the Markdown.

