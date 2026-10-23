package main

import (
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// A script format's own buttons in the toolbar insert what that kind of
// script uses often: pages and panels in a comic, acts in a series,
// blackouts in a play. Lexington's settings can set them per format, in a
// table Lexington itself skips:
//
//	[[Accolade.Buttons.comic]]
//	Label = "Panel"
//	Insert = "Panel {n}. "
//	Restart = ".PAGE"
//
// A format can also be an alias: Lexington's element set under a name
// of its own, with buttons of its own (a TV series is a screenplay with
// acts; French comics with Dutch labels):
//
//	[Accolade.Formats.strip]
//	Name = "Strip (Franse stijl)"
//	Elements = "bd"
//
// {n} numbers it ({N} in words: ONE, TWO): one more than the lines up to
// the cursor that start as Insert does (up to the number) with a number,
// counting again after a line starting with Restart. A page break (===)
// starting Insert is left out for number 1.

type formatAlias struct {
	Name     string
	Elements string // the element set's key: a preset or one of the settings
}

var defaultAliases = map[string]formatAlias{
	"series": {Name: "TV series", Elements: "default"},
}

// accoladeSettings are Accolade's part of Lexington's settings.
type accoladeSettings struct {
	Buttons map[string][]formatButton
	Formats map[string]formatAlias
}

var accoladeConf = readAccoladeSettings()

func readAccoladeSettings() accoladeSettings {
	var conf struct{ Accolade accoladeSettings }
	_, _ = toml.DecodeFile(lexingtonConfig(), &conf)
	return conf.Accolade
}

// aliases are the formats that are aliases, by key: Accolade's and the
// settings'.
func aliases() map[string]formatAlias {
	out := map[string]formatAlias{}
	for k, a := range defaultAliases {
		out[k] = a
	}
	for k, a := range accoladeConf.Formats {
		out[k] = a
	}
	return out
}

// elementsKey is the element set a format lays out with, for Lexington.
func elementsKey(key string) string {
	if a, ok := aliases()[key]; ok {
		return a.Elements
	}
	return key
}

type formatButton struct {
	Label   string
	Insert  string
	Restart string `toml:",omitempty"`
	// Line: on the line after the cursor's, not as a paragraph of its own
	// (lyrics go under the singer's name, line after line)
	Line bool `toml:",omitempty"`
}

var defaultFormatButtons = map[string][]formatButton{
	"default": {
		{Label: "Montage", Insert: "BEGIN MONTAGE:\n\n\n\nEND MONTAGE."},
		{Label: "Intercut", Insert: "> INTERCUT WITH:"},
		{Label: "Flashback", Insert: "FLASHBACK:"},
		{Label: "Super", Insert: "SUPER: "},
	},
	// as in Lexington's examples/tv-episode.fountain: a section for the
	// outline, a centred heading for the page
	"series": {
		{Label: "Cold Open", Insert: "# COLD OPEN\n\n>COLD OPEN<"},
		{Label: "Main Titles", Insert: ">MAIN TITLES<"},
		{Label: "Act", Insert: "# ACT {N}\n\n>ACT {N}<"},
		{Label: "End of Act", Insert: ">END OF ACT {N}<\n\n==="},
		{Label: "Tag", Insert: "# TAG\n\n>TAG<"},
		{Label: "Intercut", Insert: "> INTERCUT WITH:"},
	},
	// examples/comic.fountain and bd.fountain
	"comic": {
		{Label: "Page", Insert: "===\n\n.PAGE {N}"},
		{Label: "Panel", Insert: "Panel {n}. ", Restart: ".PAGE"},
		{Label: "Caption", Insert: "CAPTION\n"},
		{Label: "SFX", Insert: "SFX\n"},
	},
	"bd": {
		{Label: "Planche", Insert: "===\n\n.PLANCHE {n}"},
		{Label: "Case", Insert: "Case {n}. ", Restart: ".PLANCHE"},
		{Label: "Récitatif", Insert: "RÉCITATIF\n"},
		{Label: "Onomatopée", Insert: "ONOMATOPÉE\n"},
	},
	// stageplay.fountain, musical.fountain and radio.fountain
	"stageplay":    stageButtons,
	"stageplay-uk": stageButtons,
	"musical": {
		{Label: "Scene", Insert: ".SCENE {n} - "},
		{Label: "Song", Insert: "> {n}.  <"},
		{Label: "Lyrics", Insert: "~", Line: true},
		{Label: "Blackout", Insert: "> BLACKOUT."},
	},
	"radio": {
		{Label: "Sound", Insert: "!"},
		{Label: "Music", Insert: "!MUSIC: "},
		{Label: "Lyrics", Insert: "~", Line: true},
		{Label: "Crossfade", Insert: "> CROSSFADE TO:"},
		{Label: "Fade Out", Insert: "> FADE OUT."},
	},
}

var stageButtons = []formatButton{
	{Label: "Scene", Insert: ".SCENE {n}"},
	{Label: "Lights Up", Insert: "LIGHTS UP."},
	{Label: "Blackout", Insert: "> BLACKOUT."},
	{Label: "Curtain", Insert: "> CURTAIN."},
}

// formatButtons are the buttons of the format with key: from Lexington's
// settings if they set any, else Accolade's own.
func formatButtons(key string) []formatButton {
	if b, ok := accoladeConf.Buttons[key]; ok {
		return b
	}
	if b, ok := defaultFormatButtons[key]; ok {
		return b
	}
	return defaultFormatButtons[elementsKey(key)] // an alias without its own: its set's
}

var numberWords = strings.Fields("ONE TWO THREE FOUR FIVE SIX SEVEN EIGHT NINE TEN ELEVEN TWELVE THIRTEEN FOURTEEN FIFTEEN SIXTEEN SEVENTEEN EIGHTEEN NINETEEN TWENTY")

// numberWord is n in words, as acts and pages are numbered (ONE, TWO),
// in figures past twenty.
func numberWord(n int) string {
	if n >= 1 && n <= len(numberWords) {
		return numberWords[n-1]
	}
	return strconv.Itoa(n)
}

// numbered reports whether rest (a line after a button's prefix) starts
// with a number, in figures or words: "> 3. WHO" is a song, "> BLACKOUT."
// is not.
func numbered(rest string) bool {
	word, _, _ := strings.Cut(strings.TrimSpace(rest), " ")
	word = strings.TrimRight(word, ".:<")
	if word == "" {
		return false
	}
	if word[0] >= '0' && word[0] <= '9' {
		return true
	}
	for _, w := range numberWords {
		if strings.EqualFold(word, w) {
			return true
		}
	}
	return false
}

// insertButton inserts a button's text as a paragraph of its own at the
// cursor's line, numbered, with the cursor after it.
func insertButton(b *buffer.Buffer, fb formatButton) {
	line, _ := b.Position(b.Cursor())
	n := 1
	if i := strings.IndexAny(fb.Insert, "{"); i >= 0 && (strings.HasPrefix(fb.Insert[i:], "{n}") || strings.HasPrefix(fb.Insert[i:], "{N}")) {
		prefix := strings.TrimSpace(strings.TrimPrefix(fb.Insert[:i], "===\n\n"))
		for l := 0; l <= line; l++ {
			t := strings.TrimSpace(b.Line(l))
			switch {
			case fb.Restart != "" && strings.HasPrefix(t, fb.Restart):
				n = 1
			case prefix != "" && strings.HasPrefix(t, prefix) && numbered(t[len(prefix):]):
				n++
			}
		}
	}
	text := strings.ReplaceAll(fb.Insert, "{n}", strconv.Itoa(n))
	text = strings.ReplaceAll(text, "{N}", numberWord(n))
	if n == 1 {
		text = strings.TrimPrefix(text, "===\n\n")
	}
	at := b.LineEnd(line)
	switch {
	case fb.Line && strings.TrimSpace(b.Line(line)) != "":
		text = "\n" + text
	case strings.TrimSpace(b.Line(line)) == "":
		at = b.LineStart(line)
		if line > 0 && strings.TrimSpace(b.Line(line-1)) != "" {
			text = "\n" + text
		}
	default:
		text = "\n\n" + text
	}
	b.Replace(at, at, text)
	b.SetCursor(at+len([]rune(text)), false)
}
