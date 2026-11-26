package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The adaptation benchmark's synthetic pair (testdata/adapt-bench/
// synth-carol/): a screenplay made to measure from the Christmas Carol
// pair's source, with the cases the period plays have too few of, each
// scene tagged with its kind in mapping.tsv's seventh column:
//
//   - merged-near: two stretches of one chapter with a stretch between
//     left out, played as one scene;
//   - merged-far: two stretches staves apart (one of them out of the
//     story's order) played as one scene;
//   - out-of-order: a scene from the end played early (a flash-forward);
//   - reword-light: lifted, its narration cut to a sentence a passage and
//     common words swapped;
//   - reword-heavy: written anew (by hand, below), the names and a line or
//     two kept, as a free adaptation does;
//   - merged-reworded: two stretches, written anew as one scene;
//   - invented: from nowhere in the story.
//
// The script and mapping are generated here, from source.md and the
// scenes below: ACCOLADE_WRITE_SYNTH=1 go test -run TestAdaptBenchSynth
// writes them; without it the test checks the committed files are what
// it would write. The source is the Christmas Carol pair's (source.ref).
func TestAdaptBenchSynth(t *testing.T) {
	dir := filepath.Join("testdata", "adapt-bench", "synth-carol")
	src, err := os.ReadFile(filepath.Join("testdata", "adapt-bench", "christmas-carol", "source.md"))
	if err != nil {
		t.Fatal(err)
	}
	script, mapping := synthCarol(passagesOf(string(src)))
	files := map[string]string{"script.fountain": script, "mapping.tsv": mapping, "source.ref": "../christmas-carol/source.md\n"}
	for _, name := range []string{"script.fountain", "mapping.tsv", "source.ref"} {
		path := filepath.Join(dir, name)
		if os.Getenv("ACCOLADE_WRITE_SYNTH") != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(files[name]), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil || string(b) != files[name] {
			t.Errorf("%s is not what the generator writes (ACCOLADE_WRITE_SYNTH=1 to write it)", path)
		}
	}
}

// synthScene is one scene of the synthetic script: lifted from ranges
// (light: reworded lightly), or written by hand (text) for them.
type synthScene struct {
	heading, kind, note string
	ranges              [][2]int
	speakers            []string // lifted: who speaks, the first named in a passage's narration, else by turns
	light               bool
	text                string
}

