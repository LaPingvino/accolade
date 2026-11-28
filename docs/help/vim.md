# Vim and Helix Modes

Preferences > Editor > Keys makes the editor's keys vim's or helix's. This page is made from Accolade's own table of those keys, and a test types every key in each mode: none does something without a line here.

Both start in normal mode, where letters move and edit; i, a and the rest go to insert mode, where you write as ever — Tab and Enter there are the screenplay's ([Coming from Final Draft](final-draft.md)) — and Esc comes back. The menus' shortcuts work in every mode ([Keyboard Shortcuts](shortcuts.md)), and so do the command palette and the ribbon's key tips. **:** opens the command line, which completes every command as you type, Accolade's own (`:renumber`, `:move 3..5 after 9`, `:scene`) among vim's: [Ex Commands](ex.md).

The screenplay's own: a **scene** is a text object (`as`, `is`), so is a **section** with all it holds (`aS`, `iS`); `]]` and `[[` go from scene to scene, `]s` and `[s` from section to section. Inside a scene, a **speech** (`ad`, `id`), its **cue** (`ac`, `ic`), a **beat** (`aB`, `iB`), a **note** (`an`, `in`) and an **action paragraph** (`aa`, `ia`) are objects too, and `]d`, `]B` and `]n` go to the next speech, beat or note. They read the script as the window does, in its language: an Esperanto script's EN. and EKST. are its scene headings. A scene deleted or yanked as one (`das`, `yas`) is put back after or before the cursor's scene, never inside it; a speech or paragraph (`dad`, `yaa`) after or before the cursor's. Moving goes through the structure engine, as Alt+Up and View > Structure do: the links, tags, colours and locked numbers go with the scenes.

## Vim

### Modes

- **Esc** — back to normal mode (in vim onto the character typed last)
- **i a** — insert before / after the cursor (helix: the selection's start / end)
- **I A** — insert at the line's first character / its end
- **o O** — a new line below / above, inserting
- **v V** — visual mode by characters / by lines; again: back to normal

### Moving

- **h j k l** — left, down, up, right (helix: the selection dropped)
- **w b e** — the next word's start, the word's start before, the word's end (helix: selecting the word)
- **0 ^ $** — the line's start, its first character, its end
- **gg G** — the first line / the last
- **Enter, Backspace** — down a line to its start; left
- **1 … 9** — a count before a motion or command: 3w, 2dd, 2]]

### Changing

