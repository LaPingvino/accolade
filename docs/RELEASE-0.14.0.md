# Accolade 0.14.0

## Scripts from other apps: Fade In, Trelby, KIT Scenarist

Wrote your script somewhere else? Open it in Accolade and keep writing, and save a copy back for the people who still use that app.

- **Fade In:** a `.fadein` file, zipped or as a folder, or an `.osf` (the Open Screenplay Format, versions 1.2 to 4): every element with its bold, italic and underline, dual dialogue, scene numbers, page breaks, notes, synopses and the title page.
- **Trelby:** a `.trelby` file, with its notes, its act breaks as sections and the title page.
- **KIT Scenarist:** a `.kitsp` project, with its emphasis, dual dialogue, scene numbers and colours, folders as sections, review comments as notes.

Every import opens as a new, unsaved script; Save As… (or File > Export…) writes a copy in the other app's format. Help > **Coming from Another App** has a page per app (Highland, Final Draft, WriterDuet and Celtx, Storyboarder too): what comes in, what goes back out, and what each app has no place for.

The same on the command line, any format to any, each file told by its name:

```
lexington old.fadein new.fountain
lexington script.fountain script.trelby
```

## Trelby keeps your emphasis and dual dialogue

Trelby's text has no styles and Trelby has no dual dialogue, so a copy for Trelby used to lose them. Now they are written as Fountain's own markers (`*italics*`, `**bold**`, `_underline_`, `MARY ^`), as Trelby's own Fountain import does: they show and print in Trelby as they are, and come back as bold, italics and dual dialogue when the file is opened again.

## Also new

- **Adapting across languages:** the adaptation benchmark now has a cross-language pair, an English play of *A Christmas Carol* against the French *Cantique de Noël*, so matching a script to a source in another language is measured, not guessed. The converter behind it takes a story alone, joins subtitles and broken hyphens, drops footnotes, and keeps a dialogue dash as a dash.
- **Tests no longer hang:** an intermittent hang in the window tests (a page-show hook that could block the UI) is gone.
