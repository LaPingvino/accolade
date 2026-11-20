# Accolade Blocks & Links

All of this is plain Fountain (notes and boneyard): other apps open the file and do not print it. Full page: [Fountain Reference](fountain.md#accolades-own-markup).

## End blocks

Notes is always last; Accolade reads the other three in any order.

- **Production** — `/* Production` … `*/`: lock, revision, pages
- **Adaptation** — `/* Adaptation` … `*/`: cuts and learned notes
- **Source** — `/* Source` … `*/`: the story, when kept in the script
- **Notes** — `/* Notes` … `*/`: the notepad (Ctrl+Shift+K)

## Inside the blocks

- **Number lock** — `scene numbers: locked, 10A`
- **Revision** — `revision: Blue, 2026-11-20`
- **Page lock** — `pages: locked, letter, …` and a line per page
- **OMITTED runs** — `omitted runs: print as one`
- **Alias** — `characters: ANNA = Anna, Annie`
- **Not a name** — `not characters: Monday, Lord`
- **Place** — `places: the house = INT. KOVAČ HOUSE`
- **Cut passage** — `cut:` then `#:~:text=It was late.`

## In the script

- **Source link** — `[[from: #:~:text=She opened the door,into the rain.]]`
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
