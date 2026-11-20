# Adapting a Story

Accolade can keep a story beside the script you make from it: a novel, a short story, a play in prose. You work through it passage by passage. The script links each scene to the passages it comes from, and the source shows what is adapted, what you cut, and what is still to do.

Everything Accolade writes for this is ordinary Fountain: a note in the script for each link, and a boneyard block at the end for cuts and what it learned. Other Fountain apps open the script and print it without them.

The commands are in the Adaptation menu, which is there for screenplays (not for Markdown documents).

## The source

The source is a story in Markdown. Its paragraphs, headings, quotes and list items are the passages you adapt.

### Choosing a source

Adaptation > Choose Source… takes a story as Markdown (.md, .markdown), plain text (.txt), Word (.docx), OpenDocument (.odt) or a web page (.html). Accolade converts it to Markdown and keeps that copy for the script, so the script finds it again after a rename or Save As.

A story you have as PDF, EPUB or another format: open it with File > Import as Markdown… (or Import with pandoc… on the desktop), save it as .md, then choose that.

### Where the source is kept

Accolade looks for a script's source in this order and uses the first it finds. The pane names the one in use.

1. **In the script,** in a `/* Source` block at its end. Adaptation > Keep Source in the Script moves it in, and again to move it out. Off by default.
2. **In the bundle,** for a script saved as .accolade, .highland or another bundle: the source travels inside the file.
3. **Kept by Accolade,** the copy Choose Source… made, in Accolade's data folder (in the browser, its kept settings).
4. **Beside the script:** `story.md` next to `story.fountain`.

Adaptation > Save Source As… writes the source out as a file of its own (the browser downloads it).

### Starting from a story

With a story open as a Markdown document, File > Start a Screenplay from This Source (also a button in the toolbar) makes a new script for it:

- a title page that names the story (`Source: Based on …`);
- the story's chapters as sections;
- the first scene started for the first passage, with its link.

A saved .md on the desktop stays the source itself, so you can go on writing the story. A script that has a source but nothing written yet gets the same offer in the source pane.

## The source pane

View > Source, or Adaptation > Source Pane, shows the source as a tab beside Notes. Each passage is a row.

- **The cursor's scene lights its passages,** and the pane scrolls to them.
- **Tapping a passage picks it** and goes to the scene that links to it. Tap again for the next scene, if more than one does.
- **Edit** opens the source as text in the pane; **Done** saves it where it is kept. If another program changes the source file, the pane reads it again.
- **What the source shows,** at the bottom, is how Accolade reads the picked passage: who says what, the scene heading it suggests, what a camera cannot see, the names in it.

On a narrow screen (a phone), the panes go under the script, and a toolbar button shows or hides the source.

## Links

A link is a Fountain note in the scene, in the browsers' Text Fragment form. It holds the passage's own words: its first words, and for a longer passage its last words after a comma.

```
INT. KITCHEN - NIGHT

[[from: #:~:text=She opened the door to the garden and,out into the rain.]]
```

Because a link holds words, not a line number, the story can change around it and the link still finds its passage.

### Making a link

Pick the passage in the pane, put the cursor in the scene, and choose Adaptation > Link to Source Passage. The link goes in at the cursor: on an empty line, after a line of dialogue, or on the line under a heading.

When the same words are in the story more than once, Accolade quotes more of the passage, or the words just before or after it (`text=said Anna.-,Yes.`). If two passages and their neighbours are exactly alike, the link cannot tell them apart and the status bar says so.

### When the source changes

A link whose words were edited in the source is still found if most of it is the same. The pane then shows that passage's mark in the warning colour, and the count says how many are "changed since linked". Link the scene again to bring it up to date.

### Old links

Earlier versions of Accolade wrote the first words in quotes: `[[from: "She opened the door to the…"]]`. These are still read. Adaptation > Update Source Links rewrites every link it can find in the new form, in one undo step. Links it cannot place are left as they are.

### Suggested links

