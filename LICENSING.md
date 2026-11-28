# Licensing

Accolade's own code is offered under either license, at your choice:

- the MIT License ([LICENSE-MIT](LICENSE-MIT)), or
- the GNU General Public License, version 3 or later ([LICENSE-GPL](LICENSE-GPL)).

SPDX: `MIT OR GPL-3.0-or-later`. Every Go file says so on its first line
(`TestSPDXHeaders` checks it), as do the scripts. The items listed under
"Not Accolade's own" below keep their own licenses.

The offer covers the files in the current tree. Accolade began as a fork of
Apostrophe (formerly UberWriter, GPL-3.0-or-later); the commits from before
the Go rewrite, Apostrophe's and Accolade's GTK version, stay in git history
under GPL-3.0-or-later only. AUTHORS and the About dialog credit Apostrophe's
authors.

## The audit (2026-11-28)

Before the MIT offer, the tree at origin/main (ae2e6c5) was checked for
anything that is Apostrophe's or another person's. Sources compared:

- Apostrophe's code at the fork point (dff1c5e, 2024-05-01, its last commit
  in this repository) and all of its history up to then;
- the GTK version's files that 420261d ("Remove the GTK version's
  leftovers") deleted;
- Apostrophe's current code on GitLab GNOME (World/apostrophe, 7fe8f1d,
  2026-11-21).

Checks:

1. **Identical files.** Every current blob against every blob in Apostrophe's
   history, in the deleted GTK files and in upstream's current tree.
2. **Copied text.** Every distinct line of 30 characters or more from those
   sources (code, UI files, CSS, help, metainfo, desktop files, templates,
   translations' template) searched for in every current text file. The only
   hits were boilerplate: XML declarations, SVG and ODF namespace lines, a
   Markdown table rule, and the items listed below.
3. **Line authorship.** `git blame -w -M -C` over every current file. All
   lines are Joop Kiefte's (his agents' commits carry his name, with
   Claude as Co-Authored-By) except the items listed below. About 1500 of the
   2303 commits are by Apostrophe's authors and translators; none of their
   code, translations or styles is in the current tree.
4. **Icons.** The app icon (packaging/icon.svg, Icon.png, web/icon-*.png) is
   Accolade's own drawing from 2026-10-24, not Apostrophe's (whose SVGs are
   in history only).
5. **Ported code.** The Go code has no Apostrophe identifiers, regular
   expressions or comments; no Go file mentions a source it was ported from.

What the audit found from Apostrophe and its contributors, and what became
of it:

| Item | Origin | Done |
|------|--------|------|
| screenshots/focus, footnote, formula, preview, search, table .png | Apostrophe's screenshots of its own UI (Manuel Genovés, 2018–2024), unused | removed |
| .gitlab/issue_templates/*.md | Apostrophe's GitLab issue templates (Manuel Genovés), unused on GitHub | removed |
| NEWS | Apostrophe's release notes (1.0–3.0) | removed (Accolade's are in docs/RELEASE-*.md and the GitHub releases) |
| accolade.doap | DOAP skeleton from apostrophe.doap, used by nothing | removed |
| .gitignore | a few patterns from Apostrophe's | rewritten |
| README.md | a few blank lines and the "About"/"Building" headings | trivial; no other content |
| LICENSE | the GPL-3.0 text itself | now LICENSE-GPL, in its current wording from gnu.org |
| AUTHORS | Joop's credit of Apostrophe's authors | kept |

No Apostrophe-derived file remains, so nothing in the current tree is
GPL-3.0-or-later only.

## Not Accolade's own

These keep their own licenses and are not offered under MIT by Accolade:

| Item | License |
|------|---------|
| icons.go: the Material Design icons' SVG path data | Apache-2.0 (Google); the file's SPDX line is `(MIT OR GPL-3.0-or-later) AND Apache-2.0` |
| internal/cite/csl/*.csl, locales-en-US.xml | CC-BY-SA-3.0, Citation Style Language project (see CREDITS there) |
| internal/prose/fonts/NotoSans*.ttf | OFL-1.1 (OFL-*.txt beside them) |
| testdata/hekaya/ | MIT, Hekaya Project (LICENSE there) |
| testdata/textbundle/ | MIT, Shinyfrog (LICENSE there) |
| CODE_OF_CONDUCT.md | Contributor Covenant 1.4 (CC-BY-4.0) |
| examples/a-chameleon.*, testdata/adapt-bench/*, internal/storyread/testdata/* | public-domain literature (Project Gutenberg), adapted where marked |

The Go modules Accolade builds with (go.mod) are under their own permissive
licenses (BSD, MIT, ISC, Apache-2.0, Unlicense), none copyleft, so a binary
built from this tree may be distributed under either of Accolade's licenses
together with theirs. One indirect module, github.com/danlock/pkg (through
gogosseract, Lexington's OCR), ships without a license file.
