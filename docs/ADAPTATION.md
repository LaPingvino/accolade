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

- **Links** are ordinary Fountain notes in the script: `[[from: "She opened the door to…"]]`. They hold the
  passage's first words, so a link is found again by its text, and edits elsewhere in the draft don't break it.
- **Moving the cursor** in a scene highlights its passages in the source; clicking a passage goes to its scene.
- **Each passage has a status:** not started, adapted, or cut on purpose. The status shows as a bar beside the passage,
  with a count ("62% adapted").
- **Next passage** jumps to the first passage not yet adapted and starts a scene for it, with the link in place.
- **Chapters become sections:** Markdown `## Chapter 3` can be carried over as the Fountain section `## Chapter 3`.

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

What the writer accepts, changes or ignores goes into the adaptation's notes. They are kept in the script as a
Fountain boneyard block (like View > Notes), so they travel with the file and stay readable:

```
/* Adaptation
characters: ANNA = Anna, Annie, Miss Kovač
not characters: Monday, Lord
places: the house = INT. KOVAČ HOUSE
*/
```

- **A cue renamed after it was accepted:** the old name becomes another name for the new one.
- **A suggested name deleted twice:** it goes on the "not characters" list.
- **A heading written by hand for a passage:** the place is matched to it from then on.
- **A passage skipped with Next:** it is asked whether to mark it as cut.

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
7. The learned notes.
8. AI, if wanted.
