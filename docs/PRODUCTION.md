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
- Accolade numbers the unnumbered scenes at export: the Export dialog's "Number the scenes (those without a
  number)" (export_dialog.go:128). It counts on from the last explicit number, so after 7 comes 8. Since step 1
  (§7) this is Lexington's `lex.NumberScenes`, and Production > Lock Scene Numbers, A-scenes and OMITTED exist.

**Cards, outline and title page**

- View > Cards (cards.go) and View > Outline (outline.go, outline_move.go): drag to reorder, colours, folding.
  This is Accolade's counterpart to Final Draft's Beat Board and Story Map, and the base for a stripboard.
- File > Title Page… (title_page_dialog.go): where issued revision sets would be listed.

**Pagination**

- Lexington paginates as screenplay software does: keep-with-next, widows and orphans, and `(MORE)` / `NAME
  (CONT'D)` in the PDF. This is Lexington 1.6.0 (CHANGELOG) and the Export dialog's page-break choice
  (export_pagination.go).
- Missing: the scene `CONTINUED:` / `(CONTINUED)`, locked pages, A-pages, and running headers other than the page
  number (pdf/create.go:483-487 prints only `n.`).

**Colours and tags**

- A note on a heading or section, `[[red]]` or `[[#3366ff]]` as Beat writes it, colours the outline, the cards and
  the export.
  - Accolade: Format > Scene or Section Color… (Ctrl+Shift+L, scene_colors.go).
  - Lexington: `-colors`, `-colorstyle`, `-colorscope` and patterns.
  - FDX keeps the colour as SceneProperties Color (fdx/colors.go).
