# Coming from Another App

Whatever you wrote your script in, it can come into Accolade, and go back out to the people who still use that app. Accolade reads and writes these through Lexington, its own converter: the same formats are on its command line (`lexington convert old.fadein new.fountain`, [Lexington on the Command Line](lexington.md)).

Every import opens as a new, unsaved script; Save puts it in a .fountain (or an .accolade) and leaves the original as it was. Check it once: each app keeps some things Fountain has another way of writing.

## From any program: its PDF

Every screenwriting app prints a PDF, so this is the way in that always works: File > Open… and pick the PDF ("Screenplay (PDF)"), or drop it on the window.

- Accolade reads the pages back into Fountain by their layout: headings, action, names, speeches, parentheticals, transitions, dual dialogue, scene numbers in the margins, the title page. Final Draft's, Highland's, Fade In's, WriterDuet's, Word's layouts and others read the same way, a stage play's names at the margin too.
- A scanned PDF is read with OCR; Accolade asks what language the pages are in.
- What came out wrong, it came out wrong the same way every time: Edit > Fix Lines Like This… sets every line of a kind right at once ([Fixing an Imported Script](fixing-imports.md)).
- A PDF shows how the script looked, not what the app knew about it: notes, colours and revisions are gone. When the app can save one of the formats below, that keeps more.

## Highland 2

- **Open:** a `.highland` opens as it is, a zip or a folder: it is Fountain inside, so nothing is converted. Its drafts, its revisions and the other files in it are kept.
- **Save:** Save keeps it a `.highland` that Highland opens; Save As… can make it an `.accolade` or a plain `.fountain`.
- Highland's `{{HEADER}}`, `{{INCLUDE}}`, `{{TOC}}` and the other directives work and print as Highland prints them. See [File Formats](formats.md#bundles).

## Final Draft

- **Open:** a `.fdx` opens with its scene numbers and colours, notes, revisions, locked pages and breakdown tags.
- **Save:** File > Export… > Final Draft (FDX): a copy for Final Draft, your script staying the one you write in.
- Final Draft's keys (Tab for a character, Enter for the dialogue) work here too: [Coming from Final Draft](final-draft.md).

## Fade In

- **Open:** a `.fadein` file, or an `.osf` (Fade In's Open Screenplay Format, from File > Export in Fade In), in any of its versions: 1.2, 2, 3 and 4.
- **What comes in:** every element and its bold, italic and underline, dual dialogue, scene numbers, page breaks, Fade In's notes as Fountain notes, a scene's synopsis as its synopsis, the title page.
- **Save:** Save As… with the Fade In type (or File > Export… > Fade In), a copy for Fade In. Fade In has no outline: a section (`# Act One`) or a synopsis not under a scene heading goes in a note on the next line, which Fade In does not print either.

## Trelby

- **Open:** a `.trelby` file.
- **What comes in:** every element, Trelby's notes as notes, its act breaks as sections, the title page.
- **Save:** Save As… with the Trelby type, a copy for Trelby. Trelby keeps plain text: no bold or italics, no dual dialogue (the two speeches one after the other), no scene numbers (Trelby numbers scenes itself), no page breaks; a centred line is action there. Deeper sections and synopses go as notes that read back as what they were.

## KIT Scenarist

- **Open:** a `.kitsp` project.
- **What comes in:** every element and its emphasis, dual dialogue, scene numbers and colours, folders as sections, scene descriptions as synopses, review comments as notes, the title page's fields (its genre is the credit, its year the draft date, as KIT's own Fountain import has them).
- **Save:** Save As… with the KIT Scenarist type, a copy as a new KIT project. KIT has no page breaks.
- KIT Scenarist's successor, Story Architect (STARC), opens Fountain and Final Draft files: take either of those.

## WriterDuet and Celtx

Neither has a file of its own to open outside it, but both export and import Final Draft and Fountain:

- **In:** export the script from WriterDuet or Celtx as Final Draft (.fdx), which keeps the most (notes, scene numbers), or as Fountain, and open that.
- **Out:** File > Export… > Final Draft (FDX), or save the Fountain itself, and import it there.
- Failing both, their PDF reads as any other ([From any program: its PDF](#from-any-program-its-pdf)).

## Storyboarder

Storyboarder's boards come into a script's storyboard: File > Import Storyboarder Boards… reads a `.storyboarder` project (or the `storyboards/` folder beside the script), its boards and their timings, into frame files (flattened PNGs) and `[[board:]]` notes in the script, each frame then drawn further in the app you choose (Inkscape, Krita). It is a one-way step: the boards live on as open files, not in Storyboarder.
