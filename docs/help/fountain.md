# Fountain Reference

Fountain is plain text with a few rules. A line's place and its capitals say what it is: a blank line before a line in capitals makes a character cue, a line starting with INT. makes a scene heading. Accolade colours each element as you type and prints them in screenplay layout. The short version is the [Fountain Syntax](cheat-fountain.md) cheat sheet.

Every example below is a piece of a .fountain file, exactly as typed.

Accolade reads these rules as the Fountain spec has them. Lexington, which prints the script, read four of them more loosely before (version 1.19): a scene heading or a character cue right under a line of action, any line ending in TO: as a transition, a line of spaces as the end of a speech. A script written against that reading says so when it opens, once: *Keep It As It Was* adds the blank lines and force marks that make it read as before (one Undo takes them back), *Read It the New Way* leaves it as it is.

## Title page

The title page is a set of `Key: value` lines at the very top of the file, before anything else. A blank line ends it. A value can go on the lines under its key, indented.

```
Title: The Long Rain
Credit: Written by
Author: Anna Kovač
Source: Based on the story by Tom Reed
Draft date: 20 November 2026
Contact:
    anna@example.com
    +31 6 1234 5678
```

Edit > Title Page… (Ctrl+Shift+T) fills these in for you.

## Scene headings

A line that starts with INT., EXT., INT./EXT., I/E or EST., with a blank line before and after it, is a scene heading. Accolade writes it in capitals when you press Enter.

```
INT. KITCHEN - NIGHT

Rain against the window.
```

