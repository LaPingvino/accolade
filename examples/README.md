# The sample script

`sample.accolade` is the script File > Open Sample opens (the ribbon: File > New > Sample), in the desktop app and
in the browser: a script to try every feature on. It is embedded in the app (`sample.go`), and opened as a copy of
its own (the data folder's `samples/`, the browser's `/files/`), so it can be changed freely; Open Sample again gives
a fresh copy beside a changed one.

It is generated, not written by hand. `sample_test.go` builds it from the two files here through Accolade's own
commands, as a writer would use them:

- `a-chameleon.md`: the story, the source, with front matter (`title:`, `author:`, `translator:`, `rights:`).
- `a-chameleon.fountain`: the adaptation as typed, with `{{n}}` where a link to the source's passage n goes and
  `{{i-j}}` where one spans passages i to j.

```
UPDATE_SAMPLE=1 go test -run 'TestMakeSample$' .   # writes examples/sample.accolade
go test -run 'TestSample' .                         # checks it, and checks a freshly made one
```

The generator saves the script as an .accolade, chooses the source (kept in the bundle), puts in each link with
Link to Source Passage, marks a passage as cut, fills in the Characters pane, writes what Accolade learned and the
Notes, saves a draft, locks the scene numbers, omits a scene, types a scene in after the lock, locks the pages, tags
the breakdown, puts the scenes on two shoot days, issues White, makes Blue's changes and issues Blue, then saves.
`TestSample` opens the embedded file as File > Open Sample does, in a window with new settings and data, and checks
each feature below; `TestSampleFresh` makes the sample again and checks that, so the generator keeps working as the
formats change. When it fails because a format changed, make the sample again with `UPDATE_SAMPLE=1`.
The file is not byte for byte the same each time it is made: the zip's times and the drafts' "kept on" dates are
the day it was made.

## The story, and why it is free to use

