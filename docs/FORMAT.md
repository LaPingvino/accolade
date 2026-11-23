# The .accolade bundle: a rough description

> **Draft:** this describes what Accolade and Lexington do today. It is not a stable spec, and it will change.

## Transparent first

A bundle is made of plain files and nothing else: the script as plain Fountain or Markdown, JSON and TOML beside
it, and pictures as ordinary image files under `assets/`. Unzip one, or keep it in its folder form, and any text editor can read
your work and get it back. You don't need Accolade for that. What matters here is being able to recover your work,
more than having a fixed spec.

## Recovering a script by hand

1. Unzip the file (`unzip Film.accolade`, or rename it to `.zip` and open it). Skip this step for the folder form.
2. Open the folder inside it, `Film.textbundle/`.
3. `text.fountain` is the script, in plain Fountain that any Fountain app or text editor can open. For a script in
   another language, `text.<lang>.fountain` (e.g. `text.eo.fountain`) is the script as it was written.
4. The rest is plain text too:
   - each draft is in `assets/drafts/NN-name/text.fountain`, and `assets/drafts/drafts.json` lists their names
     and dates;
   - the source story is `assets/source.md`;
   - the production data is JSON in `assets/production/`;
   - the script's Lexington settings are TOML in `assets/lexington/`;
   - the pictures the script uses (storyboard frames, cast photos, a document's images) are ordinary image files
     under `assets/`.

To take out only the script, `lexington -i Film.accolade -o Film.fountain` writes it, or copy `text.fountain`
(`unzip -p Film.accolade '*text.fountain' > Film.fountain` does it in one step). Both give the file as Accolade
wrote it, Accolade's own blocks included.
`lexington check Film.accolade` tells you whether a bundle is whole and readable.

## How stable it is

The format is unlikely ever to be fully frozen. Elements that are already there stay supported, though:
Accolade keeps reading what it once wrote, and new things are added beside the old ones, not in their place.
**Other apps should read the text file and keep what they don't understand.**

## Two forms, the same files

A bundle is a [TextBundle](http://textbundle.org) folder. It is stored in one of two forms that hold exactly
the same files:

| Type | Zip form | Folder form |
|---|---|---|
| Accolade | `Name.accolade` (the default) | `Name.accolade/` |
| Highland 2 | `Name.highland` (the default, and the form Highland 2 is described as writing) | `Name.highland/` (whether Highland opens it is untested) |
| TextBundle | `Name.textpack` | `Name.textbundle/` (the default) |

- **Reading:** Lexington and Accolade tell the forms apart by what is on disk, a file or a folder, never by the
  name alone.
- **Layout in a zip:** the files sit inside one top-level `Name.textbundle/` folder, as TextBundle's own writer
  does, and as Highland 2 is described to (no file saved by Highland 2 has been tried yet; plain-text `.highland`
  files, which exist too, read as Fountain). Files at the zip's root are read too.
- **Writing:** a bundle keeps the form and layout it was read in. Save As (Accolade folder, TextBundle folder),
  Rename and `lexington -form zip|folder` switch it.
- **Why the folder keeps the `.accolade` name:** the folder form keeps the bundle's own name, `Name.accolade/`,
  rather than something like `Name.accolade.d`. Windows and Linux show it as an ordinary folder. It still says
  what it is, every tool that reads the extension (the language in `film.eo.accolade`, the type) works the same
  way for both forms, and zipping the folder gives back the file.

## What is inside

| Path | What it is | Whose |
|---|---|---|
| `text.fountain` | The script, standard Fountain | Everyone's: **read this** |
| `text.md` | The text of a Markdown document (instead of `text.fountain`) | Everyone's |
| `info.json` | TextBundle's metadata: `version` 2, `type` (`com.quoteunquoteapps.fountain` or `net.daringfireball.markdown`), `transient`, `creatorIdentifier` | Everyone's: the top-level keys |
| `info.json` → `"eu.kiefte.Accolade"` | Accolade's own key (below) | Accolade's |
| `info.json` → `"com.quoteunquoteapps.highland2"` | Highland's state | Highland's, kept as is |
| `text.<lang>.fountain` | A script in another language as written, e.g. `text.eo.fountain`. `text.fountain` beside it is the same script in standard Fountain, unless the script chose to leave it out | Accolade's and Lexington's |
| `assets/` | TextBundle's folder for files the text uses | Everyone's |
| `assets/source.md` | The story the script adapts (or the entry Accolade's key names) | Accolade's |
| `assets/drafts/drafts.json` | The drafts' index | Accolade's |
| `assets/drafts/01-first-draft/text.fountain` | One saved draft, with `source.md` beside it when the source changed | Accolade's |
| `assets/production/breakdown.json`, `schedule.json`, `revisions.json` | Production data: tags, the shooting schedule, the revisions | Accolade's |
| `assets/lexington/lexington.toml`, `assets/lexington/formats/<key>.toml` | The part of Lexington's settings the script uses (its language's words, its format), so it prints the same on any machine, in Accolade and with `lexington -i Name.accolade` (laid over that machine's settings; `-e` still picks another format). Only an `.accolade` carries these | Accolade's and Lexington's |
| `characters.json`, `sprints.json`, … | Highland's sidecars | Highland's, kept as is |
| anything else | Another app's files | Kept as is |

**Accolade's key** in `info.json` holds `version` (1) and, only when there is something to say, the following
fields. When nothing needs saying, the key is left out, so opening and saving a Highland file changes only its
text.

- `documentType`: `"markdown"` or `"screenplay"`, when the text file's name says otherwise.
- `source`: the entry that holds the source.
- `drafts`: the index.
- `language`: the language of `text.<lang>.fountain`.
- `fountainHash`: a hash of the `text.fountain` Accolade wrote. If it no longer matches, another app changed that
  file.
- `noFountainCopy`: the script keeps only its language's text.
- `format`: the script's format, when it is not the screenplay.
- `highlandWarned`: Accolade has said once that Highland may drop its files.

## What other apps may rely on

- **The text file:** `text.fountain`, else `text.md`, `text.markdown`, `text.txt`, at the bundle's root.
- **`info.json`'s standard top-level keys:** it may end with a trailing comma, as TextBundle's own samples do.
  Read it leniently.
- **Writing a bundle:** change the text file and your own reverse-DNS key in `info.json`, and copy every other
  entry as you found it. Accolade and Lexington write bundles that way. Zip entries are copied raw, and
  `info.json` is edited as text so that its other keys keep their order and bytes.

Everything under `assets/drafts/`, `assets/production/`, `assets/lexington/` and Accolade's key is Accolade's
own. Read it if it helps, but don't rely on its shape staying the same.