var synthCarolScenes = []synthScene{
	{heading: "INT. SCROOGE AND MARLEY'S COUNTING-HOUSE - DAY", kind: "merged-near",
		note:   "the nephew's visit and the two gentlemen, lifted; Bob's aside and the gentlemen's coming in left out between",
		ranges: [][2]int{{11, 42}, {46, 59}}, speakers: []string{"SCROOGE", "NEPHEW", "GENTLEMAN"}},
	{heading: "INT. SCROOGE'S ROOMS - NIGHT", kind: "reword-light",
		note:   "Marley's ghost comes in, lifted with the narration cut and common words swapped",
		ranges: [][2]int{{101, 111}}, speakers: []string{"SCROOGE", "GHOST"}, light: true},
	{heading: "INT. AN EMPTY SCHOOLROOM - NIGHT", kind: "reword-heavy",
		note:   "the boy reading alone, Ali Baba and the parrot, Fan fetching him home: written anew",
		ranges: [][2]int{{231, 246}}, text: `The SPIRIT and SCROOGE stand at the back of a cold, bare classroom. A lonely BOY reads by a weak candle. Outside the glass, figures from his book drift past: a robed traveller leading a donkey loaded with wood.

SCROOGE
Ali Baba! I'd know him anywhere. He turned up just like that, the year they left me here on my own.

A green bird with a yellow tail flaps across the window.

SCROOGE (CONT'D)
And the parrot! Robin Crusoe's bird. "Where have you been, Robin?"

He laughs, then his face crumples.

SCROOGE (CONT'D)
That poor lad.
(fumbling in a pocket)
A child sang at my door last evening. I sent him off with nothing. I wish I hadn't. Too late now.

SPIRIT
Then let us look at another year.

The room ages around them: cracked panes, plaster on the floor. The BOY, older now, paces. A small GIRL bursts in and throws her arms round him.

FAN
I've come to take you home! Father's gentle now, he said yes. You're never coming back here.

BOY
Fan, you sound like a grown lady.`},
	{heading: "INT. FEZZIWIG'S WAREHOUSE - NIGHT", kind: "merged-reworded",
		note:   "Fezziwig's ball and the broken engagement, two stretches of stave II played in one set and written anew",
		ranges: [][2]int{{256, 271}, {286, 299}}, text: `Shutters up, desks pushed to the walls. A fiddler tunes. OLD FEZZIWIG, round and red-faced, claps his hands at his two apprentices, young EBENEZER and DICK WILKINS.

FEZZIWIG
Enough ledgers, boys! It's Christmas Eve. Clear the floor!

Neighbours pour in, couples spin, there is cake and cold roast and more dancing. SCROOGE watches from the shadows, grinning in spite of himself.

Later. The guests gone, the lamps low. Young EBENEZER sits apart with a girl in black, BELLE. Her eyes are wet.

BELLE
Something else has taken my place in your heart. If it comforts you better than I could, I won't complain.

EBENEZER
Taken your place? What could?

BELLE
Money.

EBENEZER
The world punishes nobody like the poor, and then it sneers at anyone trying not to be.

BELLE
We promised each other when we had nothing. You were someone else then.

EBENEZER
I was a child.

BELLE
I let you go. Be happy with the life you picked.`},
	{heading: "EXT. A LONDON STREET - CHRISTMAS MORNING", kind: "out-of-order",
		note:   "flash-forward: stave V's boy at the window, lifted and played before the ghosts of stave III",
		ranges: [][2]int{{656, 666}}, speakers: []string{"SCROOGE", "BOY"}},
	{heading: "EXT. CITY STREETS - CHRISTMAS DAY", kind: "reword-heavy",
		note:   "the Spirit's torch over the dinners and the Sabbath closing: written anew",
		ranges: [][2]int{{353, 366}}, text: `Snow on the rooftops. Families hurry along with covered dishes for the bakers' ovens. The SPIRIT OF CHRISTMAS PRESENT, a huge green-robed figure, shakes sparks from a torch over the dishes; tempers soften wherever they land.

SCROOGE
What is it you're sprinkling? Has it a taste?

SPIRIT
It has mine.

SCROOGE
On any meal?

SPIRIT
On any shared kindly. On a poor man's most.

SCROOGE
Why that one?

SPIRIT
Because he needs it most.

SCROOGE
Odd, then, that your sort want the bakers shut every Sunday. That's the one hot dinner these people get.

SPIRIT
Mine? Never.

SCROOGE
It's done in your name.

SPIRIT
There are people who claim to know us and use our name for their own spite and pride. Blame them, not us.`},
	{heading: "INT. RAILWAY WAITING ROOM - NIGHT", kind: "invented",
		note: "invented: a stationmaster and a porter wait for a late train; nothing in the story",
		text: `A gas lamp hisses. A STATIONMASTER winds his watch while a PORTER stacks parcels by the stove.

STATIONMASTER
The nine-fifteen is held at Reading again. Signals frozen.

PORTER
Then I'll brew another pot. The parcels for the vicarage can wait till morning.`},
	{heading: "INT. THE CRATCHITS' KITCHEN - CHRISTMAS DAY", kind: "merged-far",
		note:   "the Cratchits waiting for Bob and Tim, then the two children under the Spirit's robe from the end of the stave, lifted",
		ranges: [][2]int{{369, 383}, {464, 469}}, speakers: []string{"MRS CRATCHIT", "MARTHA", "BOB", "SCROOGE", "SPIRIT"}},
	{heading: "INT. SCROOGE AND MARLEY'S COUNTING-HOUSE - MORNING", kind: "merged-far",
		note:   "Bob late on Boxing Day, with stave I's passage on the old sign over the door cut in before it (out of the story's order), lifted",
		ranges: [][2]int{{6, 6}, {703, 712}}, speakers: []string{"SCROOGE", "BOB"}},
}

// synthLight is the light rewording: common words for others (whole
// words, a capital kept), in this order.
var synthLight = [][2]string{
	{"said", "told him"}, {"replied", "answered"}, {"cried", "called"}, {"asked", "wondered"},
	{"looking", "staring"}, {"looked", "stared"}, {"very", "quite"}, {"little", "small"},
	{"great", "big"}, {"spirit", "phantom"}, {"ghost", "spectre"}, {"door", "doorway"},
	{"room", "chamber"}, {"chair", "seat"}, {"eyes", "gaze"}, {"head", "skull"},
	{"believe", "trust"}, {"want", "wish"}, {"know", "see"}, {"never", "not ever"},
}

var synthQuote = regexp.MustCompile(`"[^"]*"`)