"A Chameleon" by Anton Chekhov, in Constance Garnett's English translation, from *The Cook's Wedding and
Other Stories* (The Tales of Chekhov, vol. 12). The text is Project Gutenberg's eBook #13417, fetched from
gutenberg.pglaf.org (`cache/epub/13417/pg13417.txt`; its header names itself "The Project Gutenberg eBook of The
Cook's Wedding and Other Stories" and points to www.gutenberg.org/ebooks/13417). The eBook's page on
www.gutenberg.org could not be fetched (it answered 503, retry after 6 hours; the Wayback Machine 429). The
catalogue record beside the text (`cache/epub/13417/pg13417.rdf`) says:

> `<dcterms:rights>Public domain in the USA.</dcterms:rights>`

Project Gutenberg's licence and header are left out, as its licence allows for a public-domain text. Outside the
USA, check your own country's law before using it. The script's Notes say the same.

## What it shows

| Feature | In the sample |
|---|---|
| Title page | Title, Credit, Author, Source, Draft date |
| Source kept in the bundle | `assets/source.md` (so the source in the middle and the Source pane work) |
| The source's front matter | `title`, `author`, `translator`, `rights` |
| Source pane | 31 passages, every link found |
| A link in Text Fragment form | `[[from: #:~:text=…]]` |
| A range link | `textStart,textEnd` within one passage |
| A link over two passages | Prohor's arrival, passages 22–23 |
| A twin told apart by a prefix | `"Yes."` also opens `"Yes, your honour."`: `text=visit?-,"Yes."` |
| Passages cut | "Do you know that for a fact?" and "Yes, your honour." |
| Passages still to adapt | the last two, for Next Source Passage |
| Learned: an alias | `/* Adaptation */`: YELDYRIN = the policeman |
| Learned: not characters | Petersburg, Moscow |
| Learned: a place | the timber-yard = EXT. PITCHUGIN'S TIMBER-YARD |
| Characters block | notes and the fields age, look, arc |
| An alias in the Characters pane | Otchumyelov: "the superintendent" |
| A listed name who doesn't speak | General Zhigalov |
| Notes | the sample's notes, with the story's public-domain status |
| Scene numbers locked | `scene numbers: locked` |
| An OMITTED scene | 3, the tavern, its text kept in boneyard |
| An A-scene | 1A, written after the lock |
| Drafts | a writer's draft ("First draft"), the White and Blue issues |
| Revision marks | Blue's changes |
| Locked pages with an A-page | page 3A, pushed by Blue's longer paragraph |
| Page Setup in the file | the page lock says the format and paper it was locked in |
| Breakdown tags | Wardrobe, Props, Animals, Makeup/Hair, Background Actors |
| A lost tag | Prohor's basket, gone in Blue |
| A schedule over two days | Day 1 the square and the timber-yard, Day 2 the General's yard |
| Source in the Middle | the source as a middle tab beside the script |

One row of the table is one check in `sampleFeatures` (`sample_test.go`).

## What it does not show

- **Page Setup's own settings** (paper, title page, scene numbers, CONTINUEDs, colours, revision marks, pagination)
  are the machine's (`settings.json`), not the file's: only the script format (when it is one of your own) and
  what the page lock recorded travel with the script.
- **A script language other than English**: no `text.<language>.fountain` beside `text.fountain`, no
  `"language"` in `info.json`, no force marks for another language's scene words, no right-to-left lines. The
  bundle does carry English's part of Lexington's settings (`assets/lexington/lexington.toml`).
- **A script format of one's own** (`assets/lexington/formats/`), and the language rules' `list` cue rule.
- **A Markdown document**: the sample is a screenplay; its source is the Markdown (with front matter).
- **The other ways to keep a source**: in the script (`/* Source */`), kept by Accolade, beside the script.
- **Collaboration**: nothing of a shared session.

# The samples in other languages

`sample.<lang>.fountain` is a short scene (about a page) in each language of Lexington's default configuration that
has one, written in that language: its own scene words, cues, an extension, a parenthetical, a transition and title
page keys, each as Lexington's rules for the language have them, so it reads back as written
(`TestLanguageSamplesRead` in `sample_rtl_test.go` checks every one: headings, cues, transitions and the title page).
They are embedded in the app (`sample.go`, every `sample.*.fountain` here); New in that language starts from it, and
File > Open Sample lists them all. A language without one gets the English template in its words.

Each is original writing on a public-domain story or a historical setting of the language's culture. **All are
machine-written, to be checked by a native reader** (Esperanto: Joop).

| file | language | story |
|---|---|---|
| `sample.eo.fountain` | Esperanto | Bulonjo-ĉe-Maro, August 1905, the first Universala Kongreso: delegates who learned the language alone meet at the door; Zamenhof backstage with Klara before his speech |
| `sample.nl.fountain` | Dutch | Rembrandt's studio, 1642: Captain Banninck Cocq poses for the Night Watch, a militiaman complains about his place |
| `sample.de.fountain` | German | the Bremen Town Musicians (Grimm): the donkey meets the dog on the road; the four scare the robbers |
| `sample.fr.fountain` | French | the crow and the fox (La Fontaine) |
| `sample.it.fountain` | Italian | Pinocchio (Collodi): Geppetto carves the puppet, who runs off into the street |
| `sample.ru.fountain` | Russian | Krylov's Quartet: the monkey, the donkey, the goat and the bear change seats to play better |
| `sample.tok.fountain` | toki pona | the ant and the grasshopper (Aesop): toki pona has no stories of its own yet |
| `sample.zh.fountain` | Chinese | 守株待兔 (Han Feizi): the farmer who waits by the stump for another hare |
| `sample.ja.fountain` | Japanese | 桃太郎: the peach on the river, the boy born from it |
| `sample.he.fountain` | Hebrew | a mother reads the tower of Babel to her son (Genesis 11:1–4, pointed as by Mechon Mamre, no cantillation) |
| `sample.ar.fountain` | Arabic | Ali Baba and the Forty Thieves: "Open Sesame" (افتح يا سمسم, vowelled) |
| `sample.hi.fountain` | Hindi | the monkey and the crocodile, English first as most Hindi scripts are written: English Fountain's words (`EXT. RIVERBANK, JAMUN TREE - DAY`, `Title:`, `(V.O.)`, `CUT TO:`), the action and dialogue in Hindi in Latin letters |
| `sample.hi-Deva.fountain` | Hindi in Devanagari | the monkey and the crocodile (Panchatantra, book 4): the crocodile's wife wants the monkey's heart; in mid-river the monkey says he left it in the jamun tree |
| `sample.hi-Latn.fountain` | Hindi in Latin letters | the same in Hindi's words in Latin letters where it has them (the time `DIN`, the title keys `Sheershak:`…), English Fountain's for the rest (`EXT.`, `CUT TO:`, `(V.O.)`) |

Where a language's rules have no word of their own, the sample uses what the rules give it: Esperanto and toki pona
have English's extensions (`O.S.`), and Esperanto, toki pona, Dutch and Russian end on a forced transition
(`> MALLUMIĜO.`), as they have no whole-line transitions. The title page keys are Lexington's (its `TitleKeys`; for
all but Arabic, Lexington's own choice; Hindi's extension `वॉयस ओवर` and its transitions are sourced, the rest Lexington's
own: its docs/LANGUAGES.md says which). Lexington's PDF font has no Chinese, Japanese or Devanagari: those three print through
HTML. Devanagari is also not yet shaped in the editor (its vowel signs stand apart, its conjuncts broken).
