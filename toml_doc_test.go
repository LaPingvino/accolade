package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/rules"
)

// tomlFixtures are real files of each kind, read from where they are:
// Accolade's own lexington.toml and FyneApp.toml, its format files and
// Lexington's presets.
func tomlFixtures(t *testing.T) map[string]struct {
	text string
	kind tomlKind
} {
	t.Helper()
	out := map[string]struct {
		text string
		kind tomlKind
	}{}
	read := func(name, path string, kind tomlKind) {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = struct {
			text string
			kind tomlKind
		}{string(b), kind}
	}
	read("lexington.toml", "lexington.toml", tomlSettings)
	read("FyneApp.toml", "FyneApp.toml", tomlPlain)
	files, _ := filepath.Glob("formats/*.toml")
	for _, f := range files {
		read("accolade-"+filepath.Base(f), f, tomlFormat)
	}
	for _, p := range rules.Presets()[1:] {
		src, _ := rules.PresetSource(p.Key)
		out["lexington-"+p.Key+".toml"] = struct {
			text string
			kind tomlKind
		}{src, tomlFormat}
	}
	return out
}

// A .toml is what its name, its folder or what it holds says: Accolade's
// FyneApp.toml is no Lexington file; an empty file, a Base alone, a
// configuration with only an element set, a format with only an element.
func TestClassifyTOML(t *testing.T) {
	scratchSettings(t)
	for name, f := range tomlFixtures(t) {
		// by content: named otherwise, outside a formats folder
		if got := classifyTOML(filepath.Join(t.TempDir(), "x.toml"), f.text); got != f.kind && name != "lexington.toml" {
			t.Errorf("%s: %v, want %v", name, got, f.kind)
		}
	}
	for _, c := range []struct {
		path, text string
		want       tomlKind
	}{
		{"/x/lexington.toml", "", tomlSettings},
		{"/x/lexington.toml", "[broken", tomlSettings},
		{"/x/a.toml", "", tomlPlain},
		{"/x/a.toml", "[broken", tomlPlain},
		{"/x/a.toml", "Base = \"bd\"\n", tomlFormat},
		{"/x/a.toml", "[Elements.mine.action]\nLeft = 1.5\n", tomlSettings},
		{"/x/a.toml", "[Elements.scene]\nLeft = 2.0\n", tomlFormat},
		{"/x/a.toml", "Standard = \"eo\"\n", tomlSettings},
		{"/x/a.toml", "[package]\nname = \"x\"\n", tomlPlain},
		{"/x/a.toml", "[Elements]\nfoo = 1\n", tomlPlain},
		{"/x/a.toml", "Name = \"my app\"\nversion = 2\n", tomlPlain},
		{"/x/a.toml", "Extensions = [\"md\", \"txt\"]\n", tomlPlain},
		{"/x/a.toml", "Standard = 3\n", tomlPlain},
		{"/x/a.toml", "Base = \"no-such-format\"\n", tomlFormat},
		{"/x/a.toml", "Base = 3\n", tomlPlain},
		{filepath.Join(userFormatDir(), "mine.toml"), "", tomlFormat},
	} {
		if got := classifyTOML(c.path, c.text); got != c.want {
			t.Errorf("%s %q: %v, want %v", c.path, c.text, got, c.want)
		}
	}
}

// comments are a text's comment lines.
func comments(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			out = append(out, strings.TrimSpace(l))
		}
	}
	return out
}

// Edits of every fixture keep every comment and change nothing but their
// path (edit checks that), and read back as meant: an element changed
// whole in a format, a root key, a set's element, a language's field.
func TestTOMLEditsKeepTheRest(t *testing.T) {
	scratchSettings(t)
	for name, f := range tomlFixtures(t) {
		if f.kind == tomlPlain {
			continue
		}
		d := newTOMLDoc(filepath.Join(t.TempDir(), "x.toml"), f.text)
		d.kind = f.kind
		var ops [][]tomlOp
		if f.kind == tomlFormat {
			ops = [][]tomlOp{
				{set(2.25, "Elements", "scene", "Left"), set("b", "Elements", "scene", "Style")},
				{set("Mine", "Name")},
				{drop("Elements", "action")},
			}
		} else {
			ops = [][]tomlOp{
				{set(2.25, "Elements", "default", "scene", "Left")},
				{set([]string{"EN", "EKS"}, "Languages", "xx", "Scenes")},
				{set("eo", "Standard")},
				{set("eo", "Extensions", "esp")},
			}
		}
		for _, o := range ops {
			before := comments(d.text)
			if err := d.edit(o...); err != nil {
				t.Errorf("%s: %v: %v", name, o, err)
				continue
			}
			after := strings.Join(comments(d.text), "\n")
			for _, c := range before {
				if !strings.Contains(after, c) {
					t.Errorf("%s: %v lost the comment %q", name, o, c)
				}
			}
		}
		switch f.kind {
		case tomlFormat:
			list, err := rules.WithPresetFiles(rules.Presets(), map[string][]byte{"x": []byte(d.text)})
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			for _, p := range list {
				if p.Key != "x" {
					continue
				}
				if p.Elements["scene"].Left != 2.25 || p.Elements["scene"].Style != "b" || p.Name != "Mine" {
					t.Errorf("%s: read back %+v, %q", name, p.Elements["scene"], p.Name)
				}
			}
		case tomlSettings:
			c := rules.DefaultConf().With(d.conf())
			if c.Elements["default"]["scene"].Left != 2.25 || c.Standard != "eo" || c.Scenes("xx")[1] != "EKS" || c.FileExtensions()["esp"] != "eo" {
				t.Errorf("%s: read back %+v %q %v", name, c.Elements["default"]["scene"], c.Standard, c.Scenes("xx"))
			}
		}
	}
}

