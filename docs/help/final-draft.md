# Coming from Final Draft

Accolade writes Fountain, a plain text file, but you can type it with the keys you know from Final Draft: Tab for a character, Enter for the dialogue, Tab again for a parenthetical. Underneath, every key writes ordinary Fountain that any other app reads the same way.

## The keyboard flow

### Tab on an empty line starts a character

On an empty line that is not inside a speech, Tab starts a character cue.

- Accolade adds the blank line above it if there is none, and one below it if action follows straight after.
- What you type on that line comes out in capitals.
- The header bar's element shows Character, and the status bar says "Character: type the name; Enter or Tab for the dialogue".
- For a name in a script without capitals (Hebrew, Arabic, Chinese), Accolade leaves it as typed when the language's rules read it as a cue (a name in the character list, one used before), and puts Fountain's `@` in front of it otherwise.

### Tab or Enter after the name goes to the dialogue

At the end of the cue, Tab or Enter starts the dialogue line under it. This works for a name you typed yourself in capitals too: Tab at its end goes to its dialogue.

### Tab in dialogue makes a parenthetical

- On an empty dialogue line, Tab writes `()` with the cursor inside.
- At the end of a dialogue line, Tab puts `()` on a new line under it, the cursor inside.

### Tab or Enter in a parenthetical goes back to dialogue

Inside the brackets (or right after them), Tab or Enter starts the next dialogue line. Enter in an empty `()` takes the brackets away and leaves you on the dialogue line.

### Enter on an empty dialogue line goes to action

Pressing Enter on an empty line in a speech ends the speech: that line stays as the blank line Fountain needs, and you type action on the next.

### Enter after a scene heading goes to action

Enter at the end of a scene heading writes it in capitals, adds the blank line after it and leaves you on the action line. If you are used to pressing Enter twice there, the second Enter adds nothing.

Enter at the end of a transition such as FADE OUT. also writes it in capitals and adds the `>` that makes other Fountain apps read it as a transition.

### An example

Typed with the flow, from an empty script (the keys in square brackets; the lines below only break where it is easy to read):

```
ext. pier - night [Enter]
Waves. [Enter] [Tab]
mara [Enter]
Who's there? [Enter] [Enter] [Tab]
the gull [Tab] [Tab]
squawks [Enter]
Kraa. [Enter] [Enter]
She laughs.
```

gives this Fountain:

```
EXT. PIER - NIGHT

Waves.

MARA
Who's there?

THE GULL
(squawks)
Kraa.

She laughs.
```

## Completion and suggestions come first

While the status bar shows "Tab: …", Tab takes what it offers instead of moving on in the flow:

- a character name or a scene heading being completed, from the ones already in the script;
- a line, an action or a heading suggested from the story you are adapting (see [Adapting a Story](adapting.md#suggestions-while-writing)).

Esc puts a suggestion away for that line.

## Turning it off

The flow follows Preferences > Editor > "Format the line on Enter (capitals for scene headings and transitions)". With it off, Enter only starts a new line, and Tab still takes a "Tab: …" offer but otherwise types spaces.

The flow is only for screenplays. A Markdown document keeps its own Enter (lists go on). In vim or helix keys, normal mode keeps Tab and Enter for itself; insert mode has the flow.

## Element keys

Final Draft sets a paragraph's element with a key or the element list. Accolade has both: the element list (or buttons) in the header bar, and Format menu keys.

- **Ctrl+1** — Scene Heading
- **Ctrl+2** — Action
- **Ctrl+3** — Character
- **Ctrl+4** — Parenthetical
- **Ctrl+5** — Dialogue
- **Ctrl+6** — Transition
- **Ctrl+7** — Centered
- **Ctrl+8** — Lyrics

On a Mac, Cmd instead of Ctrl. An element key rewrites the line with the Fountain marks that make it that element: `.` for a heading that does not start with INT. or EXT., `@` (or the character list) for a name in a script without capitals, `>` for a transition, `> … <` for centred text, `~` for lyrics, `!` for action that would read as something else. All shortcuts are in [Keyboard Shortcuts](shortcuts.md).

## What is different

- **The file is the text.** A .fountain file holds what you see, nothing hidden. A line's element comes from how it is written: capitals, blank lines, a mark at the start. There is no stored element per paragraph to get out of step.
- **Any editor can open it.** A co-writer can work on it in Highland, Beat, a phone notes app or a plain text editor.
- **Notes are in the text,** between `[[` and `]]`, and cut material can stay in the file inside `/*` and `*/`. See [Fountain Reference](fountain.md).
- **Production data lives with the script:** the scene number lock and revisions in a block at the end of the file, drafts and breakdown in a bundle or beside it. See [Production](production.md).

## Opening Final Draft files

File > Open… (in the browser: Open from Your Computer…) takes an .fdx file and turns it into Fountain. It opens unsaved: Save asks where to put the .fountain file and leaves the .fdx as it was.

What comes across:

- the title page;
- scene headings with their numbers (`#12A#`) and their colours (as `[[red]]` and the like);
- action, characters, parentheticals, dialogue, transitions, lyrics and dual dialogue;
- script notes, as Fountain notes;
- outline elements as sections, and summaries as synopses;
- elements Fountain has no name for (Shot, New Act, Cast List …) as action, centred or on a new page when Final Draft has them so;
- revisions: the script is kept as an issued draft in its latest colour, with its revision marks, until your next Issue;
- locked pages, kept in the script's Production block;
- breakdown tags, kept by their words.

## Saving for Final Draft

File > Export… with Final Draft (FDX) writes an .fdx file:

- each Fountain element as Final Draft's paragraph type, lyrics as a Lyrics type;
- scene numbers, and OMITTED scenes as a heading with their number;
- scene colours as Final Draft's scene colour, notes as script notes;
- each scene's length and page;
- the breakdown's tags, with the cast taken from the cues;
- with "Revision marks" on, the revision sets and which lines changed;
- locked pages, when the script has them.

Sections, synopses and boneyard are left out: Final Draft has no place for them.

Lexington writes the FDX the way Final Draft 13 sets up a new script. Check an important file in Final Draft itself before you send it on.

File > Export… also writes PDF in the usual screenplay layout. [File Formats](formats.md) has every format.

## Missing something?

If a Final Draft habit does not work here, or works differently from what you expect, Help > Send Feedback… opens a new issue on GitHub with Accolade's version filled in. Nothing is sent until you send it there.
