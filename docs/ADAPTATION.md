# Adapting a story into a screenplay (plan)

Joop, 2026-11-19: Accolade comes from Markdown. Import DOCX, ODT, PDF … into a Markdown story draft, then work it
into a screenplay, slowly, passage by passage, with the draft beside the script. Nothing here is built yet.

Nothing found does this. Scrivener keeps prose and script in one project but doesn't track which passage became which
scene. Highland writes both but has no adaptation workflow. FluidScriptr has AI write the whole screenplay at once.
BookNLP (Python, English) is the research reference for finding characters in prose and attributing quotes; the
methods carry over, the code doesn't.

## 1. Markdown as a first-class file

A `.md` file opens with Markdown colouring, an outline from its headings, and a formatted preview (goldmark, pure Go,
so the browser version has it too). Search, vim, spell check, stats, collaboration and autosave are shared with Fountain.

## 2. Import and export

Built in, so the browser version has them as well:

- DOCX and ODT into Markdown: paragraphs, headings, bold and italic (zipped XML).
- PDF into Markdown through Lexington's `pdfin`, OCR included, with a prose mode.
- HTML into Markdown.
- Markdown to HTML (goldmark), and to DOCX, ODT and PDF through Lexington's writers in a prose mode.

pandoc, when installed on the desktop, adds the rest (EPUB, RTF, LaTeX …).

## 3. The source pane and its links

`story.md` next to `story.fountain` is the script's source. The Notes pane can show it.

- **Links** are ordinary Fountain notes in the script, in the browsers' Text Fragments syntax (below):
  `[[from: #:~:text=She opened the door to the garden and,out into the rain.]]`. They hold the passage's words,
  so a link is found again by its text, and edits elsewhere in the draft don't break it.
- **Moving the cursor** in a scene highlights its passages in the source; clicking a passage goes to its scene.
- **Each passage has a status:** not started, adapted, or cut on purpose. The status shows as a bar beside the passage,
  with a count ("62% adapted").
- **Next passage** jumps to the first passage not yet adapted and starts a scene for it, with the link in place.
- **Chapters become sections:** Markdown `## Chapter 3` can be carried over as the Fountain section `## Chapter 3`.

### The link format

Joop, 2026-11-20: the WICG's URL Fragment Text Directives (https://wicg.github.io/scroll-to-text-fragment/, in
Chrome, Safari and Firefox), so a link is also an address into the source as a web page. Built in textfrag.go.

```
[[from: #:~:text=She opened the door,stepped out]]
[[from: story.md#:~:text=said Anna.-,Yes.]]
[[from: #:~:text=The rain kept falling.,-It was late]]
[[from: #:~:text=One,two&text=Five]]
```

- After `from:` an optional file, then `#:~:` and one or more `text=` joined by `&`; each is
  `text=[prefix-,]textStart[,textEnd][,-suffix]`.
- **textStart[,textEnd] is the passage range:** from the passage textStart is in to the passage of the first
  textEnd after it. One passage, short (up to 12 words): textStart is all of it; longer: its first 8 words and
  its last 4. A term is matched in one block of the page, so a passage of several (a quote's paragraphs, a list
  item's, a nested item) has textStart from its first and textEnd from its last. A link may span consecutive passages (a suggestion of neighbours is one link). Several `text=` are
  one link each, at the note's place.
- **prefix- and -suffix** are the words just before and after: in the passage, or at the end of the one before
  (the start of the one after). Written only when needed: when the words are found elsewhere too, Accolade first
  quotes more of textStart (then of textEnd), then adds the words before, then those after. They replace the old
  form's "(2)". Two passages whose words and neighbours are all alike cannot be told apart: the link is ambiguous,
  and Mark as Cut (or Skip) writes the old form's `"…" (2)` for it, which still counts.
- **The file** is optional: none is the script's source. A name is the source's when it is the name of any of
  its layers (the file beside the script, the bundle's entry, the file chosen, story.md for one kept in the
  script), its extension aside (story.html, the source exported, is it too). One source for now: a link naming
  another is read against it, and the pane says so. Accolade writes no file part.
- **Written readable:** the passage's own characters as its page shows them (the browser compares the text: no
  link targets, list marks or footnote marks; prose.Doc.Shown), only `,` `-` `&` `%`
  percent-encoded as the grammar needs, and `[` `]` so the note's `]]` stays its own; spaces stay spaces. Read
  both encoded and plain: `%20` or a space, a `-` inside a word taken as text (the spec rejects it), spaces
  around the parts aside, an `&` not followed by `text=` or another `name=` part of the text before it. A
  directive other than `text=` is ignored, as browsers do. A fragment that cannot be
  read stays a link, shown as one that cannot be read (and an entry so in the block is kept as written).