A script written before it had a source has no links. Adaptation > Suggest Links to the Source goes through the scenes and finds the passage each one reads like: a line of dialogue the story has, or the passage's words in the scene's. For each, the pane asks: Link, Skip or Not Now. Nothing is linked unless you say so. A skipped one is remembered and not offered to that scene again.

## Statuses

Every passage that is not a heading or a break has one of three statuses, shown as a mark at its left:

- **adapted** (a solid green mark): a scene links to it;
- **cut** (a dashed red mark): you cut it on purpose;
- **to do** (a thin grey line): neither yet.

The count at the top of the pane reads like "50% adapted · 1 cut · 3 to go". The percentage is of the passages you did not cut.

### Cutting a passage

Pick it and press Mark as Cut in the pane, or choose Adaptation > Mark Source Passage as Cut. The same command on a cut passage takes the cut back. Cuts are kept in the script's `/* Adaptation` block under `cut:`, so they travel with it. Linking a cut passage takes its cut away.

## Next Source Passage

Adaptation > Next Source Passage (Ctrl+Alt+J), or Next in the pane, takes you to the next passage still to do: the first after the last passage the cursor's scene links to. At the end of the story it goes round to the start.

By default it starts a scene for it after the cursor's scene, with a placeholder heading (`INT. PLACE - DAY`, selected so you can type over it) and the link in place. If the story enters a new chapter there, the chapter's section comes first. With Preferences > Script > Adapting "Next Source Passage starts a scene for it" off, Next only shows the passage, and the pane offers Start a Scene.

If you press Next again while the passage it showed is still not linked, the pane asks whether to mark it as cut.

## Sections from the source

Adaptation > Source Headings as Sections writes the story's headings into the script as Fountain sections (`## Chapter 3`), at the same levels. If the script has none of them yet, they all go in as an outline at the cursor's line. Otherwise each missing one goes in its place among those already there. A heading is never written twice. One undo step.

## Suggestions while writing

With a source, Accolade offers the next piece of the story in the status bar as you write, as "Tab: …". Tab takes it, in one undo step; Tab again offers the next one; Esc puts it away for that line.

- **After a cue** (or on the empty line under it): that character's next line in the passage, with how they say it as a parenthetical.
- **Typing a cue:** the names of the passage's speakers first.
- **On an empty line:** the next piece not yet in the scene, a line of dialogue with its cue, or the narration as action.
- **Typing a scene heading:** INT. or EXT., the place and the time, as read from the passage.

The suggestions come from the passages the scene links to, else the one Next showed. On a phone, a small "Tab" button before the hint does the same as Tab. Preferences > Script > Adapting "Suggest from the source while writing" turns them off.

Accolade reads the story with word lists for English, Dutch, Portuguese, Spanish, French, German and Esperanto. It is a heuristic reader, with no AI: it gets most lines right, and some speakers wrong. It needs no network.

## What Accolade learned

As you adapt, Accolade learns from what you do, quietly, as part of your own undo step:

- **A name:** you rename a cue Tab put in (TOM to THOMAS), so the story's name becomes another name for that character.
- **Not a name:** you throw away a cue Tab put in twice (Monday, Lord), so the word is no character.
- **A place:** you write your own heading for a linked scene, so that place gets your heading from then on.
- **A skipped link:** a suggested link you skipped is not offered to that scene again.

It keeps them in the script's `/* Adaptation` block:

```
/* Adaptation
characters: ANNA = Anna, Annie, Miss Kovač
not characters: Monday, Lord
places: the house = INT. KOVAČ HOUSE
cut:
  #:~:text=It was late.
*/
```

Adaptation > What Accolade Learned… opens the list in the pane. Each lesson has a button that takes it away, and it is not learned again in that session. You can edit the block by hand too. Nothing is learned while you edit the script together with others.

See also: [Fountain Reference](fountain.md#source-links), [Markdown Syntax](cheat-markdown.md), [Accolade Blocks & Links](cheat-blocks.md).