// Files written by hand: [[blocks]] with no [Accolade] above them, an
// inline table, a description over several lines with a [ in it, keys in
// quotes; and a dotted key, which is refused, nothing changed.
func TestTOMLEditsHandWritten(t *testing.T) {
	scratchSettings(t)
	text := "# my comic\nName = \"Mine\"\nDescription = \"\"\"\nFor pages.\n[not a table]\n\"\"\"\n\n" +
		"# the panel button\n[[Accolade.Buttons]]\nLabel = \"Panel\"\nInsert = \"Panel {n}. \"\n\n" +
		"# the page button\n[[Accolade.Buttons]]\nLabel = \"Page\"\nInsert = \"===\\n\\n.PAGE {n}\"\n"
	d := newTOMLDoc("/x/mine.toml", text)
	if d.kind != tomlFormat {
		t.Fatal(d.kind)
	}
	if err := d.edit(set("comic", "Base"), set(2.0, "Elements", "scene", "Left")); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d.text, "# my comic\nName = \"Mine\"\nDescription = \"\"\"\nFor pages.\n[not a table]\n\"\"\"\nBase = \"comic\"\n") {
		t.Errorf("root key not after the root's keys:\n%s", d.text)
	}
	if !strings.HasSuffix(d.text, "[Elements.scene]\nLeft = 2.0\n") {
		t.Errorf("table not at the end:\n%s", d.text)
	}
	// the buttons the other way round, their comments with them
	err := d.editText(func(s string) (string, error) { return moveBlock(s, []string{"Accolade", "Buttons"}, 1, 0) },
		func(m map[string]any) map[string]any {
			b := m["Accolade"].(map[string]any)["Buttons"].([]map[string]any)
			return withAt(m, []string{"Accolade", "Buttons"}, []map[string]any{b[1], b[0]})
		})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d.text, "# the page button\n[[Accolade.Buttons]]\nLabel = \"Page\"") ||
		!strings.Contains(d.text, "# the panel button\n[[Accolade.Buttons]]\nLabel = \"Panel\"") ||
		strings.Index(d.text, "Page\"") > strings.Index(d.text, "Panel\"") {
		t.Errorf("moved:\n%s", d.text)
	}
	if s := d.spec(); len(s.Accolade.Buttons) != 2 || s.Accolade.Buttons[0].Label != "Page" || s.Elements["scene"].Left != 2 {
		t.Errorf("read back %+v", s)
	}
	// a key in a block, one added, one block out
	if err := d.editText(func(s string) (string, error) {
		return setInBlock(s, []string{"Accolade", "Buttons"}, 1, "Restart", tomlValue(".PAGE"))
	}, func(m map[string]any) map[string]any {
		b := m["Accolade"].(map[string]any)["Buttons"].([]map[string]any)
		b[1]["Restart"] = ".PAGE"
		return m
	}); err != nil {
		t.Fatal(err)
	}
	if s := d.spec(); s.Accolade.Buttons[1].Restart != ".PAGE" {
		t.Errorf("restart: %+v\n%s", s.Accolade.Buttons, d.text)
	}

	// a comment at the end of a line stays with its key
	d = newTOMLDoc("/x/mine.toml", "Base = \"comic\" # Lexington's preset, as it is\nName = \"A # not a comment\" # mine\n")
	if err := d.edit(set("bd", "Base"), set("B", "Name")); err != nil {
		t.Fatal(err)
	}
	if d.text != "Base = \"bd\" # Lexington's preset, as it is\nName = \"B\" # mine\n" {
		t.Errorf("trailing comments:\n%s", d.text)
	}

	// an inline table: written again whole
	d = newTOMLDoc("/x/lexington.toml", "[Languages.zh]\n# the shape\nShape = {Words = 0, Chars = 6, Ends = \"。\"}\nRepeat = 2\n")
	if err := d.edit(set(int64(4), "Languages", "zh", "Shape", "Chars")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d.text, "# the shape\nShape = {Chars = 4, Ends = \"。\", Words = 0}\nRepeat = 2") {
		t.Errorf("inline:\n%s", d.text)
	}
	// quoted keys
	if err := d.edit(set("Title", "Languages", "zh", "TitleKeys", "标题")); err != nil {
		t.Fatal(err)
	}
	if d.conf().Languages["zh"].TitleKeys["标题"] != "Title" {
		t.Errorf("quoted key:\n%s", d.text)
	}
	// dotted keys: refused
	d = newTOMLDoc("/x/lexington.toml", "[Languages]\nzz.Scenes = [\"A\"]\n")
	before := d.text
	if err := d.edit(set([]string{"B"}, "Languages", "zz", "Scenes")); err == nil || d.text != before {
		t.Errorf("dotted: %v\n%s", err, d.text)
	}
	// a wrong value: refused with why
	d = newTOMLDoc("/x/lexington.toml", "")
	if err := d.edit(set("diagonal", "Languages", "xx", "Direction")); err == nil || !strings.Contains(err.Error(), "ltr or rtl") {
		t.Errorf("direction: %v", err)
	}
	if d.text != "" {
		t.Errorf("changed: %q", d.text)
	}
}
