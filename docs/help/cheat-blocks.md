# Accolade Blocks & Links

All of this is plain Fountain (notes and boneyard): other apps open the file and do not print it. Full page: [Fountain Reference](fountain.md#accolades-own-markup).

## End blocks

Notes is always last; Accolade reads the others in any order. In the editor each is folded to one line that says what it holds (▸ Production · scene numbers locked · Blue): click it or move the cursor in to read it, and it folds again when you leave. Its ▸ in the margin, View > Fold or Unfold Here (Shift+.) or vim's zo keeps it open; Unfold All opens them all. An omitted scene's kept text folds the same way.

- **Production** — `/* Production` … `*/`: lock, revision, pages
- **Adaptation** — `/* Adaptation` … `*/`: cuts and learned notes
- **Characters** — `/* Characters` … `*/`: the character list and its notes
- **Fix-Up** — `/* Fix-Up` … `*/`: Fix Lines Like This rules, to apply again after a re-import
- **Source** — `/* Source` … `*/`: the story, when kept in the script
- **Notes** — `/* Notes` … `*/`: the notepad (Ctrl+Shift+K)

## Inside the blocks

- **Number lock** — `scene numbers: locked, 10A`
- **Revision** — `revision: Blue, 2026-11-20`
- **Page lock** — `pages: locked, letter, …` and a line per page
- **OMITTED runs** — `omitted runs: print as one`
- **Character** — `ANNA: the lead` then `  age: 34`, `  look:`, `  arc:`, `  actor:` indented
- **Listed only** — `ben` alone on a line: in the list, nothing written yet
- **Alias** — `characters: ANNA = Anna, Annie`
- **Not a name** — `not characters: Monday, Lord`
- **Place** — `places: the house = INT. KOVAČ HOUSE`
- **Cut passage** — `cut:` then `#:~:text=It was late.`

## In the script

- **Source link** — `[[from: #:~:text=She opened the door,into the rain.]]`, drawn as a chip: ↗ source
- **Link to a file** — `[[from: story.md#:~:text=said Anna.-,Yes.]]`
- **Old link** — `[[from: "She opened the door to…"]]`
- **Colour** — `INT. KITCHEN - NIGHT [[red]]` or `[[#ff8800]]`
- **Tag** — `INT. KITCHEN - NIGHT [[London]]` (Export filters)
- **Note** — `[[Check the date.]]`
- **Scene number** — `INT. KITCHEN - NIGHT #12A#`
- **Any label** — `#I-1-A#` is kept as it is
- **Omitted scene** — `.OMITTED #12#` with `/* Omitted 12` … `*/` under it
- **Boneyard** — `/*` … `*/`: kept, never printed

Breakdown tags and the schedule are not in the text: they live in a file beside the script. See [Production](production.md).