- Any note on a heading or section is a tag. The Export dialog's "Only scenes and sections tagged" exports just
  those scenes (export_filters.go:18-60, 133-189; Lexington's `lex.FilterTags`, `-tag`).
- These are scene-level tags only. There are no breakdown categories and no tagged words in the text.

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
  CSV and as PDF. Breakdown sheets wait for tagging.

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
- FDX has no `<Revisions>`, `<LockedPages>`, `<TagData>` or `<SceneProperties Length=…>`.

**Bundles**

- Accolade opens and saves .highland, .textpack and .textbundle (bundle.go, docs/HIGHLAND.md).
- It copies every unknown entry raw, keeps its own state under info.json's `"eu.kiefte.Accolade"` and keeps the
  adaptation source at `assets/source.md`. This is the natural container for production data (§4).

**Not handled: Beat's data block**

Accolade doesn't read Beat's data block, the trailing `/* If you're seeing this … BEAT: {…} END_BEAT */`. It stays in
the text as boneyard. Its offsets go stale as soon as Accolade edits the text (§3).

## 2. What the field does

### Production revisions

**Convention**

- The page colours run white, blue, pink, yellow, green, goldenrod, buff, salmon, cherry, tan, ivory, then
  "double white", "double blue" and so on (https://en.wikipedia.org/wiki/Shooting_script; Final Draft cites WGA
  West for the same order: https://kb.finaldraft.com/hc/en-us/articles/15575314119316).
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
- **Movie Magic Scheduling** imports .fdx directly and makes breakdown sheets from its scenes and tags, adding
  categories it lacks (https://mms-docs.ep.com/Breakdown/ImportingBreakdownSheets.html).
  - Its other route, `.sex`, is closed. Fade In: "a closed, proprietary format", so it doesn't write it
    (https://fadeinpro.com/kb/content/1/85/en/export-to-movie-magic-scheduling-or-movie-magic-budgeting.html).
  - Final Draft seems to say tags map fully only through .sex. This comes from a search snippet of
    https://kb.finaldraft.com/hc/en-us/articles/15575252743316-Can-Final-Draft-export-to-Movie-Magic-Scheduling-and-Budgeting;
    the page itself returned 403. *Test* tagged FDX against MMS.
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
| `pages.json` | locked pages: page number to its first words in the locked draft, with A-pages | JSON (own, after FDX `<LockedPage>`, anchored by words) |
| `breakdown.json` | tag definitions in FDX's categories (with their UUIDs and numbers), and each tag's scene number and quoted words | JSON (own; FDX/OSF categories) |
| `schedule.json` | shoot days, strips in order, day breaks, banners, units | JSON (own) |
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
     (`rules.Pagination` with locks), Accolade for Lock Pages and Unlock.
4. **Breakdown tagging.**
   - What: select words and tag them in a category (Cast fills itself from the cues, Location from the heading).
     Each scene gets its eighths, D/N, INT/EXT and script day, and a breakdown sheet.
   - Who has it: Final Draft Tagger, Fade In, WriterDuet, Beat, StudioBinder, Celtx.
   - Representation: breakdown.json by words.
   - FDX: write `<TagData>` and `TagNumber` as Beat does, and `<SceneProperties Length Page>`. That is the route
     into Movie Magic, Gorilla, StudioBinder and Yamdu. Read FDX and Beat tags.
   - Effort: M. Where: Accolade (a Breakdown pane, the editor's tint for tagged words); Lexington (eighths from the
     layout, FDX TagData).
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
     PDFs).
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

The **Production pane** (View > Production, a tab beside Notes and Source) shows the stage and one count per stage:

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

   Done marks it done.
4. **Schedule.** "18 of 87 scenes on a day · 3 days". The stripboard, with the unscheduled strips at the end and
   Next Day.
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
   only. FDX `<Revisions>` and `RevisionID` out, and in as marks.
5. **Locked pages and A-pages** in Lexington's paginator, with `CONTINUED`. FDX `<LockedPages>`.
6. **The Production pane** with Lock and Revise, so the workflow stands before breakdown starts.
7. **Breakdown:** the categories, tagging by words, Next scene, eighths, breakdown sheets and element reports. FDX
   `<TagData>` and `<SceneProperties>` out (test against MMS), Beat's and FDX's tags in.
8. **Schedule:** the stripboard (built on the cards), days, one-liner, DOOD, .ics and CSV.
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
    letters only scenes "inserted between previously numbered scene headings". A number that is not digits and
    letters (Storyboarder's `3-x2k9q`) is no neighbour, but is never given again.
- `lex.WithSceneNumber` puts `#n#` in a heading before its notes. `-number count|10A|A11` on the command line.
- `fountain.OmitScene` / `fountain.RestoreScene` (and `lex.OmittedNumber`): `.OMITTED #12#`, a blank line, then
  `/* Omitted 12`, the scene exactly as it was (heading with its number and notes included), `*/`. A `*/` inside is
  kept as `* /`, as the notes block does (so a literal `* /` in an omitted scene comes back as `*/`).
- FDX: the number stays `Number="12A"`; OMITTED is a Scene Heading `<Text>OMITTED</Text>` with `Number="12"`, as
  OpenDraft's golden file writes it (*unverified* against Final Draft itself). It reads back as `.OMITTED #12#`. No
  `<SceneNumberOptions>` is written: I have seen no real file's values. Fixed on the way: FDX wrote Fountain's
  boneyard (notes blocks too) as General paragraphs, which Final Draft prints; it is left out now. So an omitted
  scene's text does not survive FDX: Restore needs the Fountain (or the bundle).

**Accolade** (production.go; a Production menu, hidden for Markdown; no pane yet: that is step 6)

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
- **Unlock Scene Numbers…** asks, then takes the lock line out (the block too if empty), nothing else: the numbers
  stay, as they went out with the script. Lock's own Undo takes its numbers away again; a later "remove the numbers"
  would need to know which were Lock's, which nothing records.
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
  them). The status bar says which.
- The browser's Save As now asks for a name (the download's), so it can change the kind too.
- **The Highland warning:** saving a `.highland` that holds Accolade's files (drafts, the source) — or saving the
  first draft into one — saves, then says once that Highland may drop them and that a copy as `.accolade` keeps
  them safe. `highlandWarned` in the key makes it once per file (and if Highland drops the key, it warns again,
  which is right). Untested with a real Highland: that is the test still to do.

**Bundle writes in the background** (bundle_queue.go, PERF-SOURCE item 7): every write into a bundle (Save,
auto-save, the source's Done, Save Draft) is a job, run one after another in a goroutine, each from the bundle the
job before left; the zip loop calls `pace.Yield` for the browser's page. A failed save makes the script unsaved
again and says why. Rename and closing the window wait for the writes; the browser's download follows the write.
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

- **File > Save Draft…:** a name (default "Draft n"), at any time, before a lock too; says where they are kept.
- **File > Drafts…:** the list, newest first (name; date, "same text as …", "source changed"), Compare,
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
