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
// {n} numbers it: one more than the lines up to the cursor that start
// as Insert does (up to {n}), counting again after a line starting with
// Restart. A page break (===) starting Insert is left out for number 1.

type formatButton struct {
	Label   string
	Insert  string
	Restart string
}

var defaultFormatButtons = map[string][]formatButton{
	"default": {
		{Label: "Cold Open", Insert: ">COLD OPEN<"},
		{Label: "Act", Insert: ">ACT {n}<"},
		{Label: "End of Act", Insert: ">END OF ACT {n}<"},
		{Label: "Tag", Insert: ">TAG<"},
		{Label: "Montage", Insert: "BEGIN MONTAGE:\n\n\n\nEND MONTAGE."},
		{Label: "Intercut", Insert: "> INTERCUT WITH:"},
	},
	"comic": {
		{Label: "Page", Insert: "===\n\n.PAGE {n}"},
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
	"stageplay":    stageButtons,
	"stageplay-uk": stageButtons,
	"musical":      append(stageButtons, formatButton{Label: "Song", Insert: ">SONG: <"}),
	"radio": {
		{Label: "SFX", Insert: "SFX: "},
		{Label: "Music", Insert: "MUSIC: "},
		{Label: "Fade", Insert: "FADE OUT."},
	},
}

var stageButtons = []formatButton{
	{Label: "Lights Up", Insert: "LIGHTS UP."},
	{Label: "Blackout", Insert: "BLACKOUT."},
	{Label: "Curtain", Insert: "> CURTAIN."},
}

// formatButtons are the buttons of the format with key: from Lexington's
// settings if they set any, else Accolade's own.
func formatButtons(key string) []formatButton {
	var conf struct {
		Accolade struct{ Buttons map[string][]formatButton }
	}
	if _, err := toml.DecodeFile(lexingtonConfig(), &conf); err == nil {
		if b, ok := conf.Accolade.Buttons[key]; ok {
			return b
		}
	}
	return defaultFormatButtons[key]
}

// insertButton inserts a button's text as a paragraph of its own at the
// cursor's line, numbered, with the cursor after it.
func insertButton(b *buffer.Buffer, fb formatButton) {
	line, _ := b.Position(b.Cursor())
	n := 1
	if i := strings.Index(fb.Insert, "{n}"); i >= 0 {
		prefix := strings.TrimSpace(strings.TrimPrefix(fb.Insert[:i], "===\n\n"))
		for l := 0; l <= line; l++ {
			t := strings.TrimSpace(b.Line(l))
			switch {
			case fb.Restart != "" && strings.HasPrefix(t, fb.Restart):
				n = 1
			case prefix != "" && strings.HasPrefix(t, prefix):
				n++
			}
		}
	}
	text := strings.ReplaceAll(fb.Insert, "{n}", strconv.Itoa(n))
	if n == 1 {
		text = strings.TrimPrefix(text, "===\n\n")
	}
	at := b.LineEnd(line)
	switch {
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
