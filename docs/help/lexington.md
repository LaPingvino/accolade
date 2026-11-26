# Lexington on the Command Line

Lexington is the converter inside Accolade: it reads and prints every script Accolade exports. It also works on its own, on the command line, for batches, scripts and servers.

## Getting it

- **With Go:** `go install github.com/lapingvino/lexington@latest` puts `lexington` in your Go bin folder.
- **Ready-made programs:** the releases on GitHub, github.com/LaPingvino/lexington/releases.
- **In the browser, nothing to install:** lapingvino.github.io/lexington converts a script on the page; it stays on your computer.
- **Accolade itself:** started with options, Accolade runs Lexington instead of opening a window: `accolade -i script.fountain -o script.pdf`.

## Reading and writing

Lexington reads Fountain, FDX (Final Draft), Fade In (`.fadein`, `.osf`), Trelby (`.trelby`), KIT Scenarist (`.kitsp`), its own lex format and PDF. A PDF is read back into a script by its indents (Lexington's own, Final Draft's, Highland's, Fade In's, Word's); a scan with only images is read with OCR when tesseract is installed. The input format comes from the file name, else from what the file holds, or `-from`.

`lexington convert IN OUT` converts any of them into any format below, each by its name: `lexington convert script.pdf script.fountain`, `lexington convert film.fadein film.fdx`. Other options go before or after the two files. See [Coming from Another App](coming-from.md).

It writes the formats below; the output format comes from the file name, or `-to`.

<!-- formats -->

- **Its own:** PDF (`pdf`), HTML (`html`), Word (DOCX) (`docx`), OpenDocument (ODT) (`odt`), Final Draft (FDX) (`fdx`), Fade In (`fadein`), Open Screenplay Format (OSF) (`osf`), Trelby (`trelby`), KIT Scenarist (`kitsp`), Fountain (`fountain`), LaTeX (`latex`), Lexington's own (lex) (`lex`).
- **With other programs installed:** PDF through HTML (`htmlpdf`), with wkhtmltopdf; PDF through LaTeX (`latexpdf`), with LaTeX; EPUB (`epub`), with pandoc; Kindle (AZW3) (`azw3`), with Calibre; Mobipocket (MOBI) (`mobi`), with Calibre; Mobipocket (PRC) (`prc`), with Calibre; FictionBook (FB2) (`fb2`), with Calibre; eReader (PDB) (`pdb`), with Calibre; Sony Reader (LRF) (`lrf`), with Calibre; Microsoft Reader (LIT) (`lit`), with Calibre; Rich Text (RTF) (`rtf`), with pandoc; Markdown (`markdown`), with pandoc; reStructuredText (`rst`), with pandoc; Pandoc JSON (`json`), with pandoc; Pandoc native (`native`), with pandoc; Manual page (`man`), with pandoc; Textile (`textile`), with pandoc; MediaWiki (`mediawiki`), with pandoc; Org mode (`org`), with pandoc; AsciiDoc (`asciidoc`), with pandoc.

<!-- /formats -->

## Every flag

Made from Lexington's own list, so none is missing. `-i` and `-o` default to standard input and output, so Lexington works in a pipe.

<!-- flags -->

### Input and output

- `-i string` — Input from provided filename. - means standard input. A bundle (.accolade, .highland, .textbundle, .textpack, zipped or a folder) is read for its text; a .fadein too, zipped or a folder. (default "-")
- `-o string` — Output to provided filename. - means standard output. A bundle (.accolade, .highland, .textbundle, .textpack) gets the script as Fountain, all else in it kept; a .fadein its OSF document. (default "-")
- `-from string` — Input file type: fountain, fdx, pdf, fadein, osf, trelby, kitsp, lex (default: as the input's name, else its content, says).
- `-to string` — Output file type. Choose from pdf, html, docx, odt, fdx, fadein, osf, trelby, kitsp, fountain, latex, lex, or external formats requiring other programs: htmlpdf, latexpdf, epub, azw3, mobi, prc, fb2, pdb, lrf, lit, rtf, markdown, rst, json, native, man, textile, mediawiki, org, asciidoc.
- `-template string` — Path to a custom template file (e.g., for HTML, FDX, or LaTeX output).

### Format, paper and pagination

- `-e string` — Element settings to use: a preset (see -presets) or a set from the settings file (default: an .accolade's own format, else default).
- `-presets` — List the presets for -e (stage play, radio, ...).
- `-formats string` — Folders of format files of your own (name.toml, as Lexington's presets), as a path list, to use with -e.
- `-page string` — Paper size for pdf, docx and odt: letter, a4 or a5 (default: the preset's suggestion, else letter).
- `-pagination string` — How pages break in pdf, docx, odt and printed html: all (the default: headings, names and parentheticals never last on a page, transitions never first, two lines of a paragraph on each side of a break, speeches broken after a sentence with (MORE) and CONT'D), none, or keep, widows and more joined with + (keep+widows).
- `-config string` — Configuration file to use. (default "lexington.toml")
- `-dumpconfig` — Dump the default configuration to the location of --config to be adapted manually.
- `-scenein string` — Configuration to use for scene header detection on input.
- `-sceneout string` — Configuration to use for scene header detection on output.

### Scene numbers, pages and reports

- `-number string` — Number the scenes without a number: count (on from the last whole number), or as a locked script, keeping the numbers there are: 10A (a new scene after 10 is 10A, the default style) or A11 (before 11).
- `-mergeomitted` — Print a run of OMITTED scenes as one heading, 12-14 OMITTED (not in FDX, Fountain or lex, which keep one a number).
- `-continueds` — Print the scenes' CONTINUEDs in the PDF: (CONTINUED) at the foot of a page a scene goes on from, CONTINUED: at the top of the next.
- `-lockpages` — Write the lock of the script's pages as the PDF prints them (with -lock: locked again, A-pages kept), instead of the script.
- `-lock string` — Keep the locked pages in this file (as -lockpages writes them): each page's breaks and number, A-pages (12A) for text that grew; the FDX gets them as <LockedPages>.
- `-lengths` — Write each scene's page and length in eighths into the FDX (SceneProperties Page and Length), measured as -e and -page print them.
- `-report string` — Write a report as CSV instead of the script: scenes (number, heading, page, length in eighths, who speaks), locations, characters or cast (numbered by size). Pages are measured as -e and -page print them.

### Colours

- `-colors` — Print scene headings and sections in the colors of their notes ([[red]], [[COLOR BLUE]], [[#3366ff]]).
- `-colorstyle string` — How -colors shows a color: line (a bar beside the scene, the default), marker, background or text (or with -colorscope, -colortint: marker+block, line+tint, ...); switches colors on.
- `-colorscope string` — What a marker, background or text color covers: heading (the default) or block (the scene, or a section whole, with the color).
- `-colortint` — With the line style: a light tint under the block too.
- `-colorpatterns` — With the line style: each color's bar in a pattern of its own (dashes, dots, double, ...), not told apart by hue alone.

### Parts of the script and sides

- `-tag value` — Keep only the scenes and sections with this note ([[London]]) or color, with all under them; give it more than once for any of several.
- `-character string` — Keep this character's sides; needs -sides.
- `-sides string` — With -character: full (the scenes they speak in), cue (their speeches after their cues) or isolated (their speeches only).
- `-cuewords int` — With -sides cue: the cue's last words only, this many (0: the whole speech before).
- `-strip string` — Take out of the Fountain input, as a comma list: Accolade's data always (its end blocks, the language line, [[from: …]] links), and the presets clean (Accolade's data only), reading (notes and outline), writer (scene numbers and OMITTED); or the options notes (every note [[…]]), boneyard (every boneyard /* … */, an omitted scene's kept text too), numbers (the scene numbers #12#), omitted (the OMITTED headings, with the scene kept under each), titlepage (the title page), outline (the writer's outline: sections # and synopses =). They add up: writer,notes. Fountain out (the default with -strip) is the text as it was, less what went.

### Other

- `-lint` — Run the Fountain linter on the input file
- `-help` — Show this help message
- `-version` — Show version information
- `-footer string` — The page's footer for a script with no {{FOOTER: …}} of its own, as -header.
- `-form string` — The form of a bundle written: zip or folder (default: the bundle's own when rewritten, else zip, a .textbundle a folder).
- `-header string` — The page's header for a script with no {{HEADER: …}} of its own, as Highland writes one: "Title | %p." (%p the page, %pp the last page, %# the section; left | centre | right).
- `-keeploose` — Write the Fountain input so that it reads under the spec as Lexington before 1.19 read it (blank lines and force marks added), instead of converting it; says how many places changed.
- `-loose` — Read Fountain as Lexington before 1.19 did: a scene word starts a heading anywhere, capitals after action are a name, any line ending in " TO:" is a transition, a line of spaces ends a speech.
- `-osfversion string` — The Open Screenplay Format version osf and fadein are written in: 4.0 (the default, Fade In 4's), 2.0 or 1.2.
- `-titlepage string` — The title page template: standard (Fountain's, the default), bold, poster, or one of the [TitlePages] in the settings file (see -titlepages).
- `-titlepages` — List the title page templates for -titlepage.

<!-- /flags -->

## Examples

Fountain to a PDF, on A4:

```sh
lexington -i script.fountain -page a4 -o script.pdf
```

Final Draft to Fountain:

```sh
lexington -i script.fdx -o script.fountain
```

A printed (or scanned) PDF back to Fountain; a scan needs tesseract:

```sh
lexington -i scan.pdf -o script.fountain
```

Scene numbers for a shooting script, keeping the ones there are (a new scene after 10 becomes 10A), OMITTED runs printed as one line, with CONTINUEDs:

```sh
lexington -i script.fountain -number 10A -mergeomitted -continueds -o shooting.pdf
```

Locked pages: write the lock once, then print later revisions on the same pages (text that grows goes onto A-pages):

```sh
lexington -i script.fountain -lockpages -o pages.lock
lexington -i script.fountain -lock pages.lock -o revised.pdf
```

A scene breakdown as CSV (number, heading, page, length in eighths, who speaks), and the cast numbered by size:

```sh
lexington -i script.fountain -report scenes -o scenes.csv
lexington -i script.fountain -report cast -o cast.csv
```

One actor's sides, the scenes they speak in:

```sh
lexington -i script.fountain -character ANNA -sides full -o anna.pdf
```

A clean Fountain copy for another app (Accolade's blocks and source links left out, as File > Export Clean Fountain… does), and the writer's version without scene numbers, OMITTED headings and notes:

```sh
lexington -i script.fountain -strip clean -o clean.fountain
lexington -i script.fountain -strip writer,notes -o draft.fountain
```

See also the [Lexington Flags](cheat-lexington.md) cheat sheet and [File Formats](formats.md).