- **Matching** is the spec's, on the passages (each one block): every term within one passage, as whole words;
  the prefix right before textStart, the suffix right after the match (whitespace and wordless breaks between
  allowed, as the spec's "next non-whitespace position"); textEnd the first after textStart that the suffix
  follows. Unlike a browser: words are compared normalised (case, quotes and punctuation aside; accents count),
  a link that fits more than one place is ambiguous (a browser takes the first), and near matches are kept: a
  textStart of 4 or more words with a quarter edited, near a passage's opening; a textEnd of 4 or more near the
  end of one of the next 40 passages; textEnd or the context not found, the start's passage alone. Near links
  say the source changed since they were linked.
- **The old form,** `[[from: "first words…"]]` (the words a passage opens with), and `"…" (2)` in the block,
  are read as before. Adaptation > Update Source Links writes every link and block entry that is found in the new
  form, for the same passages, in one undo step; a near one keeps its words (so it still says the source
  changed); ambiguous and missing ones, and those the new form cannot tell apart, stay as they were.
- **Exports:** the source exported or printed as a web page (Export, Print) is the page the fragments are
  written for: `story.html#:~:text=…` opens at the passage. A link's words are the passage's text as the page
  shows it (tested against the spec's algorithm on the exported page). The script's own HTML export does not
  print notes (Lexington leaves them out), so its links are not hyperlinks there yet.

## 4. The reading helpers: heuristics, local, no AI needed

The draft is read with heuristics, for the script's language (language.go).

**Characters**
- Names are capitalised words not at the start of a sentence, counted. Words next to a speech verb weigh more
  ("said Anna", "Anna asked", a name addressed in dialogue), and so do words after a title (Mr, Mrs, Dr, Sra., S-ro).
- Words after "in", "at", "to" are kept apart as places, for scene headings.

**Dialogue**
- The quoting styles: “ ” " " ‘ ’ « » „ “, and the dash at the start of a paragraph (Portuguese, Spanish, French,
  Esperanto …).
- **Who speaks:**
  1. a speech verb with a name in the same sentence;
  2. otherwise the last named speaker in the paragraph;
  3. otherwise the other one of the two who were just talking.
- The manner of speaking becomes a parenthetical: "she whispered" → (whispering).

**Scenes**
- A new scene starts at a chapter heading, at a break (`* * *`, `---`), or at a phrase that shifts the place or the
  time ("The next morning", "Back at the house").
- From these comes a suggested heading, INT./EXT. PLACE - TIME. INT. or EXT. comes from words like room, kitchen,
  street, field.

**Unfilmable prose**
- Thoughts and feelings ("she thought", "he felt", "remembered") are flagged with a note: show it, cut it, or turn
  it into voice-over (V.O.).

## 5. Suggestions while writing

These build on the Tab completion that exists today (complete.go), with the suggestion in the status bar or as grey
text after the cursor; Tab accepts it.

- **Typing a character's name** offers that character's next line from the current passage.
- **On an empty line in a scene** comes the next piece of the passage: a name with their line, or the narration as
  action, in present tense.
- **Typing a new scene heading** offers the place and time read from the passage.
- **When a passage's lines are used**, the passage counts as adapted.

## 6. Learning from what the writer does

What the writer accepts, changes or skips goes into the adaptation's notes. They are kept in the script as a
Fountain boneyard block (like View > Notes), so they travel with the file and stay readable (built: step 7):

```
/* Adaptation
characters: ANNA = Anna, Annie, Miss Kovač
not characters: Monday, Lord
places: the house = INT. KOVAČ HOUSE
skipped links:
  INT. KITCHEN - NIGHT = #:~:text=Anna opened the kitchen door.
cut:
  #:~:text=It was late.-,The rain kept falling.
*/
```

The block sits after the script, before the Source block and the notes. Each section takes its entries on the
key's line or on the lines below it (indented, as a list, or not), up to the next `name:` line; several on the
key's line are parted by `;` (by `,` for not characters). Keys are read in any case, with a few other words for
them (aliases:, names:, not names:, skipped:). An entry that cannot be read is kept in its section as written,
and lines outside the sections are kept as they are. Accolade writes a section with one entry on the key's line
and more one per line below, in the order above.

- `characters:` a cue, then the source's names for that character. The source's reading takes them as one
  speaker, and suggestions give their lines under the cue.
- `not characters:` capitalised words that are no one: dropped as characters and as speakers.
- `places:` a place as the source reads it (KITCHEN, the house; a leading article does not count), then the
  heading the writer gave it, without the time. A heading suggested for that place is that one, with the time read.
- `skipped links:` a scene's heading, then a passage offered to it as a link and skipped: not offered to it again.
  The link in the link format (section 3; an `=` inside `text=` is no separator); the old form's quote is read.
- `cut:` (step 4) the passages cut on purpose, each as a link names it (section 3), one per line (or after
  `cut:`, parted by `;`); a range cuts each passage in it. The old form, the first words with "(2)" for the
  second of passages that open alike, is read.

When it learns (quietly: no question, no status; on the pane's follow after typing pauses, the pane shown or not,
the cursor's scene only; only the block changes, as part of the writer's last undo step, so Undo takes both):

- **A cue Tab put in, renamed** (or a line the source reads as someone's written under another cue): the source's
  name becomes another name of the new cue; not when the old name's own cue still speaks elsewhere in the script
  (then the reading was wrong, not the name).
- **A cue Tab put in, thrown away twice** (its name deleted, the line left with no cue, or renamed to another cue
  than the one learned): the name goes on the "not characters" list. A line deleted whole or reworded says nothing
  about the name.
- **A heading written for a scene linked to a passage** (over Next's placeholder, or a link put under a heading
  written in this session), other than any the source's place makes: the place is matched to it from then on.
- **A passage skipped with Next:** it is asked whether to mark it as cut (step 4).
- **Skip on a suggested link:** remembered under `skipped links:`.

It learns from actions, not from how the script stands: what the script had when it was opened teaches nothing, and
a cue or heading is looked at once the cursor has left it (half typed, nothing). The source pane's "What Accolade
learned (n)" line opens a list of the lessons, each with a button that takes it away (one undo step); a lesson
taken away is not learned again in the session. Nothing is learned while the script is shared live (a peer's edits
come into the window's undo steps).

## 7. Later, and optional

An AI suggestion per passage, which the writer accepts or edits, like machine translation in a translation tool. It
runs on the writer's own choice of tool, as GameViber's Terminal does, and is never needed for the rest.

## Order

1. Markdown mode.
2. Import and export.
3. The source pane with its links.
4. Statuses and Next passage.
5. The characters and dialogue heuristics, with tests on public-domain prose in several languages.
6. Suggestions while writing.
7. The learned notes (built).
8. AI, if wanted.