// synthLift makes passages into Fountain: their quoted speech dialogue,
// the rest action (light: its first sentence only, and reworded).
func synthLift(ps []passage, sc synthScene) string {
	var b strings.Builder
	turn := 0
	for _, rg := range sc.ranges {
		for i := rg[0]; i <= rg[1]; i++ {
			text := ps[i].Plain
			quotes := synthQuote.FindAllStringIndex(text, -1)
			var narr []string
			last := 0
			for _, q := range quotes {
				narr = append(narr, text[last:q[0]])
				last = q[1]
			}
			narr = append(narr, text[last:])
			action := strings.TrimSpace(strings.Join(narr, " "))
			action = strings.Join(strings.Fields(action), " ")
			if len(quotes) == 0 {
				if sc.light {
					action = firstSentence(action)
				}
				fmt.Fprintf(&b, "%s\n\n", synthWord(action, sc.light))
				continue
			}
			who := ""
			at := len(action) + 1
			for _, s := range sc.speakers {
				name := strings.SplitN(s, " ", 2)
				w := name[len(name)-1]
				// a name's first mention not as an owner ("Scrooge's nephew")
				low := strings.ToLower(action)
				for k := strings.Index(low, strings.ToLower(w)); k >= 0; {
					if !strings.HasPrefix(low[k+len(w):], "'s") {
						if k < at {
							who, at = s, k
						}
						break
					}
					n := strings.Index(low[k+1:], strings.ToLower(w))
					if n < 0 {
						break
					}
					k += 1 + n
				}
			}
			if who == "" {
				who = sc.speakers[turn%len(sc.speakers)]
				turn++
			}
			var speech []string
			for _, q := range quotes {
				speech = append(speech, strings.Trim(text[q[0]:q[1]], `"`))
			}
			line := strings.Join(strings.Fields(strings.Join(speech, " ")), " ")
			fmt.Fprintf(&b, "%s\n%s\n\n", who, synthWord(line, sc.light))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func firstSentence(s string) string {
	if k := strings.IndexAny(s, ".!?;"); k >= 0 {
		return s[:k+1]
	}
	return s
}

func synthWord(s string, light bool) string {
	if !light {
		return s
	}
	for _, sw := range synthLight {
		re := regexp.MustCompile(`(?i)\b` + sw[0] + `\b`)
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			if m[0] >= 'A' && m[0] <= 'Z' {
				return strings.ToUpper(sw[1][:1]) + sw[1][1:]
			}
			return sw[1]
		})
	}
	return s
}

// synthStart is how the mapping names a passage: its first words.
func synthStart(p passage) string {
	f := strings.Fields(p.Plain)
	return strings.Join(f[:min(6, len(f))], " ")
}

func synthCarol(ps []passage) (script, mapping string) {
	var s, m strings.Builder
	s.WriteString("Title: A Christmas Carol (synthetic)\nCredit: made to measure from\nAuthor: Charles Dickens\nNotes: Generated by adapt_bench_synth_test.go for Accolade's adaptation benchmark: merged, reworded, out-of-order and invented scenes.\n\n")
	m.WriteString("# scene\tsource\tconfidence\tfirst\tlast\tnote\tkind\n")
	for _, sc := range synthCarolScenes {
		body := sc.text
		if body == "" {
			body = synthLift(ps, sc)
		}
		fmt.Fprintf(&s, "%s\n\n%s\n\n", sc.heading, body)
		src, first, last := "none", "", ""
		if len(sc.ranges) > 0 {
			var parts []string
			for _, r := range sc.ranges {
				parts = append(parts, fmt.Sprintf("%d-%d", r[0], r[1]))
			}
			src = strings.Join(parts, ",")
			first = synthStart(ps[sc.ranges[0][0]])
			last = synthStart(ps[sc.ranges[len(sc.ranges)-1][1]])
		}
		fmt.Fprintf(&m, "%s\t%s\tsynthetic\t%s\t%s\t%s\t%s\n", sc.heading, src, first, last, sc.note, sc.kind)
	}
	return s.String(), m.String()
}

// The synthetic pair's flash-forward, lifted near word for word but out
// of the story's order: placed by the third look, marked as such, and
// the invented scene still silent.
func TestAlignUnordered(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "adapt-bench", "christmas-carol", "source.md"))
	if err != nil {
		t.Fatal(err)
	}
	ps := passagesOf(string(src))
	script, _ := synthCarol(ps)
	gs := alignScenes(scenesOf(script, []string{"INT.", "EXT.", "INT./EXT.", "I/E"}), ps, nil)
	for _, g := range gs {
		switch {
		case strings.Contains(g.Heading, "LONDON STREET"):
			if !g.Unordered || g.First > 666 || g.Last < 656 {
				t.Errorf("the flash-forward: %+v", g)
			}
		case strings.Contains(g.Heading, "RAILWAY"):
			if g.guessed() {
				t.Errorf("the invented scene guessed: %+v", g)
			}
		case g.Unordered:
			t.Errorf("%s out of order: %+v", g.Heading, g)
		}
	}
}

// A script written as far as stave III, with the flash-forward from stave
// V before its last scene: stave IV is not yet, not cut.
func TestUnorderedNoCuts(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "adapt-bench", "christmas-carol", "source.md"))
	if err != nil {
		t.Fatal(err)
	}
	ps := passagesOf(string(src))
	script, _ := synthCarol(ps)
	cut := strings.Index(script, "INT. FEZZIWIG'S")
	from := strings.Index(script, "EXT. A LONDON STREET")
	to := strings.Index(script, "EXT. CITY STREETS")
	kitchen := strings.Index(script, "INT. THE CRATCHITS'")
	end := strings.Index(script, "INT. SCROOGE AND MARLEY'S COUNTING-HOUSE - MORNING")
	script = script[:cut] + script[from:to] + script[kitchen:end]
	starts := []string{"INT.", "EXT.", "INT./EXT.", "I/E"}
	gs := alignScenes(scenesOf(script, starts), ps, nil)
	if !gs[3].Unordered || !gs[4].guessed() {
		t.Fatalf("the flash-forward, the kitchen: %+v %+v", gs[3], gs[4])
	}
	for _, sec := range sectionsOf(ps, newSourceModel(ps, script, starts).status, gs) {
		if sec.skipped && sec.first > 366 {
			t.Errorf("%s (%d–%d) offered as cut", sec.name, sec.first, sec.last)
		}
	}
}