- **d c y + a motion** — delete, change (then insert), copy what the motion passes: dw, c$, y3j
- **dd cc yy** — the line (a count: lines)
- **D C Y** — to the line's end; Y the line
- **x X** — delete the character under / before the cursor
- **p P** — put after / before (lines after or before the line; a scene or section after or before the cursor's)
- **J** — join the line below onto this one
- **r + a character** — replace the character under the cursor
- **u, Ctrl+R** — undo, redo (each command is one step)

### Text objects and the structure

- **]] [[** — the next / previous scene heading (helix: select that scene)
- **]s [s** — the next / previous section heading (helix: select that section)
- **as is** — a text object: a scene with the blank lines after it / its inside: das, cis, yas, vis, 2das
- **aS iS** — a section with all it holds / its inside: daS, yiS, vaS
- **ad id** — a speech: its cue, parentheticals and dialogue (two side by side: the pair) / its dialogue lines: dad, cid, 2dad; yad then p puts it after the cursor's speech or paragraph
- **ac ic** — a character cue with its extension, (V.O.) or (CONT'D) / its name alone: cic renames, the extension kept
- **aB iB** — a beat (a synopsis standing on its own, a section marked a beat or every section with Every Section Is a Beat) / its lines: daB, yaB
- **an in** — a note with its [[ ]] / what it says: dan, cin
- **aa ia** — an action paragraph with the blank lines after it / alone: daa, yaa
- **m + an object** — move it through the structure engine: mas opens :'<,'>move , then after 12 or before #Act Two; mad, maa and a synopsis's maB select it for Alt+j and Alt+k
- **]d [d** — the next / previous speech's cue, each of a pair (helix: select that speech)
- **]B [B** — the next / previous beat (helix: select it)
- **]n [n** — the next / previous note (helix: select it)
- **Alt+j Alt+k** — move the unit at the cursor (a scene or section with all it holds, a speech, a paragraph) down / up past the next
- **Alt+h Alt+l** — promote / demote it (a scene out of its section, into the one before)
- **Alt+J Alt+K** — into the next / previous scene
- **Alt+H Alt+L** — pass the last speech on / take in the next (barf, slurp)

### Selecting

- **a motion in visual mode** — extends the selection; then d x y c act on it
- **vas viS …** — the scene or section as the selection, line by line

### Search and commands

- **/** — the search bar (Find)
- **:** — the command line, completing as you type (in visual or with a selection: '<,'> the lines): [Ex Commands](ex.md)

### Folds and the outline

- **za zc zo** — fold or unfold the heading at the cursor, fold it, unfold it
- **zR zM** — unfold all, fold all
- **Tab, Shift+Tab** — on a heading: cycle its fold; all of them
- **gj gk** — the next scene or section heading / this one's or the one before
- **gh gl** — the section the heading is in / the first heading inside it
- **gs** — the scene's guessed link to the source (Link Guess for This Scene…)

### Not there

Vim mode is not a full Vim: marks (m and ' with a letter: m is the structure's move), named registers (" and a letter: one register), macros (q, @), repeat (.), f t F T, %, * #, n N (the search bar's Find Next and Find Previous, Ctrl+G, do that), the word and paragraph objects (iw aw ip ap: the screenplay's ad aa aB stand in), block visual (Ctrl+V), > < ~, and :set.

## Helix

Selection first: a motion selects what it passes over, then a command acts on the selection.

### Modes

- **Esc** — back to normal mode (in vim onto the character typed last)
- **i a** — insert before / after the cursor (helix: the selection's start / end)
- **I A** — insert at the line's first character / its end
- **o O** — a new line below / above, inserting
- **v** — select mode: motions extend the selection; again: back

### Moving

- **h j k l** — left, down, up, right (helix: the selection dropped)
- **w b e** — the next word's start, the word's start before, the word's end (helix: selecting the word)
- **0 ^ $** — the line's start, its first character, its end
- **gg G** — the first line / the last
- **Enter, Backspace** — down a line to its start; left
- **1 … 9** — a count before a motion or command: 3w, 2dd, 2]]

### Changing

- **d c y** — delete, change, copy the selection (none: the character under the cursor)
- **p P** — put after / before (lines after or before the line; a scene or section after or before the cursor's)
- **r + a character** — replace the character under the cursor
- **u U, Ctrl+R** — undo, redo

### Text objects and the structure

- **]] [[** — the next / previous scene heading (helix: select that scene)
- **]s [s** — the next / previous section heading (helix: select that section)
- **mas mis maS miS** — select a scene, its inside, a section, its inside; then d c y p, or : to move it
- **mad mid mac mic maB miB man min maa mia** — select a speech, a cue, a beat, a note, an action paragraph (a: with its blank lines, extension or [[ ]]; i: inside); then d c y p
- **]d [d** — the next / previous speech's cue, each of a pair (helix: select that speech)
- **]B [B** — the next / previous beat (helix: select it)
- **]n [n** — the next / previous note (helix: select it)
- **Alt+j Alt+k** — move the unit at the cursor (a scene or section with all it holds, a speech, a paragraph) down / up past the next
- **Alt+h Alt+l** — promote / demote it (a scene out of its section, into the one before)
- **Alt+J Alt+K** — into the next / previous scene
- **Alt+H Alt+L** — pass the last speech on / take in the next (barf, slurp)

### Selecting

- **x** — the line; again: the next one too
- **%** — everything
- **;** — the selection down to the cursor

### Several selections

- **s** — select what a pattern matches inside the selections (asked on the command line); @ and a kind selects the screenplay's elements instead, read by the structure engine: @cue @speech @dialogue @paren @action @heading @section @transition @note @beat (% s @cue: every cue; mas s @dialogue: a scene's dialogue); @@ is a literal @
- **S** — split the selections where a pattern (or an @kind) matches
- **Alt+s** — split the selections into lines
- **C** — a selection like the last one on the next line
- **M** — in a speech: every speech of that character as a selection (each of a pair alone); then d c y i a at all of them
- **,** — keep the primary selection only
- **Alt+,** — drop the primary selection
- **d c i a with several** — act at every selection; typing goes to all of them

### Search and commands

- **/** — the search bar (Find)
- **:** — the command line, completing as you type (in visual or with a selection: '<,'> the lines): [Ex Commands](ex.md)

### Folds and the outline

- **za zc zo** — fold or unfold the heading at the cursor, fold it, unfold it
- **zR zM** — unfold all, fold all
- **Tab, Shift+Tab** — on a heading: cycle its fold; all of them
- **gj gk** — the next scene or section heading / this one's or the one before
- **gh gl** — the section the heading is in / the first heading inside it
- **gs** — the scene's guessed link to the source (Link Guess for This Scene…)

### Not there

Helix mode is not a full Helix: goto mode beyond gg (g and a letter: the outline's, as vim's), space mode, match mode beyond mi and ma with s and S (mm, ms), registers, macros, repeat (.), f t, n N (the search bar does that), the tree-sitter objects (a function, a class).

## Selecting by kind

Helix's `s` and `S` take the screenplay's kinds as well as patterns: `@cue`, `@speech`, `@dialogue`, `@paren`, `@action`, `@heading`, `@section`, `@transition`, `@note` and `@beat`, completed as you type the @. They read the script as the window does, in its language, never by a pattern: `%` then `s @cue` selects every cue; `mas` then `s @dialogue` a scene's dialogue; `maS` then `s @paren` and `d` takes a section's parentheticals out; `%`, `s @cue`, `s BOB` and `c` renames BOB everywhere. Select All Character Cues and the rest do the same in any keys, from the command palette ("select all cues") or the : line (:select-cue), and Alt+O grows a selection to the speech, the scene, the section. After ] [ m and an object's i or a, the status bar lists what can come next.

## How they differ from vim and helix

- One register, shared by the modes and the `:` line's :d and :y; the system clipboard is Ctrl+C and Ctrl+V as ever.
- Each command is one undo step, and Undo asks before taking back a step an issue tagged, as the menu's does.
- Ctrl and Alt keys of the menus come first: a menu's shortcut is never a mode's (the keyboard audit checks that, docs/KEYBOARD.md).
- The search is the search bar's, with its own options; `:s` and `:g` take vim's patterns.
