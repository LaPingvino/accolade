# From screenplay to shoot: the shooting script and pre-production (research and plan)

Research, 2026-11-20, for Joop. The adaptation plan (docs/ADAPTATION.md) brings a story into a screenplay. This is
the other direction, from a finished screenplay to the day it is shot:

- lock the script;
- revise it in coloured sets;
- break it down;
- schedule it;
- make each day's sides and call sheet.

Nothing here is built. Sources for claims about other products are linked inline. Where something is marked
*unverified*, I looked for it and didn't find it.

## 1. What Accolade and Lexington already have for production

**Scene numbers**

- Fountain's `#12A#` is read by `lex.SceneNumber` (lexington lex/type.go:65-71) and printed in both margins of the
  PDF (pdf/create.go:128-159).
- DOCX and ODT put it at the right margin (office/docx.go:243, office/odt.go:246). FDX writes it as
  `Paragraph Number` (fdx/write.go:188), and the layout package has it as `Line.SceneNumber`.
- The PDF reader finds numbers in the margins (pdfin/classify.go:260-278).
- Accolade numbers the unnumbered scenes at export: Page Setup's "Number the scenes not numbered"
  (pagesetup.go; the Export dialog's until 2026-11-21). It counts on from the last explicit number, so after 7 comes 8. Since step 1
  (§7) this is Lexington's `lex.NumberScenes`, and Production > Lock Scene Numbers, A-scenes and OMITTED exist.

**Cards, outline and title page**

- View > Cards (cards.go) and View > Outline (outline.go, outline_move.go): drag to reorder, colours, folding.
  This is Accolade's counterpart to Final Draft's Beat Board and Story Map, and the base for a stripboard.
- Edit > Title Page… (title_page_dialog.go): where issued revision sets would be listed.

**Pagination**

- Lexington paginates as screenplay software does: keep-with-next, widows and orphans, and `(MORE)` / `NAME
  (CONT'D)` in the PDF. This is Lexington 1.6.0 (CHANGELOG) and Page Setup's page-break choice
  (export_pagination.go).
- Since step 5 (§11): the scene `CONTINUED:` / `(CONTINUED)`, locked pages and A-pages. Still missing: running
  headers other than the page number and the revision slug.

**Colours and tags**

- A note on a heading or section, `[[red]]` or `[[#3366ff]]` as Beat writes it, colours the outline, the cards and
  the export.
  - Accolade: Format > Scene or Section Color… (Ctrl+Shift+L, scene_colors.go).
  - Lexington: `-colors`, `-colorstyle`, `-colorscope` and patterns.
  - FDX keeps the colour as SceneProperties Color (fdx/colors.go).
- Any note on a heading or section is a tag. The Export dialog's "Only scenes and sections tagged" exports just
  those scenes (export_filters.go:18-60, 133-189; Lexington's `lex.FilterTags`, `-tag`).
- These are scene-level tags. Breakdown categories and tagged words came with step 7 (§13), kept outside the text.

**Sides**

- Export > "Sides for character" has three modes: full scenes, cue lines (the last N words of the cue) or lines only
  (export_filters.go:113-153; Lexington's `lex.Sides`, `-character -sides -cuewords`).
- The file name gets " - EDWARD sides" (export_filters.go:93).
- Sides by scene list or by shoot day are missing, and so are the revision header and the "sides" cover.

**Reports**

- View > Statistics… (details.go:20-248) counts:
  - words, pages and printed lines;
  - duration at a page a minute (all, action, dialogue);
  - scenes, interior and exterior;
  - scenes by heading start;
  - each role's speeches and words.
- The status bar shows "Scene 2 of 3 · 412 words · ~2 pages" (stats.go).
- Since step 2 (§8): Production > Reports… has the scene, location, character and cast reports, on screen, as
  CSV and as PDF. Breakdown sheets and the element report since step 7 (§13).

**Notes and collaboration**

- `[[notes]]` survive every writer. FDX `ScriptNote` is read and written (fdx/write.go:130-151).
- The notes block is `/* Notes */` (notes.go). The source, adaptation and notes blocks are boneyard at the end of
  the script.
- View > Changes (changes.go) is the history of a shared script (Edit Together), with who and when for each
  change. It is not a revision set.

**Export formats**

- Through Lexington (export_formats.go; cli.Formats): PDF, HTML, DOCX, ODT, FDX, Fountain, LaTeX and lex. With
  external tools: EPUB and the e-book formats, RTF and Markdown.
- FDX keeps the title page, dual dialogue, styles, scene numbers, scene colours and script notes.
- FDX writes `RevisionID="0"` on every text run (fdx/write.go:232) and reads `RevisionID` without using it
  (fdx/parse.go:64, 188).
- FDX has `<Revisions>` since step 4, `<LockedPages>` since step 5, and `<TagData>` and `<SceneProperties Length=…
  Page=…>` since step 7 (§13).

**Bundles**

- Accolade opens and saves .highland, .textpack and .textbundle (bundle.go, docs/HIGHLAND.md).
- It copies every unknown entry raw, keeps its own state under info.json's `"eu.kiefte.Accolade"` and keeps the
  adaptation source at `assets/source.md`. This is the natural container for production data (§4).

**Not handled: Beat's data block**

Accolade doesn't read Beat's data block, the trailing `/* If you're seeing this … BEAT: {…} END_BEAT */`, except its
tags (since step 7, §13: read once on open, while its Text Length matches). It stays in the text as boneyard. Its
offsets go stale as soon as Accolade edits the text (§3).

## 2. What the field does

### Production revisions

**Convention**

- The page colours: the WGA West's order is white, blue, pink, yellow, green, goldenrod, buff, salmon, cherry, then
  second blue … second cherry, with no second white (Final Draft's knowledge base, "Per the WGA west, the standard
  script revision color sets are: …", https://kb.finaldraft.com/hc/en-us/articles/15575314119316, seen as a search
  snippet: the page itself answers 403; Quote-Unquote's "The WGA script revision order",
  https://blog.quoteunquoteapps.com/script-revision-colors-how-to-track-your-screenplay-changes/). Productions differ:
  Wikipedia has tan and ivory after cherry, then "double white", "double blue"
  (https://en.wikipedia.org/wiki/Shooting_script), and Final Draft's own default sets in a real FDX end Cherry, Tan,
  2nd White, 2nd Blue. Accolade follows the WGA's (§10).
- Each changed line gets an asterisk in the right margin.
- Each revised page has a header slug with the date and the colour, and the title page lists the slugs issued
  (Wikipedia, above).

**Final Draft**

- Revision Mode, editable Revision Sets, "Show Page Colors", and printing revised pages only
  (https://kb.finaldraft.com/hc/en-us/articles/27813588722708-How-do-I-use-Revision-Mode,
  https://kb.finaldraft.com/hc/en-us/articles/27815245815060-How-do-I-select-modify-or-create-a-Revision-Set).
- Real FD files show:
  - `<Revisions ActiveSet= RevisionMode= ShowPageColor=>` with `<Revision ID Name Color PageColor Mark="*"
    FullRevision Style>` entries;
  - `RevisionID` on each `<Text>` run;
  - `<DeletedText><DeletedTextLocation Offset RevisionID>` for deletions;
  - a "Last Revised" DynamicLabel in `<HeaderAndFooter>`.
  - Sample: https://github.com/michaelforrest/ShootingChecklist/blob/HEAD/ShootingChecklist/Reference/EuroMechanical.fdx.
    FDX has no official spec (https://www.nationalarchives.gov.uk/PRONOM/fmt/1845).

**Fade In** (https://www.fadeinpro.com/page.pl?content=features)

- Revisions, page locking, scene locking, omitted scenes, coloured pages, tags, and reports as HTML and CSV.
- Its format is the **Open Screenplay Format (OSF)**, open and published with an SDK
  (https://github.com/OpenScreenplayFormat/osf-sdk, v4/OSF-4.0.xml).
  - Settings carry `pages_locked`, `pagenumber_mode="1AB"` and `revision`.
  - The scene-heading style carries `number_locked`, `number_mode` and `omitted_text="OMITTED"`.
  - Text carries `<text revision="2">`.
  - It also has `<revision_colors>` (White–Tan, mark `*`) and `<tag_categories>`, the same list as FDX.

**WriterDuet**

- Production revisions with presets "Blue [MM/DD/YY]…", asterisks, and text and page colours
  (https://www.writerduet.com/article/109-view-and-track-revisions-edits).
- Lock Pages: overflow goes to 2A, and a repeat lock gives 1AA (https://www.writerduet.com/article/112-lock-pages).
- Locked scene numbers give 2A, 2B (https://www.writerduet.com/article/162-enable-disable-line-numbering).
- Omit Scene leaves a grey OMITTED (https://www.writerduet.com/article/382-omit-scene).

**Highland**

- Says it leaves this out: "Locked pages and colored revisions are specialized production features that are not
  well-suited to Highland's plain text philosophy… you're generally better off jumping to Fade In or Final Draft";
  and A/B pages: "Not presently"
  (https://quoteunquoteapps.com/highland-pro/library-files/Switching-from-Final-Draft-to-Highland2.pdf).
- Highland 2.5 added a Revision Mode, coloured tracked changes with stars
  (https://quoteunquoteapps.com/highland-2/). I found no locked pages since.

**Beat** (open source, https://github.com/lmparppei/Beat)

- Eight generations: blue `*`, pink `**`, yellow `+`, green `++`, goldenrod `@`, buff `@@`, rose `#`, cherry
  `##` (Frameworks/BeatCore/…/Revisions/BeatRevisions.m:79-87).
- They are stored as character ranges in the trailing JSON block (§3).

**Fountain itself**

- Leaves revisions out on purpose: "Not included are production features such as MOREs, CONTINUEDs, revision
  marks, locked pages, or colored pages… For this reason, Fountain does support scene numbers"
  (https://github.com/nyousefi/Fountain/blob/master/README.markdown;
  https://johnaugust.com/?p=8070).
- A 2018 fountain-dev proposal for `{{if revision==green}}` went nowhere. Rankin preferred a diff built into the
  app (https://groups.google.com/g/fountain-dev/c/6Bt2155VpuI).

### Locked pages, A-pages, A-scenes, OMITTED

**Convention**

- Once the script circulates, page numbers are fixed. A page that grows continues on 45A, 45B.
- Scene numbers are fixed too. Wikipedia: a scene inserted between 10 and 11 is A11 in the US (10A would read
  as set-up A of scene 10 on an American slate) and 10A in the UK. The software says otherwise: Final Draft
  (American) gives inserted scenes "28A"
  (https://kb.finaldraft.com/hc/en-us/articles/27810301418132-How-do-I-number-scenes), WriterDuet 2A, 2B, and
  Movie Magic Screenwriter has A11 as an option, "Flipped - A1"
  (https://techsupport.screenplay.com/hc/en-us/articles/115001068631). So the labels are disputed. Accolade names
  the two by what they make (10A, A11), not by country, and defaults to 10A (§7).
- A cut scene keeps its number as "OMITTED" and can come back
  (https://en.wikipedia.org/wiki/Shooting_script).

**Who has it**

- Final Draft locks pages, has "unlock A pages" and "unlock all"
  (https://kb.finaldraft.com/hc/en-us/articles/27817054455828-How-do-I-lock-or-unlock-script-pages-and-use-the-locking-tools,
  https://kb.finaldraft.com/hc/en-us/articles/15573887116948-Final-Draft-Locked-Pages-and-Page-Numbering).
  - FDX: `<LockedPages><LockedPage PageNumber Position LockLevel LevelIndex>`, where Position is a character
    offset.
  - FDX: `<SceneNumberOptions NumberScheme="1A">`.
  - *Unverified:* how Final Draft itself writes an omitted scene or `Number="12A"`. Only third-party fixtures show
    it, for example OpenDraft's golden file
    https://github.com/Proteus-Technologies-Private-Limited/OpenDraft/blob/HEAD/frontend/src/utils/__snapshots__/exportGolden/locked-omitted-scenes.fdx,
    which writes OMITTED as the heading with its number kept.
- STARC (GPL, https://github.com/story-apps/starc) gives each scene `Number{value, isLocked, followNumber}`. A new
  scene takes an A-number from the last locked one (src/corelib/business_layer/model/text/text_model_group_item.h).
- Fountain has `#12A#`. A fountain-dev thread agreed in principle on forced page numbers, `===17`, `===84A`
  (https://groups.google.com/g/fountain-dev/c/W_cAivCNhIk/m/ELnwExyTBAAJ). Only that thread proposes it, and no
  reader I know of implements it.

### Breakdown (tagging)

- **Final Draft's Tagger:** in FDX, `<TagData><TagCategories>` holds about 30 fixed categories with UUIDs (Cast
  Members, Extras, Props, Wardrobe, Makeup, Vehicles, Animals, Stunts, SFX, VFX, Sound, Music, Set Dressing,
  Location, Script Day, Unit …). Then come `<TagDefinitions CatId Id Label Number>`, `<Tags>`, and
  `TagNumber` on `<Text>`.
  - Beat's FDX writer is the readable reference:
    https://github.com/lmparppei/Beat/blob/main/Frameworks/BeatFileExport/BeatFileExport/Export/Modules/Final%20Draft/BeatFDXExport.m.
  - OSF has the same list.
- **Movie Magic Scheduling** imports .fdx directly, but its own page says the import brings "the header (slugline) and
  the talking characters" only. Its tags-to-breakdown route, "adding categories it lacks", is for a tagged script
  exported as `.sex` (https://mms-docs.ep.com/Breakdown/ImportingBreakdownSheets.html, read 2026-11-20).
  - Its other route, `.sex`, is closed. Fade In: "a closed, proprietary format", so it doesn't write it
    (https://fadeinpro.com/kb/content/1/85/en/export-to-movie-magic-scheduling-or-movie-magic-budgeting.html).
  - Final Draft seems to say tags map fully only through .sex. This comes from a search snippet of
    https://kb.finaldraft.com/hc/en-us/articles/15575252743316-Can-Final-Draft-export-to-Movie-Magic-Scheduling-and-Budgeting;
    the page itself returned 403. Its snippet says it plainly: "although MMS can import a .fdx file, it cannot read
    the tag information". So a tagged FDX does not carry tags into MMS (§13).
- **StudioBinder** imports FDX, Fountain and PDF and tags in its own app
  (https://intercom.help/studiobinder/en/articles/7156315-importing-your-screenplay).
- **Yamdu** imports FDX, PDF and Fountain (https://yamdu.com/hu/blog/a-blog-article-heading/).
- **Gorilla** takes tagged FDX or .sex; I found no CSV import of its own.
- **Celtx** tags in its own app and makes breakdown and one-line schedule reports for distribution
  (https://support.celtx.com/hc/en-us/articles/360059004274). I found no documented import or export of breakdowns.
- **SetHero** imports CSV with a column mapping (https://help.sethero.com/en/articles/1513681-can-i-import-data-from-movie-magic-scheduling).
- **Filmustage** exports .sex, MMSX, XLSX and CSV
  (https://help.filmustage.com/en/articles/4802061-export-to-movie-magic-and-gorilla-software).
- **No published CSV layout:** each tool maps columns.
- **Page eighths:** each scene's length is counted in eighths of a page. FDX has `<SceneProperties
  Length="5 3/8" Page="1">`.

### Reports

- **Final Draft** has Scene, Location, Character, Cast and Script reports (Tools > Reports). Its Scene report
  lists number, intro, location, time, page and length, with tags
  (https://kb.finaldraft.com/hc/en-us/articles/27782286973716-How-do-I-create-reports-on-my-scenes-locations-characters-and-other-script-statistics).
- **Trelby** (open source) has Script, Location, Scene and Character reports and a Dialogue chart
  (https://github.com/trelby/trelby/blob/master/trelby/trelbyframe.py).
- **STARC** has cast, dialogue, gender, location, scene and summary reports.
- **WriterDuet** has reports and a locations/scenes report (https://www.writerduet.com/article/215-reports-overview).

### Scheduling, call sheets, DOOD, shot lists, storyboards

**StudioBinder**

- Script to stripboard: strips by cast, location, pages, script day, D/N and INT/EXT.
- Day breaks by page count, auto-sort, and duplicate schedules to compare.
- One-liner, DOOD, breakdown sheets, daily sides, and call sheets made from a day break.
- Source: https://www.studiobinder.com/film-scheduling-software.

**Open formats**

- **Call sheets:** none. Everyone sends PDFs.
- **Calendar:** StudioBinder has "no way to sync your shooting schedule with the Calendar"
  (https://intercom.help/studiobinder/en/articles/1408719-how-to-create-calendar-events). Celtx pushes shoot days to
  Google Calendar (https://support.celtx.com/hc/en-us/articles/360059004274). iCalendar (RFC 5545) is unclaimed but
  fits.
- **DOOD:** computed, as PDF or XLSX (https://help.filmustage.com/en/articles/8278194-view-and-export-the-dood-report).
  It is a grid of element × day with SW/W/H/WF codes.

**Shot lists and editorial**

- OpenTimelineIO (Apache-2.0, ASWF) has free metadata on every clip and marker. Its integrator's guide names
  "metadata from a screenplay (scene, location, characters)" as a use
  (https://github.com/AcademySoftwareFoundation/OpenTimelineIO/wiki/OpenTimelineIO-Application-Integrator's-Guide).
  Its adapters reach EDL, FCP XML, AAF and Premiere.
- Avid ALE (tab-separated, Scene and Take columns, custom columns allowed;
  https://pomfort.com/article/ale-avid-log-exchange-files-what-they-are-and-why-you-should-understand-them/) is the
  script-supervisor-to-edit log.
- ScriptE and Scriptation hand lined scripts to editorial as PDF only
  (https://help.scriptation.com/en/article/how-do-i-export-my-lining-as-a-pdf-for-my-editor-lbt324/).

**Storyboards**

- Storyboarder (Wonder Unit, https://github.com/wonderunit/storyboarder) opens Fountain.
- It rewrites each heading's scene number to `#<n>-<5-char id>#`. That is valid Fountain.
- It keeps its boards as plain-JSON `.storyboarder` files in `storyboards/Scene-<n>-…/` beside the script.
- It is not OSI-licensed. Its last release was in 2020, and the last push to it in March 2024.

## 3. Existing formats, and what Accolade should do with each

| Format | Covers | Open? | Read/write by | Accolade |
|---|---|---|---|---|
| Fountain `#12A#` | scene numbers | spec | everyone | **both**: the numbers stay in the text |
| Fountain `===` | page break | spec | everyone | both (already) |
| `===17A` | forced page number | proposal only | nobody known | no: other apps print it as action |
| Highland `{{HEADER: …}}` | running header, `%p` | Highland's docs (https://highland-kb.quoteunquoteapps.com/kb/fountain-syntax) | Highland | **read** later in Lexington. Writing it puts directives other apps print |
| Beat's trailing JSON block | revisions, tags, tag definitions, locked, scene numbering start, colour-coded pages (BeatDocumentSettings.m:26-76) | readable source, undocumented | Beat | **read on open** (import its ranges while `Text Length` matches the file, BeatDocumentSettings.m:72) and **keep on save**. Don't write: character offsets break under any other editor, which is why Accolade's anchors are words |
| FDX `<Revisions>`, `RevisionID`, `<DeletedText>`, `<LockedPages>`, `<SceneNumberOptions>`, `<TagData>`, `<SceneProperties Length Page>`, `<HeaderAndFooter>` "Last Revised" | all production data | undocumented XML, well reverse-engineered | Final Draft, Beat, Fade In, MMS, Gorilla, StudioBinder, Yamdu | **both**, the main exchange (Lexington) |
| OSF 4.0 | the same, in Fade In | **open, published** | Fade In | **both** (Lexington), second after FDX |
| .sex / .mmsx / .msd | MMS scheduling | closed | Final Draft, Filmustage, MMS | no. Tagged FDX instead |
| CSV breakdown/stripboard | per-scene rows | no fixed layout | SetHero, Filmustage, StudioBinder (export) | **write** a documented layout, read best effort |
| iCalendar (RFC 5545) | shoot days | standard | every calendar | **write** |
| vCard (RFC 6350) | cast and crew contacts | standard | every address book | **both**, for the call sheet's contacts |
| OpenTimelineIO `.otio` | shot list as a timeline, animatic | open | Resolve, Premiere, Avid via adapters | **write**, read later |
| Avid ALE | scene/take log | de-facto, plain text | Avid, Resolve, Silverstack | write, later |
| CMX3600 EDL | cuts | SMPTE 258M | everyone | no; through OTIO if wanted |
| Fountain issue #50 `[[SHOT …]]` | shot markers in the text (https://github.com/nyousefi/Fountain/issues/50) | proposal | nobody known | **read** as a shot's anchor in its scene. The shot's data (size, lens, board) goes in shots.otio: a note per shot would crowd the script other people read |
| `.storyboarder` | boards per scene | plain JSON, undocumented, not OSI | Storyboarder | **keep its `#n-xxxxx#` numbers intact**. Read its boards as a shot list later |
| Trelby `.trelby`, KIT `.kitsp`, STARC `.starc` | own scripts (SQLite for KIT and STARC) | source only | themselves | no: nothing to exchange that FDX or OSF doesn't carry |

## 4. Where production data lives

The bundle is the container. A `.highland` stays a Highland file when saved. **`.accolade`** is the same zipped
TextBundle with Accolade's production files added. Save As offers both, plus plain `.fountain`. Everything else is an
export.

**In the Fountain text**, readable by every app:

- Scene numbers `#12A#`, once locked.
- OMITTED scenes: `OMITTED #12#` as a forced heading (`.OMITTED #12#`). The old scene goes into a
  `/* Omitted 12 … */` boneyard right under it, so it can come back. Other apps print OMITTED with its number, as the
  convention wants.
- Notes, colours, the Notes, Source and Adaptation blocks, as now.
- A revision is never marked in the text. No asterisks in Fountain and no ranges: they would be noise to other
  apps, and they break.

**In the bundle**, under `assets/production/`, next to the source's `assets/source.md`:

| File | Holds | Format |
|---|---|---|
| `revisions/01-white.fountain`, `02-blue.fountain`, … | each issued draft's text, as sent out | Fountain, whole |
| `revisions.json` | the sets: name, colour, page colour, mark, date, issued or open; FDX `<Revision>`'s fields | JSON (own, after FDX and OSF) |
| ~~`pages.json`~~ | locked pages: built in the production block instead (§11), for every kind of file | — |
| `breakdown.json` | categories of one's own, the elements (by category name), each tag's scene and words as a Text Fragment directive, the scenes marked done (built: §13) | JSON (own; FDX/OSF categories) |
| `schedule.json` | shoot days (date, call, notes), strips in order (built: §14; banners and units later) | JSON (own) |
| `cast.vcf`, `crew.vcf` | contacts, with the role or department | vCard |
| `shots.otio` | shot list per scene, with boards as media references | OpenTimelineIO |
| `boards/` | images | PNG/JPEG |

**Why the own files.** No open format exists for a schedule, a breakdown or a lock. FDX's are offsets into its own
XML, and Beat's are offsets into the text. Accolade's own rule, from the adaptation links, is to anchor by words: a
tag is `{scene: "12", category: "Props", item: "revolver", at: "the revolver on the table"}`. An edit elsewhere doesn't
move it, and a broken anchor is shown as missing, the way the source pane shows it.

**Revisions are snapshots, not marks.** Issuing a set saves the text. A line's colour is the last set it changed in,
found by diffing the issued drafts in turn. This is Rankin's "diff tool built into the app", and it survives edits
in any other app between issues.

**info.json.** `"eu.kiefte.Accolade"` gets `production: {version, stage, activeSet}`. The stage drives §6's guide.

**Risk: Highland may drop unknown files when it saves.** This is untested. Highland's own page says it keeps
"revisions" metadata in subdirectories (https://quoteunquoteapps.com/highland-pro/highland-format), and assets/ is
the likeliest kept. Test with a real Highland: open an `.accolade` renamed `.highland`, edit, save, and list the zip.
Until then, Save as .highland warns that production data may not survive Highland.

**Flattening to plain `.fountain`.** What is in the text stays: the numbers, OMITTED and the blocks. The rest follows
the source's layers:

1. a copy Accolade keeps in its data folder, per script, following renames, as `sources` does;
2. or, on choice, a `Name.production/` folder beside the file with the same files. That folder is the bundle
   unzipped, and the browser downloads it as a zip.

Save As plain says which of the two it uses. A plain file opened again finds its data that way.

## 5. The gaps, ranked for an independent filmmaker or small team

Effort is rough: S is a day or two of ticks, M a week, L more.

1. **Locked scene numbers, A-scenes, OMITTED.**
   - What: number once, then lock. A new scene between 10 and 11 gets 10A or A11 (a preference; which country
     uses which is disputed, §2). Omit keeps the number. **Built: §7.**
   - Who has it: Final Draft, Fade In, WriterDuet, STARC.
   - Fountain: `#n#` in the text, the omitted scene in boneyard.
   - FDX: `Number` round-trips already. Add `<SceneNumberOptions>`, and write OMITTED as OpenDraft does.
     OSF's `number_locked`.
   - Effort: S. Where: the numbering moves from Accolade's export-time `numberScenes` (export_lexington.go:121) into
     Lexington, as `lex.NumberScenes` with locked and A-number rules, so its CLI, the FDX and OSF writers and
     Accolade share it. Accolade gets the menu items (built as Production > Lock Scene Numbers, Unlock Scene Numbers…, Omit Scene,
     Restore Scene: §7).
2. **Production revisions: issue, colours, asterisks, headers, revised pages only.**
   - What: Issue White (the first lock), then Blue and on, in the WGA order with "double" sets.
     - The PDF puts a `*` at the right margin of each changed line.
     - The header reads "Blue Rev. 11/20/26" with the page number.
     - The title page lists the issued sets.
     - Export revised pages only, optionally on coloured paper.
   - Who has it: Final Draft, Fade In, WriterDuet, Beat, Highland (marks only).
   - Representation: snapshots and revisions.json (§4).
   - FDX: write `<Revisions>` and `RevisionID` per run from the diff, and `<DeletedText>`. Reading is lossy:
     `<DeletedTextLocation>` records where text went, not the text, so earlier drafts can't be rebuilt. An FDX that
     arrives with Blue and Pink marks keeps them as imported marks, one set per `RevisionID`, until the next
     Issue. From then on the snapshots take over.
   - Effort: M. Where: Lexington (marks, header, revised-pages filter, colour band, FDX); Accolade (Issue, the
     diff, View > Revisions showing colour bars in the margin as the source pane's marks).
3. **Locked pages and A-pages.**
   - What: after lock, page breaks stay. Overflow goes to 12A, a shrunk page gets "12-13" or a blank, and the
     scene's `CONTINUED:` and `(CONTINUED)` are added.
   - Who has it: Final Draft, Fade In, WriterDuet.
   - Representation: pages.json by words.
   - FDX: `<LockedPages>`, its Position computed when writing. OSF: `pages_locked`, `page_number`.
   - Effort: M–L, the hardest part being Lexington's paginator laying out to anchors. Where: mostly Lexington
     (`rules.Pagination` with locks), Accolade for Lock Pages and Unlock. **Built: §11.**
4. **Breakdown tagging.**
   - What: select words and tag them in a category (Cast fills itself from the cues, Location from the heading).
     Each scene gets its eighths, D/N, INT/EXT and script day, and a breakdown sheet.
   - Who has it: Final Draft Tagger, Fade In, WriterDuet, Beat, StudioBinder, Celtx.
   - Representation: breakdown.json by words.
   - FDX: write `<TagData>` and `TagNumber` as Beat does, and `<SceneProperties Length Page>`. That is the route
     into Movie Magic, Gorilla, StudioBinder and Yamdu. Read FDX and Beat tags.
   - Effort: M. Where: Accolade (a Breakdown pane, the editor's tint for tagged words); Lexington (eighths from the
     layout, FDX TagData). **Built: §13.**
5. **Reports.**
   - What:
     - scene report (number, heading, page, eighths, D/N, cast);
     - location report;
     - character report (their scenes, lines, monologues);
     - cast report;
     - breakdown sheets per scene and element report per category;
     - all as PDF, CSV and HTML.
   - Who has it: Final Draft, Fade In, Trelby, STARC, WriterDuet.
   - Representation: computed, nothing stored.
   - Effort: S–M. Where: Lexington (a report package, as `layout`); Accolade (Statistics grows into View > Reports).
     The basic reports need no tagging and come early (Order step 2); breakdown sheets follow tagging. **Basic
     reports built: §8.**
6. **Stripboard and shooting schedule.**
   - What: strips, one per scene, coloured INT/EXT × D/N. Drag them into days and add day breaks with totals in
     eighths. Sort by location, cast or D/N. Make a one-liner and a shooting schedule. DOOD comes from it.
   - Who has it: MMS, StudioBinder, Gorilla, Celtx, Yamdu.
   - Representation: schedule.json.
   - Exports: iCalendar per day, CSV per strip.
   - FDX: none. FDX carries tags, not schedules.
   - Effort: M–L. Where: Accolade (a Schedule view like the cards), Lexington (one-liner, DOOD and schedule
     PDFs). **Built: §14.**
7. **Day packs: sides and call sheets.**
   - What: for a shoot day, the day's scenes as sides (Lexington has `lex.Sides`; add by scene list, two-up, the
     revision header, a cover) and a call sheet. The call sheet has date, call times, location, scenes, cast with
     their numbers, contacts from vCard, the advance schedule and weather left blank.
   - Who has it: StudioBinder, SetHero, Celtx, Yamdu.
   - Exports: PDF and .ics.
   - Effort: M. Where: Lexington (sides by scene list, call sheet PDF), Accolade (the day's pack in one export).
8. **FDX and OSF round-trip of all of this.** This is the exchange each step above needs: it lets the team hand the
   script to a production office using Final Draft or MMS, and take it back.
   - Read and write FDX's revisions, locks, tags and scene properties.
   - Add an OSF reader and writer.
   - Effort: M, spread over steps 1–4. Where: Lexington.
9. **Shot list and storyboards.**
   - What: shots per scene (number, size, angle, movement, lens, notes, board image).
   - Who has it: StudioBinder, Celtx, Storyboarder.
   - Representation: shots.otio.
   - Also: keep Storyboarder's scene ids.
   - Effort: M. Where: Accolade.
10. **On set and after.**
    - Lined script, script supervisor notes, ALE out.
    - Who has it: ScriptE, Scriptation.
    - Effort: L. Later, and only if wanted.

## 6. The guided workflow

The adaptation workflow's strength is one pane that says where you are and what is next: "50% adapted · 1 cut · 3 to
go", and Next. Production gets the same.

The **Production pane** (View > Production, a tab in the side dock) shows the stage and one count per stage:

1. **Lock.** A check before the lock lists the scenes without a heading time, duplicate numbers and an unsaved
   title page. Then Number and Lock Scenes, and Lock Pages. Issuing White is the lock. From then on the pane says
   "Locked 11/20 · 112 pages · 87 scenes".
2. **Revise.** Writing goes on as usual. The pane shows "Blue (open) · 6 pages changed · scenes 12, A14, 30". Issue
   Blue asks for the date and gives the export of revised pages and the full script. The margin shows each line's
   colour.
3. **Break down.** "62% broken down · 33 scenes to go". **Next scene** goes to the first scene not marked done, the
   way Next passage does, with:
   - Cast filled from its cues and the location from its heading;
   - the eighths shown;
   - suggestions for the rest, from words found in earlier scenes' tags.

   Done marks it done. **Built: §13.**
4. **Schedule.** "18 of 87 scenes on a day · 3 days". The stripboard, with the unscheduled strips at the end and
   Next Day. **Built: §14** (Add Day for Next Day).
5. **Day pack.** Per day: sides, call sheet and .ics, as one export. FDX or CSV goes out to an office using other
   software.

Each stage can be returned to. A revision after scheduling marks the changed scenes on the stripboard and in the
breakdown as "changed in Blue: check", which the stages before can't show.

## Order

1. **Lock scene numbers:** A-scenes, OMITTED and Restore. Accolade's `numberScenes` moves into Lexington as
   `lex.NumberScenes`, with A-numbers. In the text only, so nothing waits on the bundle.
2. **Basic reports:** scene (with eighths, D/N), location, character and cast, as PDF and CSV. They store nothing,
   need no bundle, and pay off at once. Built as Production > Reports… (§8); View > Statistics stays as it is.
3. **The `.accolade` bundle and the production files' home:** `assets/production/`, the flattening layers and the
   Highland warning, with the drafts store (LOOP.md, Joop 2026-11-20). Built (§9); still to test with a real
   Highland.
4. **Issue and revision sets:** snapshots, the diff, margin colours, Lexington's asterisks, headers and revised pages
   only. FDX `<Revisions>` and `RevisionID` out, and in as marks. Built (§10).
5. **Locked pages and A-pages** in Lexington's paginator, with `CONTINUED`. FDX `<LockedPages>`. Built (§11).
6. **The Production pane** with Lock and Revise, so the workflow stands before breakdown starts. Built (§12).
7. **Breakdown:** the categories, tagging by words, Next scene, eighths, breakdown sheets and element reports. FDX
   `<TagData>` and `<SceneProperties>` out (test against MMS), Beat's and FDX's tags in. Built (§13).
8. **Schedule:** the stripboard (built on the cards), days, one-liner, DOOD, .ics and CSV. Built (§14).
9. **Day packs:** sides by scene list and the call sheet with vCard contacts.
10. **OSF** reader and writer.
11. **Shot list** in OTIO, keeping Storyboarder's ids and reading `[[SHOT]]` markers.
12. Later, and only if wanted: the lined script and ALE.

Open questions for Joop:

- A-numbering by default: answered (Joop): 10A, with a setting for A11. He called 10A "US"; Wikipedia calls it UK
  (above). If he meant A11, the default is one line (settings.go `scene-number-style`).
- Should `.accolade` be the default Save for a script that has production data, or should `.highland` stay the
  default with the warning? Answered (on Joop's earlier choice): `.accolade` is what Save As offers once a script
  has production data (drafts, a lock; later the sidecars). Saving such a script as `.highland` keeps working,
  and warns once that Highland may drop Accolade's files. A plain `.fountain`/`.md` save flattens: the extras go
  to the kept copy, or on choice to `Name.production/` beside the file (desktop only). Details in §9.

## 7. Step 1, built: locked scene numbers, A-scenes, OMITTED (2026-11-20)

Branch scenelock (Accolade), Lexington v1.7.0 (tagged locally).

**Lexington**

- `lex.NumberScenes(s, lex.NumberOptions{Locked, Style})`, with `lex.NumberLabels` (the same on a list of numbers,
  what Accolade uses on its editor's lines) and `lex.SceneNumberBetween`. Unlocked it counts on from the last whole
  number, as Accolade's export did. Locked, it keeps every number and gives a new scene one between its neighbours,
  never one that is taken anywhere in the script (an OMITTED scene keeps its number):
  - `lex.NumberAfter` (default, "10A"): 10, 10A, 10B, 11; between 10A and 10B, 10AA; between 10 and 10A, 10aA (a
    small a before the letter: Wikipedia's form, since nothing comes between 10 and 10A). Past Z: 10ZA.
  - `lex.NumberBefore` ("A11"): 10, A11, B11, 11; between 10 and A11, AA11; between A11 and B11, AB11 (Wikipedia's
    examples).
  - Both: before the first numbered scene A1 (then B1); after the last, the next number (88 after 87), as Final Draft
    letters only scenes "inserted between previously numbered scene headings". A label out of the scheme (Fountain's
    `I-1-A`, `1.1`, Storyboarder's `3-x2k9q`; since v1.14.0 exactly defined: §15, *Scene numbers are labels*) is
    no neighbour, but is never given again.
- `lex.WithSceneNumber` puts `#n#` in a heading before its notes. `-number count|10A|A11` on the command line.
- `fountain.OmitScene` / `fountain.RestoreScene` (and `lex.OmittedNumber`): `.OMITTED #12#`, a blank line, then
  `/* Omitted 12`, the scene exactly as it was (heading with its number and notes included), `*/`. A `*/` inside is
  kept as `* /`, as the notes block does (so a literal `* /` in an omitted scene comes back as `*/`).
- FDX: the number stays `Number="12A"`; OMITTED is a Scene Heading `<Text>OMITTED</Text>` with `Number="12"`, as
  OpenDraft's golden file writes it (*unverified* against Final Draft itself). It reads back as `.OMITTED #12#`. No
  `<SceneNumberOptions>` is written: I have seen no real file's values. Fixed on the way: FDX wrote Fountain's
  boneyard (notes blocks too) as General paragraphs, which Final Draft prints; it is left out now. So an omitted
  scene's text does not survive FDX: Restore needs the Fountain (or the bundle).

**Accolade** (production.go; a Production menu, hidden for Markdown; the pane came with step 6, §12)

- **Lock Scene Numbers** writes `#n#` into every heading without one (locked numbering, so numbers already there
  are kept and gaps get letters) and the lock into a boneyard block at the script's end, with the source and
  adaptation blocks (read in any order, before the notes):

  ```
  /* Production
  scene numbers: locked, 10A
  */
  ```

  `10A` / `A11` is the lettering the script was locked with, so a locked script keeps its style on every machine;
  Preferences > Script > Production > "A new scene's number" (`scene-number-style`, default 10A) is for new locks.
  Other lines in the block are kept (later steps add theirs). One undo step; the cursor stays.
- **New scenes while locked** get their number once the cursor has left the heading and typing pauses (700 ms; on
  every text and cursor change a debounce, never per key), as part of the writer's last undo step (`buffer.Amend`,
  as the learned notes): Undo takes the heading and its number together. Not while Redo is possible, text is
  selected, before any edit (a script written elsewhere is numbered at the first edit), or while shared (each peer
  would number it). Export and print number a locked script's unnumbered scene the same way, always.
- **Unlock Scene Numbers…** asks, then takes the lock line out (the block too if empty). Since §15 it asks Keep the
  numbers / Remove the numbers, and Lock records which numbers were there before it, so Remove takes out only
  Lock's.
- **Omit Scene** (cursor in a numbered scene): the heading becomes `.OMITTED #12#` with the scene in a boneyard under
  it. Unnumbered: it says to lock first. **Restore Scene** (cursor on the OMITTED heading) puts it back exactly. One
  undo step each. The outline and cards skip boneyard, so the old heading is no scene there.
- Text typed after a heading's number (`INT. SHED #1A# - DAY`) leaves it mid-line, where Fountain doesn't read it;
  once the cursor has left, it is moved back to the end, never replaced by a new one.
- The learned notes key the headings they knew without their numbers, so Lock teaches them no places.
- Rough: the editor shows the omitted scene's boneyard as plain text (boneyard has no colour of its own yet); the
  outline doesn't show numbers.

## 8. Step 2, built: the basic reports (2026-11-20)

Branch reports (Accolade), Lexington v1.8.0 (tagged locally, not pushed; Accolade's go.mod has `replace
github.com/LaPingvino/lexington => ../lexington` until it is).

**Lexington**

- `pdf.PDFWriter.Measure` lays the script out as the PDF prints it (rules, paper, pagination; nothing written) and
  gives each scene heading its page (numbered as printed: the first after the title page is 1) and its length. A
  scene runs from the top of its heading to the top of the next heading, the last to the foot of the script's last
  line. The room the pagination leaves at a page's foot (a heading kept with what follows it) is the scene's before
  it, as a ruler on the printed page measures it; the rest of a page after a forced `===` (an act's end) is no one's,
  as the scene's lines are what count. A page is the lines that fit on it (55 on Letter), so a full page of scenes
  adds up to 8/8 before rounding.
- `lex.EighthsOf` / `lex.Eighths`: the eighths rule (below). Written as productions write it: `3/8`, `1`, `1 3/8`,
  never reduced (`4/8`, not `1/2`; https://anonymousproductionassistant.com/2016/08/29/long-will-scene-take/: "A
  scene isn't 2 1/2 pages; it's 2 4/8").
- The `report` package: `report.Make(screenplay, Options{Scenes, Aliases, PDF})` gives the scenes, locations and
  characters; `Report.Cast()` the cast list; `Report.Table(kind)` each as rows of cells; `Table.WriteCSV`,
  `report.WriteHTML`. `cli.Report` measures with the export's settings; `lexington -report scenes|locations|
  characters|cast` writes the CSV.

**The eighths rule.** A page's text area is eight equal eighths (the "8 one-inch rows" of
https://howtofilmschool.com/script-breakdown). A scene's length is rounded to the nearest eighth, half an eighth up,
and is never less than 1/8 (Scriptation: "Every scene should be at least 1/8",
https://help.scriptation.com/en/article/how-do-i-add-page-counts-to-my-script-3jt60r/). *Unverified:* the rounding
to the nearest. I found no source that gives a rounding rule (Filmustage, Scriptation, the AD blogs all eyeball it
with a ruler); nearest is my choice. The minimum means a page of many short scenes counts more than 8/8, as on real
breakdowns. Totals (a location's, a character's) are the sum of their scenes' rounded eighths. An OMITTED scene has
no length.

**What the reports read**

- Scenes: the heading as printed, split into the place word as written (`INT`, `EXT`, `INT./EXT`, in the script's
  language: the language's scene words), the location (up to the last ` - `, `–` or `—`) and the time (after it).
  A forced heading (`.THE VOID`) has no place word. Every scene has a number: a locked script's as Export prints
  them (a new scene gets its letter), an unlocked one's counted on as Export's "Number the scenes" does.
- Characters are those who speak: a cue's name without its extension (`(V.O.)`, `(CONT'D)`), `@` and `^`. The
  adaptation block's `characters: ANNA = Annie` line merges ANNIE into ANNA. A character is "in" the scenes they
  speak in; their eighths are those scenes' (a silent presence needs tagging, step 7). Cues in boneyard (an omitted
  scene's, the source's, the blocks) count for nothing.
- Cast numbers: by size, which later steps reuse (`Report.Cast`): the scenes they speak in, then their eighths, their
  speeches, their words; a tie goes to who speaks first, then by name.

**Accolade** (reports.go): Production > Reports… makes them in the background with the export's settings (script
format, paper, pagination, title page) and shows one at a time: Report (Scenes, Locations, Characters, Cast, the last
one remembered), Sort by (the report's own order, or a column; ↓/↑ reverses). On a desktop a table, its headers
sorting too; on a phone (under 600 px) each row a card: its first columns in bold, the rest as "Title value". CSV…
saves the report as sorted (downloaded in the browser); PDF… prints it as a web page through Chromium, or Print…
opens the print dialog (Save as PDF) where there is none or in the browser.

**CSV layout** (stable; golden files in Lexington's report/testdata). RFC 4180: UTF-8, comma, quotes where needed,
CRLF. One header row of these keys, then a row each. A cell with several values joins them with `; `. Lengths come
twice: `eighths`, a whole number (11), and `length`, as written (1 3/8: a spreadsheet may read `3/8` as a date, so
use `eighths` to compute). Empty means none.

| Report | Columns, in order |
|---|---|
| scenes | `order` (1, 2, … in the script), `number`, `heading` (as printed, without number and notes), `int_ext`, `location`, `time`, `page`, `eighths`, `length`, `characters` (who speaks, in order), `omitted` (`yes` or empty; an omitted row has `heading` OMITTED, `eighths` 0 and nothing else but its number and page) |
| locations | `location`, `int_ext`, `times`, `scene_count`, `scenes`, `eighths`, `length` (in order of first appearance; omitted scenes left out) |
| characters | `name`, `cast_number`, `scene_count`, `scenes`, `first_scene`, `first_page`, `last_scene`, `last_page`, `speeches`, `words`, `eighths`, `length` (in order of first speech) |
| cast | `cast_number`, `name`, `scene_count`, `eighths`, `length`, `speeches`, `words`, `first_scene`, `first_page` |

The order of rows is the report's own in Lexington's CLI; Accolade writes them as sorted on screen.

Rough: the summary names the paper only when Export's page size is set (else "the script format's paper"); an
unlocked script whose new scene counts on into a number an OMITTED scene keeps gets that number twice
(Export's count numbering does the same; lock the numbers first); Lexington's PDF prints no page numbers without a
title page (measured pages still count from 1); the PDF of a report was checked as HTML only here (Chromium can't
run in the agent's sandbox).

## 9. Step 3, built: the .accolade bundle, the extras' homes and the drafts (2026-11-20)

Branch bundle3 (Accolade only).

**The format.** `.accolade` is the same zipped TextBundle as `.highland` (a new one in the `Name.textbundle/` folder
layout, the text in `text.fountain`, or `text.md` for a Markdown document; info.json's `type` as for Highland's).
It is a screenplay by default, as a `.highland` is; unlike a `.highland` it is never read as plain text (a `.accolade`
that is no zip does not open). Open, Save, Save As, Open Recent and the browser's picker take it. Every entry Accolade
doesn't write is copied through byte for byte, as for any bundle. Accolade's key in info.json gains `drafts`
(`assets/drafts/drafts.json`, when there are drafts) and `highlandWarned`.

**Accolade's extras** are the files under `assets/drafts/` and (from step 4 on) `assets/production/`, named as in
the bundle wherever they are (extras.go). Three homes:

1. in the bundle (`.accolade`, and also `.highland`, `.textpack`, `.textbundle`);
2. for a plain script, a copy Accolade keeps in its data folder (`~/.local/share/accolade/production/<name>-<id>/`;
   the browser's kept settings folder), remembered per script in settings.json `production-files`, following a
   rename and Save As as the source's copy does; an unsaved script's is let go of by New and Open;
3. on choice, `Name.production/` beside `Name.fountain` (desktop only): the bundle's assets unzipped.

A plain script looks in that order: the kept copy remembered for its path, else the folder beside it. A plain script
saved as a bundle (Save As `.accolade`, a rename to `.highland`) carries its extras in; the kept copy is let go of
only for a script never saved (else it stays remembered for the old path).

**Saving** (saveas.go):

- Save As offers `Name.accolade` for a screenplay with production data (drafts, or a production block with a
  lock) that would be saved plain; a `.highland` or `.textpack` keeps its kind. Save and auto-save of a file with
  a name keep writing it as it is: a `.fountain` stays a `.fountain`, its drafts in the kept copy or the folder.
- Save As `.fountain` (or `.md`) of a bundle with drafts asks where they go: kept by Accolade (the default), in
  `Name.production/` beside the file (desktop only), or left out, which asks again (the bundle on disk keeps
  them). The status bar says which. The last choice for a name wins: an earlier kept copy for it is let go of.
- The browser's Save As now asks for a name (the download's), so it can change the kind too.
- **The Highland warning:** saving a `.highland` that holds Accolade's files (drafts, the source) — or saving the
  first draft into one — saves, then says once that Highland may drop them and that a copy as `.accolade` keeps
  them safe. `highlandWarned` in the key makes it once per file (and if Highland drops the key, it warns again,
  which is right). Untested with a real Highland: that is the test still to do.

**Bundle writes in the background** (bundle_queue.go, PERF-SOURCE item 7): every write into a bundle (Save,
auto-save, the source's Done, Save Draft) is a job, run one after another in a goroutine, each from the bundle the
job before left; the zip loop calls `pace.Yield` for the browser's page. A failed save makes the script unsaved
again and says why. Rename and closing the window wait for the writes (then ask about unsaved changes as ever);
quitting and `:wq` finish them at once first; the browser's download follows the write.
The per-keystroke path is unchanged: production data is looked for only when saving.

**The drafts** (drafts.go, drafts_view.go):

```
assets/drafts/drafts.json
assets/drafts/01-first-draft/text.fountain
assets/drafts/02-for-anna/text.fountain
assets/drafts/02-for-anna/source.md
```

`drafts.json` is `{"version": 1, "drafts": [{"id": "02-for-anna", "name": "For Anna", "date": "…Z", "kind":
"draft", "text": "assets/drafts/02-for-anna/text.fountain", "source": "assets/drafts/02-for-anna/source.md"}]}`,
oldest first; `kind` is `draft` or `issue`, and `colour` is there for step 4's issues. Each draft names its text and
source by their files, so a text or a source the same as one stored already is not stored again (the new draft
names the stored one), and a draft whose text and source are both stored has no folder. The source is kept when
it changed (a source kept in the script itself is in the text). The folder's name is the draft's number and its
name in ASCII letters and digits (`03-uber-cafe` for "Über Café").

- **Collaboration > Save Draft…:** a name (default "Draft n"), at any time, before a lock too; says where they are kept.
- **Collaboration > Drafts…:** the list, newest first (name; date, "same text as …", "source changed"), Compare,
  Restore…, Save Draft….
- **Compare:** any two drafts, or one with the script now; the script, or the source when there is one. By scene
  (internal/scriptdiff: elements are the paragraphs between blank lines, Myers' diff with the common start and end
  taken off, bounded at 2,000 edits; a removed and an added element sharing half their words are one changed
  element, compared word by word). A changed scene shows its heading ("(cut)", "(new)" for a whole scene), its
  changed elements (cut struck through in the error colour, added underlined in the success colour) and counts
  the unchanged; unchanged scenes are counted between. Side by side on a desktop (a setting), one under the other
  under 600 px. Computed in the background. Read only: taking single changes over is the later partial merge.
- **Restore…** asks, keeps the script as it is now as "Before restoring X" (unless a draft holds it already), puts
  X's text in the editor in one undo step, and adds "X (restored)". The source is not restored (compare it, or
  Save Source As from the draft is a later item).

Rough: Compare renders up to 400 changed elements as plain widgets (enough for drafts of one script; two
unrelated scripts get a note); the list's dates are local time; a draft saved while a bundle save is under way
shows in Drafts… once the write is done (Drafts… waits for it); the kept copy of a `.fountain` saved as
`.accolade` stays remembered for the old path rather than being deleted (after a rename it is left over).

## 10. Step 4, built: issues and revision sets (2026-11-20)

Branch revisions (Accolade), Lexington v1.9.0 (tagged locally, not pushed; Accolade's go.mod has `replace
github.com/LaPingvino/lexington => ../lexington` until it is).

**The colours.** The WGA West's order, as two sources give it (§2): White, Blue, Pink, Yellow, Green, Goldenrod,
Buff, Salmon, Cherry, then Second Blue … Second Cherry, Third Blue (`lex.RevisionName`). Not "double white": none of
the WGA-attributed lists has a second white (Wikipedia's tan/ivory/double white is one production's way, and Final
Draft's own defaults another). A file's own set names (an FDX's) are kept.

**Issue Draft…** (Production menu; revisions.go, revisions_view.go) asks for the date (today by default) and keeps
the script as it is now in the drafts store as kind `issue`, `colour` the set ("Blue"), `issued` the date
("2026-11-20"), next to the drafts (§9). The first issue is White, the production draft: it locks the scene numbers
first (Lock Scene Numbers) if they are not. The production block records the current revision:

```
/* Production
scene numbers: locked, 10A
revision: Blue, 2026-11-20
*/
```

**Taking an issue back** (issue_undo.go; LOOP.md, Joop 2026-11-20: "Issue White / Issue Blue cannot be undone").
An issue is one undo step: the scene lock the first issue makes, the page lock it writes after laying out the pages
(in the background) and the revision line are joined into one (`Buffer.JoinSince`), and the step is tagged
(`Buffer.Tag`). Typing while the pages are laid out stops the issue ("the script changed while its pages were laid
out: issue it again"), as Lock Pages does, so the joined step never holds the writer's own typing. The editor asks
its owner before a tagged step is undone or redone, whichever way in (Edit > Undo, Ctrl+Z in the editor, vim's `u`):

- *Undo* takes the text back and the issue's entry out of the drafts store: drafts.json without it, and the files
  only it named (a draft saved after it with the same text names the same file, which stays). *Redo* writes the text
  again and puts the entry back, last, with its id and files as they were (a new id and folder if a draft took them
  since).
- The entry taken out is kept in memory, in the step's record, with its files' bytes: the session's "withdrawn"
  list, as long as the undo history lasts (another document forgets it).
- While the store write is under way (a bundle's, in the background), Undo waits: the status line says to try again
  in a moment. If the issue is no longer the newest in the store, or not there, the text is taken back and the store
  left alone, as the status line says.

**Production > Withdraw Last Issue…** (and the Revise group of the ribbon's Production tab) does the same later,
for the newest issue only: an older one has a later issue built on it (its marks, its pages). The issue's entry
notes what it made, `made` in drafts.json:

```
{"id": "02-blue", "name": "Blue", "kind": "issue", "colour": "Blue", "issued": "2026-11-20", …,
 "made": {"revision": "White, 2026-11-01", "pageLock": true, "pagesBefore": "pages: locked, letter, …"}}
```

The withdrawal takes back only what is still as the issue left it, in one undo step: the revision line, if it is
still the issue's, goes back to the one before (or goes); the page lock, if the issue wrote it and it is still the
issue's, goes back to the lock before (or goes); the scene lock, if the issue made it, goes the way Unlock's *Remove
the Numbers* goes (§15: the numbers the lock wrote go, typed ones stay). The dialog lists what goes and what stays
(a lock changed since stays), and warns that a PDF or FDX that went out is not called back: those copies say its
colour, and the next issue takes the colour again. Accolade keeps no record of exports, so the warning is always
there. Undo issues it again, Redo withdraws it again (the same store moves as above). An issue kept before `made`
existed has its entry and revision line taken back only (the dialog says so). An FDX's imported issue
(revisions.json) cannot be withdrawn: it is the imported script's own record.

Rough: a store write that fails after the text moved leaves them apart (an error dialog says so); a withdrawal that
finds nothing in the block to take back changes the store only, and Undo then cannot put it back.

**The marks.** Never in the text. Each issue's body (the text before the blocks at its end) is diffed with the one
before, line by line, blank lines left out (`scriptdiff.Lines`, the same bounded Myers as Compare): a line the same
keeps the mark it had, a new or changed line gets the issue's set, a deletion marks the line after it (none after
it: the last line), unless a changed line is there already. The script now, if it differs from the last issue, is
the open set, in the next colour, dated today. Preferences > Script > Production > "Revision marks since":

- the last issue (the default): the current revision's marks only, the open set's, or, right after an issue, that
  issue's against the one before (what its pages print);
- all issues: each line in the colour of the last set it changed in.

The editor shows them as a bar in the fold marks' column, in the set's colour (`lex.RevisionInk`), computed in the
background 400 ms after typing pauses; the issues are read again for another script or a new issue only.

**Printing** (Lexington's PDF; the export's and File > Preview PDF's when "Revision marks" is on, the default once
the script has an issue): an asterisk at 7.75" (Final Draft's `<Revisions Location>`; right of a numbered
heading's number) on every printed line of a marked row, and on each changed page a header on the page number's
line, ending before it: Preferences' "Revised page header", `{set} Revision – {date}` by default ("Blue Revision –
20/11/26"; the date 20/11/26, 11/20/26, 2026-11-20 or 20.11.2026; `-` none), the latest set on the page. Export >
PDF also has "Revised pages only" (the pages with a mark, numbered as in the whole script; the others are left out
of the PDF's page tree; bookmarks are left out then) and "Changed pages on their colour's paper" (off by default; a
tint of Final Draft's paper colours). Accolade maps its lines to Lexington's rows by `fountain.ParseLines` (each
row's source line), so the marks are those the editor shows. File > Print goes through HTML and prints no marks:
use the PDF.

**FDX out:** `<Revisions>` with a `<Revision>` per set after White (IDs 1 on, Name "Blue Rev. 20/11/26", Final
Draft's PageColor, Mark `*`, Location 7.75, ActiveSet the last) and `RevisionID` on a marked paragraph's runs:
every issue's marks, whatever the margin shows (an exchange file; the program reading it shows what it wants). No
`<DeletedText>`: its offsets are unverified beyond one file.

**FDX in:** Lexington reads each paragraph's set from its runs' `RevisionID` (the latest), or from a
`<DeletedTextLocation>` within it: the offset counts the runs' characters plus one per paragraph end, which is how
the one real sample's offsets fall (*unverified* beyond it). Only the sets the text uses are kept; "Blue Rev.
(mm/dd/yy)" reads as Blue. Accolade then keeps the imported text as an issue of the latest set ("Pink (imported)"),
writes the block's line (`revision: Pink (imported from X.fdx)`) and the marks, by line of that issue's text (which
never changes), in the extras:

```
assets/production/revisions.json
{"version": 1, "imported": {"from": "X.fdx", "draft": "01-pink-imported",
  "sets": [{"name": "Blue"}, {"name": "Pink"}],
  "marks": [{"line": 12, "set": "Blue", "text": "She runs."}]}}
```

That issue starts the chain with those marks: until the next Issue they are the last issue's marks, edits after it
are the next set's (Yellow after Pink); from then on the snapshots take over (all issues' marks still show the
imported ones on lines nothing changed since). A mark whose line's text is not what was recorded is dropped.

Rough: a deletion next to a changed line has no mark of its own; a dual dialogue is marked whole (both columns'
lines) when either speech changed; Lexington's command line has no revision flags yet; the editor's bar is the fold column's second cell
(no column of its own); the title page does not list the issued sets yet; locked pages and A-pages are step 5, so
"changed pages" are by the current pagination; the browser's PDF export of a script the PDF writer cannot print
(right to left, CJK) goes through HTML and has no marks.

## 11. Step 5, built: locked pages, A-pages and CONTINUEDs (2026-11-20)

Branch lockedpages (Accolade), Lexington v1.10.0 (tagged locally, not pushed; Accolade's go.mod has `replace
github.com/LaPingvino/lexington => ../lexington` until it is).

**The rules, and where they come from**

- A page that grows goes on to A-pages, named as A-scenes are in 10A style (`lex.PageLabelBetween`, which is
  `lex.SceneNumberBetween`): 12, 12A, 12B. 12AA comes between 12A and 12B, and 12aA between 12 and 12A.
  - 12A, 12B: Wikipedia's Shooting script; WriterDuet's 2A (§2).
  - 12AA after a second lock: WriterDuet ("1AA, 2AA … for any new additions after that point",
    https://www.writerduet.com/article/112-lock-pages). Filmustage gives 10AA between 10A and 10B, but for scene numbers,
    and I saw it only as a search snippet.
  - Small letters: Scripto falls back to them "unless that would break sort order"
    (https://help.scripto.live/en/articles/11689842-page-locking). A Final Draft user's "4aA" is seen as a
    search snippet of its knowledge base only, whose pages refuse robots.
- A page that shrinks stays short. Nothing is printed on it: I found no source for a "(page intentionally short)"
  line.
- A page whose text is all gone gives its number to the page before it, which then reads "12-13".
  - Scripto: "Page 4 becomes 4–5".
  - Final Draft: "7BB-9A", a search snippet only.
- Pages after the last locked page count on (121 after 120), as scene numbers do after the last. This is my choice:
  no source covers it.
- Scene CONTINUEDs. "(CONTINUED)" goes on the line under the last text of a page whose scene goes on, ending where a
  transition ends. "CONTINUED:" goes on the next page's first line at the heading's place, with the scene's number
  in both margins. Sources:
  - johnaugust.com/?p=3959: "most screenwriting software will offer to put CONTINUED: at the top left of the next
    page, next to the scene number"; only when "headed into production".
  - The "(CONTINUED)" at the foot: SimplyScripts' format sample (https://www.simplyscripts.com/WR_format.html) has it
    under a page's (MORE). A school's format page puts it "at the bottom of the page where the break occurs"; I saw
    that page only as a search snippet. Charles Deemer's guide (https://www.oocities.org/cdeemer/format.htm) covers
    the top's CONTINUED: and an old "CONTINUED 2:".
  - "CONTINUED: (2)" on a scene's second page after its first, and on: Final Draft's way as I remember it.
    *Unverified.*
  - A page always keeps its last line free for the foot's mark, and a continued page starts two lines down. So a
    page's text never depends on what follows it.
- Dialogue's (MORE) / (CONT'D) was there already. A lock that falls inside a speech puts them in too.

**Lexington v1.10.0**

- The anchor (`lex.PageLock`, `lex.LockedPage`) is by words, as Accolade's links are. A page has:
  - the number of the scene it starts in;
  - its first eight words, normalised (lower case, no punctuation or emphasis);
  - the eight words before it;
  - which place of its first words in the scene it is, when they repeat.
- A page is found again (`lex.LocatePages`), in this order:
  1. its first words at their place in its scene, with the words before them;
  2. the only place they stand with those words before them;
  3. their place in the scene;
  4. right after the words before it (its first line was edited).

  A page found by none of these is gone: both sides of its break were rewritten. There is no nearer matching,
  because a few words in a row ("he looks at her") repeat too often in a script to tell a page by.
  - A page found by the words before it at the next page's place is gone, as is one whose words are past the next
    page's place.
  - A page whose scene is gone (cut or omitted) is looked for anywhere after the page before.
- In the PDF (`pdf.PDFWriter.Lock`), each found page gets a break of its own before its first row, or within the
  row. A break within an action paragraph splits it at the word. One within a speech adds (MORE) and NAME (CONT'D)
  when the pagination breaks speeches so (`more`); without it, the speech splits as a paragraph does.
  - The labels are printed once the text is laid out, so "12-13" can be known.
  - No page starts with a blank line any more, and page numbers print without a title page too.
- A lock carries its paper and CONTINUEDs, and the writer prints a locked script so.
- `PDFWriter.LockPages` locks the pages as printed. Locking again keeps the A-pages, as Final Draft's Lock A Pages
  does. `PDFWriter.Pages` lists the pages: number, revision, new or not.
- Revised pages with a lock:
  - a page with a mark;
  - the pages after it in the same locked page (its A-pages, which also count as new);
  - a "12-13" page;
  - the page before a locked page whose first line is marked: a deletion at its foot marks the line after it.
- The command line has `-lockpages` (writes the lock as text), `-lock file` and `-continueds`. Programs use
  `cli.LockPages` and `cli.Pages`.
- Tests:
  - Locked and printed again, the same script gives the same pages and the same lock. This holds in every
    pagination (`none` and `keep+widows` too, with long speeches breaking over pages), on Letter, A4 and A5, with
    and without CONTINUEDs, and for all of Big Fish (FDX fixture) with and without them.
  - Growth gives one A-page, then two, on two pages at once, and at the boundary.
  - A paragraph cut gives a short page; a whole page's text cut gives 4-5.
  - A page's first words edited: it is still found.
  - Locked again: an A-page keeps its number, then 4aA and 4B.
  - Revised pages only gives the page, its A-page and their headers. A deletion at a page's foot also counts the
    short page.
  - CONTINUED marks pair up, including "(2)".
  - Final Draft's EuroMechanical.fdx read with `ParseLockedPages`: its page 4 starts at "I'm pushing to do a video",
    mid-speech as in the file. Lexington prints its four pages with no A-page. Checked by hand; the file is not
    committed.

**FDX** (`FDXWriter.Lock`, `fdx.ParseLockedPages`)

- The form is `<LockedPages><LockedPage LevelIndex LockLevel PageNumber Position/>`, as Final Draft wrote it in
  EuroMechanical.fdx (michaelforrest/ShootingChecklist).
- Its four Positions fall at paragraph starts and inside a speech after a sentence. They count the text runs'
  characters plus one per paragraph end, the same count as DeletedText's (§10). Checked on that one file only.
- *Unverified:* what Final Draft writes in `LockLevel` and `LevelIndex` for an A-page. Every page seen had 0 and
  its index. Lexington writes the number of letters (12A: 1) and the index.
- Reading takes PageNumber and Position. It drops a position past the end, or one not after the page before:
  another file seen (captioncast's small.fdx) keeps stale positions after its text was cut.
- A missing page is not written.
- Round trip: written and read back, the same labels and words.

**Accolade** (lockedpages.go)

- The lock lives in the production block for every kind of file, bundles included, and replaces §4's
  `pages.json`. Reasons:
  - Undo covers it;
  - it survives a flattening to plain Fountain and edits in other programs;
  - it is a few kilobytes of words for a feature.

  ```
  /* Production
  scene numbers: locked, 10A
  revision: Blue, 2026-11-20
  pages: locked, letter, format screenplay, pagination all
    1 -:  / int room 1 day anna walks along the
    2 1: about the house and about the people / who lived here before her long ago
  */
  ```

  The head line holds the layout the pages were locked in:
  - paper, and `continueds` when they were on (Lexington's);
  - the script format and the pagination (Accolade's).

  Each page is `label scene[*n]: words before / first words`.
- **Production > Lock Pages** locks the pages as the PDF prints them with the export's settings. Run on a locked
  script it locks them again with their A-pages.
- **Unlock Pages…** asks, then takes the lines out.
- Each is one undo step.
- **Export, Preview PDF, FDX** use the lock's layout whatever Page Setup says, when exporting a whole
  script. Filters (tags, sides) leave the lock out. Print (HTML) keeps no locked pages.
- **Issue:** revision N prints against N−1's pages, and N+1 is written against N's.
  - The block always holds the last issue's pages. Every Issue of a locked script locks them again from the text
    as issued, keeping the A-pages (Lock A Pages), so an issued 12A stays 12A. Text written later before it makes
    12aA, not 12B.
  - The first revision's Issue (Blue) offers, checked by default, to lock the pages as White printed them. The
    pages are locked so and then locked again as Blue prints, A-pages included. Later issues of an unlocked script
    offer it unchecked.
  - A PDF of the script as last issued, unchanged since, prints against the issue before's pages:
    - that issue's own lock (its snapshot's production block);
    - else that issue's text locked now (`withIssuedPages`).

    So the issue's A-pages are new pages. An A-page holding only pushed-down text is a revised page, and a page
    whose text went prints "12-13". An FDX gets the block's pages: the issue's own, A-pages included.
- **Preferences > Script > Production** has "CONTINUED where a scene breaks over pages (PDF)" (`export-continueds`,
  off). A lock keeps its own.
- **FDX import** puts an FDX's locked pages into the production block, at Final Draft's breaks. Lexington's layout
  differs from Final Draft's, so pages that print longer get A-pages: the page identities are Final Draft's.
- Tests:
  - lock, Undo, Redo;
  - text that grows exports 2A;
  - Unlock and its Undo;
  - Issue White; grow; Issue Blue with the offer: one A-page, revised pages only that page and its A-page;
  - Pink grows the same page before Blue's text: Blue's 1A stays, the new page is 1aA;
  - Pink issued: locked again with both, and revised pages only has 1aA;
  - locked before White, a short paragraph pushes unmarked text onto an A-page: Blue's revised pages are the page
    and its A-page;
  - a page's text cut: "2-3", in the full PDF and in revised pages only;
  - FDX out with the A-page, and the import keeps the lock (its status says that pages printed longer go on to
    A-pages);
  - the Issue dialog's offer (shots `lockedpages-issue-wide/phone.png`).

**Rough**

- Reports (Production > Reports…) still give whole page numbers from the current layout, not the locked labels.
- The revised-only PDF prints no number on page 1, as page 1 never has one.
- The FDX reader adds a blank line around Fountain's dual dialogue, so a script that went through FDX may print a
  line longer. A lock shows this as an A-page.
- `LockLevel` and `LevelIndex` for A-pages are unverified, and nothing written here has been opened in Final Draft.
- Pages whose text repeats word for word (the same 16 words around the break) can be told apart only by their order
  in the scene.
- Print (File > Print, through HTML) does not keep locked pages: use the PDF.

## 12. Step 6, built: the Production pane (2026-11-20)

Branch prodpane (Accolade only; production_pane.go, production_checks.go).

**Where.** A third tab of the Notes pane, beside Notes and Source: View > Production, and a toolbar button (Material's
"movie") after the source's. The button is an ordinary one on the overflow row: on a phone's width it goes into the
"more" menu (the source's stays). Under 600 px the pane is under the script with the others, and its buttons go to
the end of its list, so the list has the half screen (fixed under it, they left one row). Not in Markdown (menu item,
button, tab). The Production menu stays as it was; the pane is the guided way to the same actions.

**The strip.** `1 Lock` and `2 Revise` are buttons (since step 7 `3 Break down` too, §13; since step 8 `4 Schedule`
on a row of its own, §14); under them, greyed, "4
Schedule · 5 Day pack: coming". The stage shown stands out; the script's own stage, when another is looked at, is the plain button; Lock
gets a tick once issued. Either stage can be looked at again (§6: "each stage can be returned to"); the pane goes
back to the script's stage when it changes (White issued).

**The stage** is Lock until the drafts store holds an issue, then Revise. The store decides, not the production
block's `revision:` line, which Undo can take out while the issue stays.

**Lock.** The count line: "Not locked yet · 87 scenes · 3 to check" (scene numbers and pages may be locked before);
after White, "Locked 1 Nov 2026 · 112 pages · 87 scenes" (the locked pages' count, else the status bar's estimate,
"~"). The check before the lock, each row a tap (or its → button, for the keyboard) to its line; the body only, the
blocks at the end and the boneyards' contents left out:

- a heading without INT./EXT. (the script language's scene words, a forced `.HEADING` too) or without a time (the part
  after the last ` - `); an OMITTED heading is not checked;
- scene numbers: a number twice (on the later heading, naming the earlier's line); with some numbered and some not,
  each without one; a number left inside a heading; with none numbered, one note (not counted as a problem): "No
  scene numbers yet: Lock Scene Numbers writes them all";
- a `/*` not closed (or opened again before its `*/`: the script between is hidden), a stray `*/`; a `[[` not closed
  by its `]]` or an empty line, a stray `]]`;
- one character under two cues: a cue the adaptation block's `characters:` names as another name of a cue
  ("ANNIE (1 cue) is ANNA's other name, as learned: one cue for both?"), and cues alike but for case, dots and spaces
  ("MRS BENNET (1 cue) and MRS. BENNET (14 cues): one character?", the rarer flagged);
- a scene over three pages by the status bar's estimate (55 lines a page; action 60 columns, dialogue 35):
  "Very long, about 6 5/8 pages";
- no title page (its row opens Edit > Title Page…).

Buttons: Lock Scene Numbers ("Scene Numbers Locked", greyed, once locked), Lock Pages ("Lock Pages Again" when
locked), Issue White… ("White Issued" after).

**Issue White locks the pages.** The White dialog (from the menu or the pane) now offers "Lock the pages as White
prints them", checked: the pages are locked from the text as issued, after Issue has numbered its scenes (page
anchors carry the scene number). Blue's changes then go onto A-pages without the first revision's offer, which stays
for a script issued White unlocked. (`issueDraft` has a `lockNow`.)

**Revise.** The count line: "Blue (open) · 6 pages changed · scenes 4, 12A, 17" (the first eight, then "and n more";
"Title page" for a change before the first scene). Right after an issue, its own: "Blue issued 20 Nov 2026 · 6 pages
changed · scenes …", or "White issued 1 Nov 2026 · no changes since". The list: the changed scenes, each with a bar in
its set's colour, number and heading, a tap to its heading. Buttons: Issue Blue… (the next set's name; greyed with
no change since the last issue), Revised Pages PDF… (the Export dialog set to PDF, revision marks, revised pages only for that export,
named `story-blue-pages.pdf`; greyed with nothing changed), Compare with White (Drafts' Compare, the last issue
against the script now).

- The changed scenes are the current revision's marks only: the marks are made with the margin's (same 400 ms pause,
  same background run), and filtered by the current set, so "all issues' marks" in the margin does not widen them.
- The pages are counted by Lexington's own pagination (`cli.Pages`) in the background, with the export's settings,
  in the pages the revision prints against (locked pages, the issue before's for an issue just made), so the count is
  what Revised pages only prints, A-pages included (tested). Only while Revise is shown, once per text; "counting
  pages…" meanwhile.

**Following the text.** The pane is made again when the revision marks are (400 ms after typing pauses, and after an
issue): never per keystroke (tested with the real timers). The check is computed once per text; the rows are made
again only when what they show changes; a tab switch or a width change shows it at once.

**Tests** (production_pane_test.go): each check on its line, a clean script, omitted scenes, stray closes; the changed
scenes and their list; the stages and counts through White, edits, Blue (with all issues' marks shown), Pink and back
to Lock; the count waits for the pause; a tap on each row and its button goes to its line (a stale line to the end);
the buttons call the actions (Lock Scene Numbers, Lock Pages, the White dialog with its lock, Issue Blue, the export
set up, Compare); the page count against the revised-pages PDF with an A-page; the phone's layout (380 px wide, the
pane no wider than 320, the buttons at the list's end, the toolbar button in the menu); Markdown without it; the
pane's focus visible, a row scrolled into sight when Tab reaches it. Shots: `SHOT_DIR=… go test -run
TestProductionPaneScreenshots` (combo-prodpane-lock/revise-wide/phone.png).

**Rough**

- The check is heuristic: a time is anything after the last dash (`INT. HOUSE - CONTINUOUS` passes; a list of the
  language's times is not checked against), and a long scene is by estimate, not the paginator.
- Cue variants: only learned aliases and spellings that differ by case, dots and spaces; ANNA and ANNA K. are not
  matched.
- The page count lays the whole script out in the background after each pause while Revise is shown; in the browser
  that is one goroutine without yields, so a very long script may stall the page briefly. Revised Pages PDF… prints
  with the margin's setting (all issues' marks if chosen), while the count is the current revision's.
- The "unsaved title page" of §6 is "no title page": a title page is part of the text, so it is saved with it.

## 13. Step 7, built: the breakdown (2026-11-20)

Branch breakdown (Accolade), Lexington v1.11.0 (tagged locally, not pushed; Accolade's go.mod has `replace
github.com/LaPingvino/lexington => ../lexington` until it is).

**The categories** (`lex.TagCategories`, Lexington lex/tags.go)

- Ids, names and numbers as Final Draft 13 writes them in a new script (fdx-mcp-server's NewFile.fdx, Version 6):
  Cast Members, Background Actors, Stunts, Vehicles, Props, Camera, Special Effects, Wardrobe, Makeup/Hair, Animals,
  Animal Wrangler, Music, Sound, Art Department, Set Dressing, Greenery, Special Equipment, Security, Additional Labor,
  Visual Effects, Mechanical Effects, Miscellaneous, Notes, Script Day, Unit, Sequence, Location, Comments, Synopsis.
  These are Movie Magic's names too.
- Colours: Final Draft 13 gives every category but Cast the same dark red (`#94AA11150000`), useless for telling tags
  apart. So the colours are those of a Final Draft Version 5 file (screenplay-tools' TestFDX-FD.fdx), which Fade In's
  OSF 4.0 `<tag_categories>` has too, value for value: Cast blue `#0000FF`, Background Actors `#00BFFF`, Stunts orange,
  Vehicles `#8B0000`, Props red, Camera `#CCFF00`, Special Effects `#1E90FF`, Wardrobe purple, Makeup/Hair `#9370DB`,
  Animals `#CD853F`, Animal Wrangler `#8B4513`, Music `#20B2AA`, Sound teal, Set Dressing `#BA55D3`, Greenery green,
  Special Equipment `#BDB76B`, Security `#DAA520`, Additional Labor `#B8860B`, Visual Effects `#32CD32`, Mechanical
  Effects `#B22222`, Miscellaneous `#708090`, Notes gold. The seven Final Draft 13 added since (Art Department, Script
  Day, Unit, Sequence, Location, Comments, Synopsis) have colours of Lexington's own.
- Read as the same: Version 5's names (Cast, Animal Handler) and its Miscellaneous's other Id; Beat's keys (`prop`,
  `costume`, `setDesign`, `other` …, BeatTagging.m).
- One's own: Tag Selection… takes a new name ("New category “Puppets”"); it gets a colour in turn. In FDX it is
  written after the standard ones, numbered on from 31.

**Where tags live.** `assets/production/breakdown.json`, in the script's extras (§9): the bundle's, else the kept copy
of a plain script (made at the first tag, as for the first draft), else `Name.production/` beside it. Not the
production block: a breakdown is hundreds of tags, which would bury the block (and the lock's pages) other apps print
as boneyard; and the file is the record the bundle was made for. Undo does not cover it: Untag takes a tag away.

```
{"version": 1,
 "categories": [{"id": "…", "name": "Puppets", "color": "#C71585"}],
 "elements": [{"id": "…", "category": "Props", "name": "revolver"}],
 "tags": [{"element": "…", "scene": "12", "heading": "int kitchen night", "nth": 1,
           "at": "text=the table. A-,revolver,-again. ANNA Where"}],
 "done": [{"scene": "12", "heading": "int kitchen night"}]}
```

- An element is a name in a category, tagged once and found wherever it is tagged again (the same name, any case).
- A tag's scene: its written number (`#12#`) when it has one, the heading's words, and which of the scenes with that
  heading it is (`nth`). Its words: a Text Fragment directive (textfrag.go's syntax) with up to three words before and
  after in the scene.
- **Found again** (breakdown.go `locate`): the place with the words whose words around them match the anchor's
  best, looked for in its own scene when it is numbered; else in the scenes with its heading together (of equals,
  the one that is which-of-those it was); else anywhere. A place is taken when it is told apart: in its own scene when
  that is sure (by its number, or a heading no other scene has), else with at least two of the words around it as
  anchored, or words (more than one) that stand once in the script. Otherwise the tag is **lost**: shown in the pane,
  kept in the file, found again if the words come back. A single word that left its scene is lost rather than moved
  to another scene that has the word.
- Tested: words before it in its line, a new scene before all, the same word earlier in its scene, Lock Scene Numbers
  after tagging, a numbered scene's heading changed, the words cut and restored, a scene with the same heading and
  the word put before it, the words cut while two other scenes have the word (lost, not moved).
- A `breakdown.json` that cannot be read is never written over: tagging says so (mend or move the file).

**Tagging** (Production menu; breakdown_view.go)

- **Tag Selection…** (Ctrl+Shift+B) opens a picker: a filter that also names a new category, then the categories
  with their colours, the last used first; a tap, or Enter for the first, tags. With nothing selected it takes the
  word at the cursor. A tag stays within one line. Cast Members' names are written in capitals, as cues are.
- **Untag** takes away the tags at the cursor or in the selection. An element left without tags goes too.
- The editor's **right click** (a long press on touch) offers "Tag as Props" (the last used), Tag Selection…, Untag
  "revolver" (Props), and Cut/Copy/Paste. *Unverified* in the browser: on a phone the pane's Tag Selection… button and
  the menu are the sure way.
- **Show Tagged Words** (checked by default): tagged words in their category's colour (mixed 30% towards the text's
  colour, so that light ones read) and underlined. The editor draws them per line (`editor.Ink`, looked up by line as
  Markdown's emphasis is). The tags are found again in the background 400 ms after typing pauses, as the revision
  marks are, never per key (tested with the real timers).

**The Break down stage** (production_breakdown.go; the pane's third button)

- "62% broken down · 33 scenes to go" (OMITTED scenes left out; "· 2 lost tags" when there are).
- The scene at the cursor: "Scene 12 · done", its heading with its eighths (from the reports, made in the background
  once per text while the stage is shown: "measuring…" before), its location ("KITCHEN (INT, NIGHT)"), its cast (who
  speaks, numbered as the cast report numbers them: "1 ANNA (speaks)", then those tagged as Cast Members: a silent
  part), its elements by category in their colours (a tap selects the words). Then "Tagged in other scenes, found
  here": an element tagged elsewhere whose words this scene has, a tap tags it here. Then the lost tags, anywhere (a
  tap asks to remove one).
- The buttons (two columns beside the script, one under the other at the list's end on a phone):
  **Next Scene** goes to the first scene not marked done (the list shows its cast and location at once: they need no
  tagging). **Mark Scene Done** / **Mark Scene Not Done**. **Tag Selection…**. **Breakdown Sheets…** (the reports).
- The script's stage is Break down once White is issued and the breakdown has begun (a tag or a scene done); it can
  be looked at any time. Production > Break Down… opens it.

**Sheets and reports** (Production > Reports…, two more kinds; Lexington report/breakdown.go)

- **Breakdown sheets**: a row a scene (OMITTED ones left out). On screen the categories nothing is tagged in are left
  out; the CSV has them all. **PDF…** writes the classic sheet, a page a scene, with Lexington's gofpdf (no browser
  needed, so in the browser too): BREAKDOWN SHEET, sheet n of N, the title; a head of SCENE, INT/EXT, SET / LOCATION,
  D/N, PAGES (eighths), PAGE; the SYNOPSIS (the scene's first `= synopsis`); then a box per category in its colour,
  Cast Members first by cast number, the classic sixteen always, others when they have elements.
- **Elements**: each element with its scenes, its eighths (the sum of its scenes'), its first scene and page; the
  cast with their numbers.
- Cast numbers: who speaks as the cast report numbers them (§8), then those only tagged, in order of first tag.

**CSV layouts** (stable; golden files in Lexington's report/testdata; RFC 4180 as §8's)

| Report | Columns, in order |
|---|---|
| breakdown | `order`, `number`, `int_ext`, `location`, `time`, `page`, `eighths`, `length`, `synopsis`, `cast` (`1 ANNA; 3 CARL`), then a column per category but the cast, location and synopsis, in the standard order, keyed by name in lower case with `_` (`background_actors`, `makeup_hair`, … `comments`), then one's own categories (`puppets`) |
| elements | `category`, `element`, `cast_number` (the cast's), `scene_count`, `scenes`, `eighths`, `length`, `first_scene`, `first_page` |

**FDX**

- Out (Export > FDX, always): each heading's `<SceneProperties Length="1 3/8" Page="12">` from the PDF's measure with
  the export's settings (`cli.Options.SceneLengths`, `-lengths`); an OMITTED scene has its Page and no Length. Final Draft writes lengths so (`1`, `2/8`, `1 3/8`
  in Big Fish). With a breakdown: `<TagData>` with all the categories (Final Draft 13's list, these colours, one's
  own after), a `<TagDefinition CatId Id Label Number>` per element, a `<Tag Number><DefId>…</DefId></Tag>` per place
  tagged (two DefIds where the same words are tagged twice), and `TagNumber` on the runs of the tagged words (split
  out of their run, emphasis kept). The cast from the cues is tagged on each cue's name. The form of definitions,
  tags and TagNumber is Beat's FDX writer's (BeatFDXExport.m): no Final Draft file with tags was at hand, so that
  part is *unverified* against Final Draft.
- **Movie Magic Scheduling**: its documentation says an FDX import reads "the header (slugline) and the talking
  characters", and Final Draft's knowledge base that MMS "cannot read the tag information" of an FDX (tags reach MMS
  through `.sex`, which is closed, §3). So what MMS gets from Accolade's FDX is what it documents: Scene Heading
  paragraphs (with numbers and INT/EXT, set, D/N in the heading) and Character paragraphs; both are tested in the FDX
  out. The tags are for Final Draft, Beat, Fade In, StudioBinder and the others that read TagData; for MMS the
  breakdown CSV is the way (mapped by hand). Not tried in a real MMS.
- In (Open of an .fdx): `fdx.ParseTags` reads the TagData (standard categories by Id, else by name; others as one's
  own) and each run's TagNumber, with its text and place in the line; Accolade anchors them on the imported text by
  their words, as its own. Round trip tested: written, read back, the same elements on the same words, the cues'
  cast with them, also on the second line of a two-line action paragraph and of a two-line speech. The scenes on the
  sheets are the text's scenes matched to the report's (one to one, else by number; tested with an OMITTED scene, a
  forced heading and a title page).

**Beat's tags in.** On opening a script with Beat's data block: its `TagDefinitions` (`type`, `name`, `id`) are the
elements (Beat's types read as the standard categories), its `Tags` (`range` in UTF-16 units of the text before the
block, `definition`) the places, anchored by their words. Read only while the block's `Text Length` is the text's
(give or take the line breaks before the block): after an edit elsewhere the ranges point at other words, and nothing
is read ("Beat's tags not read: the script was changed since Beat saved them"). Tags already in the breakdown are not
added again, so opening again adds nothing. Accolade does not write Beat's block (§3). Tested with an emoji before the
tags (two UTF-16 units).

**Rough**

- A tag stays within one line; an element's words over a line break need two tags.
- The editor underlines in the word's colour (Fyne draws the underline in the text's colour), so a tagged word is
  coloured as well as underlined.
- Suggestions look for an element's exact words only (not plurals).
- The pane's eighths need the reports' layout, made in the background on each pause while Break down is shown (as
  Revise's page count; no yields in the browser).
- Undo does not take a tag back (Untag does), and tags are written when made, also before the script is saved.
- The sheet's PDF prints the boxes that fit on a page (a scene with very many elements: the CSV has them all), and its
  title is the script's file name.

## 14. Step 8, built: the schedule (2026-11-20)

Branch schedule (Accolade), Lexington v1.12.0 (tagged locally, not pushed; Accolade's go.mod has `replace
github.com/LaPingvino/lexington => ../lexington` until it is).

**The strip colours.** White INT. DAY, yellow EXT. DAY, blue INT. NIGHT, green EXT. NIGHT: Wikipedia's "Production
board" tabulates this as the generic convention and as StudioBinder's and Scenechronize's
(https://en.wikipedia.org/wiki/Production_board); StudioBinder's own stripboard article has the same table, and black
day breaks (https://www.studiobinder.com/blog/shooting-schedule-stripboard/). Movie Magic Scheduling swaps the night
ones (green inside, blue outside) in the same table: Accolade follows the generic. My choices where the four colours
say nothing:

- INT/EXT by the language's place words (Lexington's `rules` Matchups): EXT., INT./EXT. either way and EST. are
  outside (weather and light go with the outside); a forced heading or an unknown word is inside.
- Day or night by the time's words, in Lexington's languages (NIGHT, EVENING, DUSK, SUNSET; DAY, MORNING, DAWN …;
  NACHT, NOKTO, НОЧЬ, 夜 …): dusk and evening count as night, dawn and morning as day. A time that names neither
  (CONTINUOUS, LATER, none) takes the scene before's; the strip marks it `D*`/`N*`.

**Lexington v1.12.0** (report/schedule.go, report/ics.go)

- `report.Schedule{Days}`, `report.Day{ID, Date, Call, Notes, Scenes}` (each scene by its report Order, in shooting
  order). `Report.Strips(breakdown, StripOptions{Lang})`: a scene's strip, INT/EXT, D/N, colour, cast by number (who
  speaks and who is tagged, as the breakdown sheets'; the sheets now share the code).
- One-liner: `Report.OneLinerTable`, `report.WriteOneLinerPDF` (gofpdf: each day's head with date, call and notes, its
  strips in their colours with scene, I/E, set, D/N, pages, cast numbers and synopsis, then the black day break
  "End of Day 1 — 4 6/8 pages").
- Day Out of Days: `Report.DOOD`, `Report.DOODTable`, `report.WriteDOODPDF` (landscape, a column a day, as many pages
  across as the days need and down as the cast needs, the totals on each, a legend). The codes are Movie Magic Scheduling's
  (https://mms-docs.ep.com/DayOutofDays/DOODRules.html): SW the first day worked ("almost always shown as SW"), W, WF
  the last, SWF the only one, and H every day between the first and last not worked (MMS's "Allow Hold Days" on: "all
  non-working days for that Element between their first and last working day are marked as H"). Who: every cast
  member in a scheduled scene, by cast number.
- `report.WriteICS`: one VEVENT a dated day. UID `<day id>@accolade.kiefte.eu` (stable: the day's id in the JSON),
  DTSTAMP, SUMMARY "story: shoot day 2 of 3", LOCATION the day's notes, DESCRIPTION the call, the scenes in shooting
  order with their lengths and the day's total. With a call time: DTSTART at the call as a floating local time (the
  call is where the unit is, whatever the calendar's zone) and DTEND twelve hours on (`ShootDayHours`, a usual day: my
  choice); without: `VALUE=DATE`, DTEND the day after. Days without a date are left out (the status says how many).
  RFC 5545: CRLF, lines folded at 75 octets never inside a UTF-8 character, TEXT escaped (`\ ; , \n`).
- Tests: the colours and D/N words in several languages, CONTINUOUS, the DOOD codes (hold days between, a one-day
  SWF), the day totals, golden `oneliner.csv`, `dood.csv` and `shoot.ics`, the .ics's validity (line lengths, folds,
  required properties, escaping, the undated day left out), the PDFs (30 days go over pages).

**CSV layouts** (stable; golden files in Lexington's report/testdata; RFC 4180 as §8's)

| Report | Columns, in order |
|---|---|
| oneliner | `day` (1, 2, …), `date` (2026-12-01 or empty), `shoot_order` (1, 2, … over the whole schedule), `number`, `int_ext` (as written), `location`, `time`, `day_night` (`D`/`N`), `page`, `eighths`, `length`, `cast` (numbers: `1; 3`), `synopsis`, `strip` (`white`, `yellow`, `blue`, `green`); a row a scheduled strip, the unscheduled left out |
| dood | `cast_number`, `name`, `day_1` … `day_n` (`SW`, `W`, `WF`, `SWF`, `H` or empty), `work`, `hold`, `total`, `start_day`, `finish_day` |

**Where the schedule lives.** `assets/production/schedule.json`, in the script's extras as the breakdown (§13): the
bundle's, else the kept copy, else `Name.production/`. The text's Undo does not cover it; the board has its own Undo
of its last change. A file that cannot be read is never written over.

```
{"version": 1,
 "days": [{"id": "5f1c2a9b", "date": "2026-12-01", "call": "07:00", "notes": "Kovač house",
           "strips": [{"scene": "12", "heading": "int kitchen night", "nth": 1,
                       "start": "anna makes tea rain on the window is", "sum": "1c9e…", "set": "Blue"}]}],
 "pool": "location"}
```

- A strip names its scene as a breakdown tag does (written number, heading words, which of those it is) plus its
  first eight words. Found again (schedule.go `find`): by its number; else among the scenes with its heading, the one
  with its first words; else the only scene anywhere with its first words (its heading changed, another has its old
  one); else which-of-those it was; else the only one with its heading. Numbered strips are placed first. A strip
  moved gets its scene's key as it is now (so after Lock it is found by number). Tested: a scene with the same heading
  put before, Lock Scene Numbers, a number changed by hand, a heading changed, both, the scene cut (lost: kept in the
  file, listed, a tap removes it), the same scene on two days (the second left out and pruned).
- `sum` is the scene's words (heading included) when it was put on its day or checked; `set` the revision then. A
  scene whose words changed since is flagged on its strip and in the stage: "Changed in Blue: check", the set being
  the latest of its lines' marks newer than the strip's, from all issues' marks whatever the margin shows (so Pink's
  issue does not hide a Blue change), else the open set's, else "Changed since scheduled". **Checked** takes the flag
  away. A change before the strip was scheduled flags nothing.
- `pool` is how the unscheduled are grouped (`location`, `cast`, or script order): set by Group … by Location / by
  Cast with no strip on a day selected, back to script order by a tap on the unscheduled's head.

**The stripboard** (stripboard.go; Production > Stripboard, the stage's Open Stripboard). A middle tab, as the
cards' full view is, and dragged as they are (`dragged` now takes any `dropBoard`). The days one under the other: each its
head (Day 1 · Tue 1 Dec 2026 · Call 07:00, its notes, an edit button for date, call, notes or removing the day), its
strips, its black break "End of Day 1 — 4 6/8 pages"; then "Unscheduled · 69 scenes" and their strips; lost strips
last. A strip is paper in its colour, its ink dark whatever the theme: number, I/E, location, D/N (`*` from the scene
before), length, cast numbers; the flag under it in red.

- Its commands are the ribbon's contextual **Stripboard** tab (stripstab.go, as the Preview tab is; LOOP.md
  2026-11-23, Joop: "so same as preview ribbon"), there and in front while the board is: Move (Move To, Up, Down,
  Unschedule, Undo), Day (Add Day, Edit Day…, Fold / Unfold Days), Group (by Location, by Cast), Check (Open Scene,
  Checked), Close. In the Menus look the Production menu has them while the board is in front; the palette and `:`
  too; a strip's long press or right-click its own. The board has no toolbar: its first line is the count, then the
  strips, the tab's whole height.
- Drag a strip (desktop): onto a strip's upper half before it, lower half after it, onto a day's head or empty
  placeholder its start, its break its end, a folded day's band its end, the unscheduled's head back to the pool.
- A day folds to its band (its head's ▾; the tab's Fold Days): its scenes and length under its head, the board
  shorter; a strip moved onto it, or shown from the Schedule stage, unfolds it.
- Without a drag (phones, keyboard): a tap or Tab selects a strip (outlined; focused: a thicker outline), Up/Down
  select the next, Enter opens the scene. **Move Up / Move Down** (Alt+Up / Alt+Down) move it over its neighbour,
  across a day break into the day before or after; the pool's first strip up goes to the last day's end, the last
  day's last strip down back to the pool (Alt+Up/Down are Edit > Move Line Up/Down's keys, which give them to a strip
  that has the focus). **Move To** picks a day or Unscheduled. **Unschedule**, **Open Scene**,
  **Checked**.
- **Add Day** (dated the day after the last dated day), **Group Day 2 by Location / by Cast** (the selected strip's
  day, or the unscheduled): one click, once, never by itself. By location: in order of each location's first strip,
  then INT/EXT and D/N; by cast: the cast numbers compared in turn (the lead's scenes first), then location. **Undo**.
- On a phone (under 600 px) each strip is two lines with a grip (≡) at its end: a finger drags the strip by it (the
  board scrolling near its edges), a drag elsewhere scrolls; a long press has the strip's actions (Move Up/Down, Move
  To, Unschedule, Open Scene, Checked, Edit Day…). The dock's sheet folds down to its tab row while the board is in
  front.

**The Schedule stage** (production_schedule.go). The strip's fourth button (on its own row: 320 px hold three). "18 of
87 scenes on a day · 3 days" (OMITTED scenes left out; "· 2 changed scenes to check", "· 1 lost strip"); the days
(date, call, scenes, length, notes; a tap opens the board there), the changed scenes in their set's colour (a tap:
the board with it selected). Buttons: Open Stripboard, Add Day, One-Liner…, Day Out of Days…, Export .ics… (the last
three greyed with no day). The script's stage is Schedule once it has a day. Measured (the reports' layout, the strips,
all issues' marks) in the background once per text while the board or the stage is shown, after typing pauses (tested
with real timers).

**Reports.** Production > Reports… has One-liner and Day out of days (on screen, CSV, PDF by Lexington's gofpdf: no
browser needed); Production > Export Shoot Days (.ics)… and the stage's button save the calendar (the browser downloads
it).

**Tests** (schedule_test.go, production_pane_real_test.go): the strip colours; order, moves over day breaks and to and
from the pool, Move to, Undo, a day removed, the day totals and the break's text; grouping; the anchors (above);
two strips of one scene; the bundle (and the file read again after an issue); the pool's grouping undone, a silent
cast member tagged showing on the strip, Alt+Up through the menu; the revision flags through White, Blue, Pink and an open Yellow, and Checked; the file in the
kept copy, read again, never written over when unreadable, the next day's date; the one-liner and DOOD as the Reports
dialog makes them against golden CSVs (testdata/schedule/), the .ics; the stage's counts, rows and buttons, the menu
items; the keys (Up/Down, Alt+Up); drops on a strip's halves and the pool's head; the phone's layout (the board and the
pane within 380/320 px, a drag scrolling, the grip moving, the long press's menu, a folded day, the sheet folded
down); the board's focus visible and everything named; the Stripboard tab (stripstab_test.go: forward and back, its
buttons following the selection, its key tips, Production on Schedule, a phone; the Production menu in the Menus
look). Shots: `SHOT_DIR=… go test -run 'TestScheduleScreenshots|TestStripboardRibbonScreenshots'`
(combo-stripboard-wide/phone.png, combo-stripboard-ribbon-…, combo-stripboard-folded-…,
combo-schedule-stage-wide/phone.png, schedule-dood.pdf, schedule-oneliner.pdf).

**Rough**

- No drops and pick-ups, travel, fitting, rehearsal or holiday days in the DOOD (MMS's rules for a drop need a
  minimum gap setting); no banners or units in the board yet; one-liner and DOOD list the cast only (not other
  elements).
- The board shows every strip as a widget: a script of several hundred scenes builds them all at once.
- Drag or scroll is decided by the window's width, not by touch: a tablet or a phone held sideways (600 px and more)
  drags strips and cannot scroll the board by dragging over them (the scroll bar, or a narrower window).
- On a phone the actions take the board's top third; they could go to the list's end, as the pane's do.
- The pages are the current layout's (as the reports'), not a locked script's labels.
- A drag on the desktop is checked by drop positions in the tests, not by a pointer; on a phone the board is moved with
  the buttons only.
- Lexington's command line has no schedule flags (the schedule is Accolade's file).

## 15. Taking a lock back, and Renumber (2026-11-20)

Branch renumber (Accolade; renumber.go, renumber_follow.go). Joop: "locking scene numbers must be easy to take back
for a writer who plays around".

**Which numbers were Lock's.** Lock Scene Numbers now writes a second line in the production block, the numbers the
headings had before it (`none` if none):

```
/* Production
scene numbers: locked, 10A
numbered before the lock: 3, 7A
*/
```

Everything else numbered while locked, Lock's numbers and the letters new scenes got, is Lock's. A lock written before
this line existed has no record; Remove then takes every number out, and the dialog says so.

**Unlock Scene Numbers…** asks, in a dialog of its own (buttons one under the other on a phone):

- *Keep the Numbers*: the numbers stay; the lock's two lines go.
- *Remove the Numbers*: the numbers Lock wrote go, the ones typed before the lock stay (named in the dialog: "the
  ones you typed before locking stay (7)"). With none typed by hand the script is byte for byte what it was before
  the lock (tested). Decision: keep the typed ones. They are the writer's own text; Remove means "take the lock
  back", and Lock never wrote them.
- Remove is the default (the highlighted button) while the drafts store holds no issue, and Keep once it does. An
  issue means the numbers went out ("The script was issued: White (1 Nov 2026)…"). An FDX kept as an imported issue
  counts too: it is kept in the drafts store as an issue (readRevisionChain).
- One undo step either way. The page lock's scene numbers follow in the same step (a page's scene whose number went
  is "none": Lexington then looks for its words after the page before).

**Production > Renumber Scenes…** (and the pane's button) opens a dialog of strategies with a preview of old → new
(below). Its *Clean 1…n* numbers every scene 1…n in script order, so A-numbers and gaps go: Renumber as it was before
the strategies. One undo step, the locked pages following. The lock stays; its "numbered before" line keeps only the
hand-typed numbers whose scene kept its number (after Clean: `none`, every number is the lock's now). After an issue
the dialog opens with a warning that sides, schedules and printed pages that went out use the old numbers, and that
Accolade's own breakdown, schedule and locked pages follow but nothing printed does. It offers *Save a Draft, Then
Renumber* (the default: a draft named "Before renumbering" in the drafts store, then the renumber), *Renumber*, or
*Cancel*.

**OMITTED when renumbering.** An OMITTED scene loses its number, and its heading goes. The scene's text stays where it
was as boneyard, `/* Omitted scene` with the old heading inside without its number. Remove does the same to an
OMITTED scene whose number goes. Why:

- OMITTED exists to retire a number that went out with the script, so that no other scene is given it (Wikipedia,
  [Shooting script](https://en.wikipedia.org/wiki/Shooting_script): its number "is preserved in the script along
  with the phrase (OMITTED)").
- Once the numbers are out, continuity is the rule: "With rare exception, you always maintain scene number
  continuity" (John August, [Renumbering when moving a scene](https://johnaugust.com/?p=730), 2007).
- Renumbering is what closes those gaps. StudioBinder, [Scene numbering](https://studiobinder.com/blog/scene-numbering):
  omitted scenes leave gaps in the numbering, and renumbering makes sense once a rewrite has left gaps and lettered
  scenes (paraphrased). It also warns that renumbering assigns the breakdown, shot lists and schedules "to a different scene", which is why Accolade's
  sidecars follow (below) and why the warning after an issue exists.
- Final Draft's own help page on Scene Numbers (Number/Renumber, Remove Numbers) refused the fetch (403). It is not
  cited further.

So a renumbered script starts its numbers afresh, and no placeholder keeps one. None of the three sources says
this in so many words; it is my inference from what OMITTED is for. A numberless `.OMITTED` heading was
rejected: the lock would give it a letter at the next pause, and export would print it with a number.

### Renumbering strategies (2026-11-20)

Branch renumstrat (Accolade; renumber_dialog.go, renumber_strategy.go), Lexington v1.13.0 (local tag; go.mod
`replace => ../lexington`). Joop: Renumber Scenes… becomes a dialog with choices and a preview, one undo step.

**The numbering is Lexington's.** `lex.Renumber(scenes, RenumberOptions{Free, KeepLetters, Omitted, Reserved, Style})`
gives each scene its new number, or Drop for an OMITTED one (Merged went in v1.14.0: see *Scene numbers are labels*). A scene is *free* (gets a new number) or
*pinned* (keeps its own). Free scenes are counted on from the pinned scene before them (1, 2, 3 … after none, 8 after
a pinned 7), skipping a taken number (a pinned one, a kept OMITTED one, a reserved one). Where the count would reach
or pass the next pinned number, the scene gets a letter between its neighbours, as a locked script gives a new scene
one (`SceneNumberBetween`, the lock's 10A or A11). So a pinned number never moves, the numbers stay in order around
it, and no number is given twice (a property test over 3000 random scripts and strategies).

**The dialog** (left the strategy, right the preview; on a phone one above the other, buttons one under the other):

- *Scenes*: all / selected (the scenes from the selection's start to its end, a selection ending at a line's start not
  taking that line's scene; without a selection the cursor's) /
  from scene … to … (two pickers in script order; the end before the start is an error under the field, Apply
  disabled, never swapped) / from the cursor on. Scenes outside are pinned: a range cannot close
  a gap outside it (1, 2, 2A, 2B, 3 with only 2–2B free stays so: 3 is taken).
- *Numbers*: no list / *Keep these numbers* / *Only these numbers*, typed as `5, 12A, 20-24`
  (`lex.ParseNumberSet`, read against the script's labels; since v1.14.0 by the script's order, below). Kept scenes
  are pinned, and a kept label no scene has now is reserved: never given. *Only* renumbers just the listed scenes, the
  rest pinned. One list, one meaning: keep and only exclude each other. Under the field: how the list was read
  ("Read as: 6 to 9 (3 scenes) · 99 (no such scene)"), the labels no scene has, and in red what keeps it from being
  applied (a range backwards, an end no scene has, a dash read two ways), Apply disabled until fixed.
- *A-scenes*: fold into whole numbers (default) / keep the letters: 10, 10A, 11 after a cut become 9, 9A, 10. Kept
  letters are given after the whole numbers, between the new neighbours, in the lock's style (A10 with A11).
- *OMITTED*: drop them and close the gaps (Clean's) / keep them with their numbers (pinned: the gap stays) / keep
  them, print runs as one: `12-14 OMITTED` (below). Only OMITTED headings in scope; one out of scope stays as it is.
- *Presets*: *Clean 1…n (before anything went out)*; *After an issue: keep sent scenes*, the keep list filled with
  every number in any issue (the drafts store's issues; OMITTED ones too: the script's runs of scenes that all went
  out written as ranges in its order, `1-1A, 4-9`, a scene written since breaking a run; then the numbers that went
  out and no scene has now, one by one), letters folded, OMITTED kept (printed as the script prints them now): a scene cut since keeps its number
  retired, a scene written since gets a free number or a letter; *Last used*, kept in settings.json
  (`renumber-last`; a range of another script's scenes falls back to all). The default is Clean before the first
  issue, Keep sent scenes after. Changing any control makes the preset "Your own".
- *The preview*: every heading, old → new ("stays" greyed, "gone" for a dropped OMITTED; with print runs as one, an
  OMITTED heading of a run says "· prints as 12-14"), and "n headings of m change", with the runs printed as one and
  whether the setting changes. The preview and Apply call the same plan (`strategyPlan`), so the preview is
  what the script gets (tested for every strategy). Nothing to change: the buttons are disabled. If the headings
  changed while the dialog was open (the lock numbering a new scene at a pause), Renumber makes the preview again,
  says so, and applies only on the next tap.

**OMITTED 12-14 (v1.13.0, superseded).** The first strategies merged a run of OMITTED headings into one,
`.OMITTED #12-14#`. Joop: "using numbers in sequence is a bit of a hack, technically I think a scene number can be
anything", and "ranges are ill-defined — useful in dialogs, not in the document itself". Reading that form back was
dropped too (Lexington v1.15.0, below).

**The breakdown and the schedule follow** (renumber_follow.go). Tags, scenes marked done and strips name their scene
by number first, and are kept outside the text's undo. The window keeps the scene numbers it last saw. When only the
numbers changed (the same headings in the same order, OMITTED ones aside), the sidecars follow. This happens after
Lock, Remove, Renumber, Undo and Redo (the menu's, and the editor's own Ctrl+Z once typing pauses), and when a writer
retypes a number. Each tag or strip found in the text before (by its anchors, as shown) gets the number its scene has
now. A lost one is renamed only from a number (old → new). A done mark is renamed by number, else by heading and
which of those it is. A strip's words sum (it counts the heading's number) is taken over when its scene was
unchanged, so Renumber or Lock does not flag every strip "changed". The board's own Undo snapshot follows too.
Two guards apply. A number that two scenes share belongs to neither: nothing is renamed into a text that has a
number twice, and the window waits for the next text (a number being retyped, #2# → #1# → #5#, passes through one).
A number shared in the text before is renamed only where a tag or strip was found. And an OMITTED scene that
Renumber or Remove takes away takes its number with it: its strip, tag or done mark is lost, never moved to the
scene that now has that number. Undo of that step does not move it either. Before
this change, Renumber would have pointed a strip numbered 3 at the scene now numbered 3.

**The pane.** The Lock stage shows *Unlock…* and *Renumber* beside *Scene Numbers Locked* ("Locked" on a phone). The
count line says "Numbers locked, not issued" between the lock and White.

**Tests** (renumber_test.go): Remove byte for byte, then Undo and Redo; Keep; an old lock without the line; the
dialog's default before and after an issue, Cancel; Renumber with an A-scene, a gap and OMITTED, Undo, Remove with
OMITTED, Restore after Undo; Renumber after an issue (Cancel, then Save a Draft, Then Renumber with the draft stored);
a tag on a word in two scenes, a done mark and two strips through Renumber, Undo, Redo, the editor's own Undo,
Remove, and Undo of Remove (stored numbers, nothing flagged changed); the locked pages' scene numbers through
Renumber, Undo and Remove (three long scenes, pages starting in 1A and 2); a tag, done mark and strip on a scene
that is then omitted, through Renumber, Undo and Remove (none moves to the hall, which gets its number); a number
retyped through a duplicate, and Renumber from a duplicate; the pane's buttons. Shots: `SHOT_DIR=… go test -run TestUnlockDialogScreenshots` (unlock-wide.png,
unlock-phone.png, combo-prodpane-locked-wide/phone.png).
Strategies (renumber_strategy_test.go, and lex/renumber_test.go in Lexington): each scope, keep (a number, two, a
range), only, letters folded and kept, each OMITTED mode, keep with a range, a range typed backwards, the selection and
the cursor; for each the preview equals the result and Undo is one step; the "numbered before" line; the presets (Clean before an issue, Keep sent scenes after one with a scene cut and one
written since, Your own, Last used); tags, a done mark and a strip through a range-with-keep strategy, Undo and Redo;
the locked pages through a from-the-cursor strategy and Undo. Shots: `SHOT_DIR=… go test -run
TestRenumberDialogScreenshots` (renumber-wide/phone.png, renumber-wide/phone-issued.png).

### Scene numbers are labels (2026-11-20)

Branch scenelabels (Accolade; omitted_runs.go, renumber_strategy.go, renumber_dialog.go), Lexington v1.14.0 (local
tag; go.mod `replace => ../lexington`; lex/labels.go, lex/numberset.go, lex/omitted.go).

**Labels and the scheme.** Fountain allows "any alphanumerics, plus dashes and periods" in a scene number
([fountain.io/syntax](https://fountain.io/syntax): `#1#`, `#1-A#`, `#I-1-A#`, `#1.#`, `#110A#`), so a scene
number is an opaque label. Lexington's numbering (`NumberScenes`, `NumberLabels`, `SceneNumberBetween`,
`Renumber`) reads one scheme, the one it writes (`lex.InNumberScheme`):

- a whole number without a leading zero (1, 10, 110);
- with letters after it, the 10A style: 10A, 10AA, 10B, and Wikipedia's small-a filler before a letter (10aA, 10AaA;
  never at the end: `10a` is not one);
- with capitals before it, the A11 style: A11, AB11; and both (A11B, which the A11 style gives before an 11B).

Every other label is **fixed**: `I-1-A`, `1-A`, `1.`, `1.1`, `PROLOGUE`, `01`, `10a`, a Storyboarder id. A fixed
label is never renumbered (by any strategy, OMITTED or not: Drop leaves a fixed OMITTED heading; Unlock's Remove
leaves it, as no lock writes one), never given to another scene, and never a neighbour: looking for a new scene's
number, it is passed over and the nearest scheme numbers before and after it in the script are the neighbours (10,
PROLOGUE, *new*, 11 → 10A; a new scene with no scheme number before it is lettered before the first one after it,
A110A before 110A, as A1 before 1). The free scenes after a fixed one count on from the scheme number before it.
The scheme's numbers are ordered only inside the letter generator (to find a free number between two neighbours);
nothing else compares labels.

**Lists by the script's order.** In the dialog's list a range is list syntax, never a scene's number: `X-Y` is
every scene from the one labelled X to the one labelled Y as the script has them, unnumbered scenes between included,
whatever the labels ("4-2" where the scene labelled 4 comes before the one labelled 2 is a forward range;
`I-1-A – PROLOGUE` works). If Y comes only before X, it is an error ("12-5: scene 5 comes before scene 12 in the
script"), not swapped: swapping scenes is a separate action, not renumbering. Range separators: `–`, `—`, `..`, `…`,
or a spaced ` - `; a lone `-` inside a word only when the word is no label of the script and it splits at exactly
one dash into two labels the script has (else an error asking for `–` or `..`); a word that splits into a label and a
scheme number no scene has (`1-99`) is an error (no scene 99); any other word no scene has is an unknown label,
named under the field (kept: reserved). With a label twice, a range runs from the first X to the first Y at or after
it. The same rule for the From … To pickers. `renumberWith` checks it too (nothing written, the reason in the status
bar).

**OMITTED runs.** The text always keeps one `.OMITTED #n#` a scene. Printing a run as one is a setting in the
production block, `omitted runs: print as one`, so it travels with the file; Renumber's OMITTED choice *Keep them,
print runs as one* sets it, *Keep them with their numbers* takes it out, *Drop* leaves it (Apply is enabled for a
change of the setting alone). With it, Lexington (`lex.MergeOmittedRuns`, `-mergeomitted`, `cli.Options.MergeOmitted`,
`pdf.PDFWriter.MergeOmitted`) prints two or more numbered OMITTED headings with only blank lines and boneyard between
them as one, `12-14   OMITTED   12-14` in both margins (checked with pdftotext), the first and last label in script
order joined by `-` (an en dash when a label has a dash: `I-1-A–I-1-C`); HTML, DOCX, ODT and the print the same; the
PDF's pages, their lock and the scenes' lengths are the merged print's (each OMITTED heading of a run gets the run's
page). FDX, Fountain and lex keep one heading a number.

**`.OMITTED #12-14#` is a label** (2026-11-20, Lexington v1.15.0, branch dropcompat). Joop dropped the v1.13.0
compatibility entirely: "nobody used that in production, and the undoing could cause legitimate - usage in scene
numbers to break down the road". Lexington no longer reads such a heading as a run (`lex.OmittedRange`,
`lex.SplitOmittedRanges`, `fountain.SplitOmitted`, the Fountain reader's split and `NumberOptions.Omitted` are gone);
Accolade no longer rewrites it on Renumber, Unlock, Save or Save As, moves a page lock off it, or splits it on Restore
Scene. It is one scene labelled "12-14", fixed like any dashed label: never split, renumbered, dropped or a
neighbour (a new scene after 3 and an OMITTED 4-5 gets 4). Restore Scene on it looks for a `/* Omitted 12-14` block
like any other. Printing a run of OMITTED headings as one stays: a print setting, not a text form. A stray number
typed inside a heading (`INT. A #1-A# - DAY`) is any label.

**Tests.** Lexington: the scheme (every Fountain example form), fixed labels through locked and unlocked numbering
and every Renumber strategy (table and the 3000-script property test: a fixed label never changes, is never dropped
or given; nothing given out of the scheme), `.OMITTED #12-14#` a fixed label through reading, numbering, FDX and
Fountain, the lists (4-2 forward, 2-4 an error,
separators, a label winning over a split, an ambiguous dash, unknown labels, Read), `MergeOmittedRuns`, and through the
command line: one `12-14 OMITTED 12-14` line in the PDF (pdftotext), one OMITTED in
HTML, DOCX and ODT, three without the flag, FDX one heading a number with five SceneProperties, Fountain never merged.
Accolade: fixed labels through Lock, seven strategies, Clean and Remove; the backwards scope and list, and an end no
scene has, refused; a keep range by script order; an OMITTED 4-5 a fixed label; print runs as one in the preview, the read and error lines under the field, Apply for the
setting alone, the HTML print, the FDX and the PDF (pdftotext), Keep taking it out, Undo. Shots: `SHOT_DIR=… go test
-run TestRenumberDialogScreenshots` (renumber-wide.png shows print runs as one and a read list).

**Rough**

- The pre-lock check's rows do not offer Unlock / Renumber, only the pane's buttons do.
- A tag's anchor keeps the old number among its words before and after (`NIGHT #3# The-`). These only rank places,
  so it is still found, a little less surely.
- In the test renderer's shots the numbers field's text shows outside the dialog's scrolled controls: Fyne's software
  painter sets a nested scroller's clip to its own area instead of intersecting it with the outer one
  (internal/driver/util.go walkObjectTree); the GL painter intersects (ClipStack). A capture artefact.
- Renumber's choice of a range is kept by scene index, so Last used on another script means other scenes.
- A merged OMITTED run's page can anchor its page lock to "12-14" (a page starting at it); switching the setting off
  then finds that page by its words.
- "OMITTED 12-14" as one line is the form I know from shooting scripts (*unverified*: no source checked).
- The scheme is narrower than the old engine: `01`, `10a`, `12b` are fixed labels now (never renumbered).
- After an issue, Renumber marks every renumbered heading as revised in the next colour. That is correct (the
  headings changed), but it is a lot of asterisks.