Scripts in other languages have their own words: EN./EKST. in Esperanto, BIN./BUI. in Dutch, and so on. Edit > Script Language… chooses them, and a file named like `scene.eo.fountain` says its own language (a bundle keeps it inside; see [File Formats](formats.md#a-script-in-another-language)).

### Forced scene headings

A full stop at the start makes any line a heading. Two full stops do not: `...` stays action.

```
.FLASHBACK

.THE ROOF OF THE WORLD
```

### Scene numbers

A number between `#` signs at the end of the heading is the scene's number. Fountain allows any label, not only digits.

```
INT. KITCHEN - NIGHT #12#

EXT. GARDEN - DAY #12A#

INT. HALL - DAY #I-1-A#
```

Production > Lock Scene Numbers writes these for every scene. See [Scene numbers and locks](#scene-numbers-and-locks) below.

## Action

Any paragraph that is not something else is action. Line breaks and blank lines are kept as typed.

```
Anna stands at the window. She does not turn round.

Behind her, the kettle starts to whistle.
```

An exclamation mark at the start forces action, for a line that would otherwise read as a cue or a heading:

```
!BANG. The door slams.
```

## Characters and dialogue

A line in capitals with a blank line before it and text right under it is a character cue. The lines under it are the dialogue, up to the next blank line. A line of two spaces keeps the speech going: a blank line inside it.

```
ANNA
Is it still raining?

TOM
It never stops here.
```

### Forced character names

An `@` at the start makes the line a cue, even with lower case letters in it. Use it for names like McBride, and for scripts without capitals (Hebrew, Arabic, Chinese).

```
@McBRIDE
You're late.
```

### Extensions

What follows the name in brackets is its extension: V.O., O.S., CONT'D and so on.

```
ANNA (V.O.)
I left the next morning.
```

### Parentheticals

A line in brackets inside a speech is a parenthetical.

```
TOM
(quietly)
Don't.
```

### Dual dialogue

A `^` after the second cue prints the two speeches side by side.

```
ANNA
Now!

TOM ^
Now!
```

### Lyrics

A line starting with `~` is a lyric.

```
~Row, row, row your boat
~Gently down the stream
```

## Transitions

A line in capitals that ends in `TO:`, with blank lines around it, is a transition. Any other transition needs a `>` at the start; Accolade adds it when you press Enter after FADE OUT. and the like.

```
CUT TO:

> FADE OUT.
```

## Centred text

A line between `>` and `<` is centred.

```
> THE END <
```

## Emphasis

- `*italic*`
- `**bold**`
- `***bold italic***`
- `_underlined_`

Ctrl+B, Ctrl+I and Ctrl+U put the marks around the selection.

## Page breaks

A line of three or more `=` signs starts a new page.

```
===
```

## Notes

Text between double square brackets is a note. It is never printed. It can sit inside a line or take lines of its own.

```
Anna opens the letter. [[Check the date on it.]]

[[Maybe this scene goes before the funeral.]]
```

Accolade gives some notes a meaning of its own: colours and source links. See [Accolade's own markup](#accolades-own-markup).

## Boneyard

Text between `/*` and `*/` is left out of the script altogether: it is not printed and not counted. Use it for scenes you cut but want to keep.

```
/*
INT. CELLAR - NIGHT

A scene that did not make it.
*/
```

## Sections

Lines starting with `#` are sections: acts, sequences, chapters. More `#` signs make a deeper level. They show in the outline and the cards, never in print.

```
# ACT ONE

## The flood
```

## Synopses

A line starting with `=` is a synopsis of the section or scene above it. Not printed.

```
= Anna decides to leave.
```

## Accolade's own markup

Everything below is ordinary Fountain: notes and boneyard. Other Fountain apps (Highland, Beat, Slugline, Fade In) open the file and simply do not print it. Accolade reads more into it.

### Scene and section colours

A colour note on a heading or section line gives it that colour in the outline, the cards and, if you ask for it, the export. It is written the way Beat writes it. Format > Scene or Section Color… (Ctrl+Shift+L) puts it in.

```
INT. KITCHEN - NIGHT [[red]]

# ACT TWO [[color blue]]

EXT. FIELD - DAY [[#ff8800]]
```

The named colours are red, blue, green, pink, magenta, gray, purple, cyan, teal, yellow, orange and brown. A hex colour such as `#ff8800` works too.

Other notes on a heading, such as `[[London]]`, are tags: the Export dialog can print only the scenes with a tag or a colour.

### Source links

When you adapt a story, a note starting with `from:` links a scene to the passage of the source it comes from. Accolade writes the browsers' Text Fragment form: the passage's first words, then its last words after a comma.

```
INT. KITCHEN - NIGHT

[[from: #:~:text=She opened the door to the garden and,out into the rain.]]
```

A link may name a file in front, `[[from: story.md#:~:text=…]]`, and words just before or after the passage, to tell two alike passages apart. The older form, the passage's first words in quotes, is still read:

```
[[from: "She opened the door to the garden and…"]]
```

Adaptation > Update Source Links rewrites old links in the new form. More in [Adapting a Story](adapting.md#links).

### Scene numbers and locks

Once you lock the scene numbers (Production > Lock Scene Numbers), every heading carries its number, and a scene written later between 10 and 11 gets 10A (or A11, as Preferences > Script > Production says). Numbers are labels: Accolade only counts on from numbers like 10, 10A or A11, and leaves any other label (`#I-1-A#`, `#PROLOGUE#`) exactly as it is.

### OMITTED scenes

Production > Omit Scene keeps a cut scene's number in a locked script. The heading becomes OMITTED, and the scene goes into a boneyard right under it, so Production > Restore Scene can bring it back.

```
.OMITTED #12#

/* Omitted 12
INT. KITCHEN - NIGHT #12#

Anna makes tea.
*/
```

Each omitted scene keeps its own heading in the text. Printing a run of them as one line ("12-14 OMITTED") is a setting, chosen in Production > Renumber Scenes… and kept in the Production block.

### End blocks

At the end of the script Accolade keeps up to five boneyard blocks. Each starts with `/*` and its name on the first line.

```
/* Production
scene numbers: locked, 10A
revision: Blue, 2026-11-20
*/

/* Adaptation
characters: ANNA = Anna, Annie
cut:
  #:~:text=It was late.
*/

/* Characters
ANNA: the lead, thirty-four
  age: 34
  look: a red coat, always
  arc: from fear to trust
  actor: Jane Doe
  She keeps the letters she never sends.
ben
*/

/* Source
# The Long Rain

She opened the door to the garden and stepped out into the rain.
*/

/* Notes
Ask Tom about the ending.
*/
```

- **Production** holds the scene number lock, the current revision, the page lock and the OMITTED print setting.
- **Adaptation** holds the passages you cut on purpose and what Accolade learned while you adapted: names, places, skipped links.
- **Characters** is the character list (View > Characters): a name at the start of a line, its note after a colon, and under it, indented, the fields `age:`, `look:`, `arc:` and `actor:` and more lines of the note. A name alone is a character with nothing written yet. Names keep the case they are written in, so a list of names in small letters works too; a cue matches its entry in any case.
- **Source** holds the story itself, when you choose Adaptation > Keep Source in the Script. Otherwise the source lives beside the script or in the bundle.
- **Notes** is the notepad (View > Notes, Ctrl+Shift+K).

Notes always comes last. Accolade puts a new Production, Adaptation or Characters block straight after the script and the Source block just before Notes, and it reads the others in any order. You can read and edit the blocks by hand; lines Accolade does not know are kept.

Other apps ignore the blocks, but they travel with the file. To hand the script on without them, use File > Export Clean Fountain… (see [File Formats](formats.md#clean-fountain)).

### Breakdown tags

Breakdown tags are not written in the script. Accolade keeps them in a file beside it (in a bundle, or in its own data folder), found again by their words. See [Production](production.md#breakdown).
