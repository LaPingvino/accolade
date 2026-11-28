package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// esperantoFDX is a Final Draft script in Esperanto: a line of action in
// capitals starts with "EK" ("off!"), a scene word in Esperanto (EK.,
// exterior), where in English it is action as it is.
const esperantoFDX = `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
<FinalDraft DocumentType="Script" Template="No" Version="5">
<Content>
<Paragraph Type="Scene Heading"><Text>EN. ĈAMBRO - TAGO</Text></Paragraph>
<Paragraph Type="Action"><Text>EK AL LA PORDO!</Text></Paragraph>
<Paragraph Type="Character"><Text>BOB</Text></Paragraph>
<Paragraph Type="Dialogue"><Text>Saluton.</Text></Paragraph>
</Content>
</FinalDraft>
`

// An Esperanto script written as Fountain through Lexington keeps the
// force marks Esperanto needs: imported from Final Draft and exported as
// Fountain by Lexington, "EK AL LA PORDO!" stays action ("!"), where
// English needs no mark.
func TestFountainWrittenInScriptLanguage(t *testing.T) {
	eo := lexSettings.Language("eo")
	reads := func(text string) []lex.Line {
		var rows []lex.Line
		for _, l := range fountain.ParseLinesIn(eo, strings.NewReader(text)) {
			if l.Type != lex.TypeEmpty {
				rows = append(rows, l)
			}
		}
		return rows
	}
	isAction := func(text string) bool {
		for _, l := range reads(text) {
			if strings.Contains(l.Contents, "EK AL LA PORDO") {
				return l.Type == lex.TypeAction
			}
		}
		return false
	}

	text, err := importFDX(strings.NewReader(esperantoFDX), eo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "!EK AL LA PORDO!") || !isAction(text) {
		t.Errorf("imported in Esperanto, the action should keep its \"!\":\n%s", text)
	}
	if !strings.HasPrefix(text, "EN. ĈAMBRO - TAGO\n") {
		t.Errorf("the Esperanto heading needs no \".\":\n%s", text)
	}
	en, err := importFDX(strings.NewReader(esperantoFDX), langFor(nil))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(en, "!EK AL") {
		t.Errorf("in English the action needs no mark:\n%s", en)
	}

	// exported as Fountain by Lexington, the script's language's marks
	out := filepath.Join(t.TempDir(), "script.fountain")
	if err := exportAs(text, out, "fountain", exportJob{language: "eo", scenes: sceneHeaders("eo")}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !isAction(string(data)) {
		t.Errorf("exported in Esperanto, the action should stay action:\n%s", data)
	}
}
