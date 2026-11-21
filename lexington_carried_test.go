package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/lexington/rules"
)

// A language and a format of one's own, as one machine has them.
const (
	xqScript   = "ENDE. DOMO - TAGO\n\nBram kuiras.\n\nINT. is English for inside.\n"
	xqFountain = ".ENDE. DOMO - TAGO\n\nBram kuiras.\n\n!INT. is English for inside.\n"
	xqSettings = "# my own languages\n[Scenes]\nxq = [\"ENDE\", \"AUSE\"]\n\n[Matchups]\nxq = [\"ENDE.\", \"AUSE.\", \"\", \"\", \"\"]\n"
	myRadio    = "Name = \"My Radio\"\nBase = \"radio\"\n\n[Elements.action]\nStyle = \"b\"\n"
)

// scratchSettings gives the test a settings folder of its own (empty),
// Lexington's settings read from it, and the ones before back after.
func scratchSettings(t *testing.T) string {
	t.Helper()
	t.Cleanup(reloadLexingtonSettings) // after the folder is put back
	// the format chosen with these settings goes with them: a later test's
	// window must not start in a format only this folder had
	prevFormat := GetSettings().GetString("script-format")
	t.Cleanup(func() { GetSettings().SetString("script-format", prevFormat) })
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	reloadLexingtonSettings()
	return filepath.Join(dir, "accolade")
}

// withXQ is a settings folder with the xq language and the format
// my-radio.
func withXQ(t *testing.T) {
	t.Helper()
	conf := scratchSettings(t)
	os.MkdirAll(filepath.Join(conf, "formats"), 0o755)
	os.WriteFile(filepath.Join(conf, "lexington.toml"), []byte(xqSettings), 0o644)
	os.WriteFile(filepath.Join(conf, "formats", "my-radio.toml"), []byte(myRadio), 0o644)
	reloadLexingtonSettings()
}

// xqAccolade is film.accolade written on a machine with xq and my-radio:
// the script in xq, laid out as My Radio.
func xqAccolade(t *testing.T) string {
	t.Helper()
	w := draftsWindow(t)
	withXQ(t)
	dir := t.TempDir()
	plain := filepath.Join(dir, "film.xq.fountain")
	os.WriteFile(plain, []byte(xqScript), 0o644)
	if err := w.LoadFile(plain); err != nil {
		t.Fatal(err)
	}
	if w.language != "xq" {
		t.Fatalf("opened in %q", w.language)
	}
	w.setScriptFormat("my-radio")
	if err := w.renameTo("film.accolade"); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	return filepath.Join(dir, "film.accolade")
}

// An .accolade carries the language and the format it uses; opened where
// neither is, it reads and writes as on the machine it came from, and
// that machine's settings stay as they were: the bundle's are the
// script's only. Saved there, it still carries them.
func TestCarriedConfRoundTrip(t *testing.T) {
	p := xqAccolade(t)
	data, _ := os.ReadFile(p)
	const pre = "film.textbundle/"
	lexText, ok := zipFile(t, data, pre+bundleLexFile)
	if !ok || !strings.Contains(lexText, "ENDE") || strings.Contains(lexText, "\"eo\"") || !strings.Contains(lexText, "Standard") || !strings.Contains(lexText, "INT") {
		t.Fatalf("lexington.toml carried: %v\n%s", ok, lexText)
	}
	if f, ok := zipFile(t, data, pre+bundleFormatsDir+"my-radio.toml"); f != myRadio || !ok {
		t.Fatalf("the format carried: %q", f)
	}
	info, _ := zipFile(t, data, pre+"info.json")
	if a := oursIn(t, info); a.Format != "my-radio" || a.Language != "xq" {
		t.Errorf("info.json: %+v", a)
	}
	if std, _ := zipFile(t, data, pre+"text.fountain"); std != xqFountain {
		t.Errorf("text.fountain:\n%s", std)
	}

	// another machine, with neither
	w := draftsWindow(t)
	conf := scratchSettings(t)
	w.settings.SetString("script-format", rules.ScreenplayPreset) // (the test's settings are one machine's)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.language != "xq" || w.textEditor.Text() != xqScript {
		t.Fatalf("opened in %q:\n%s", w.language, w.textEditor.Text())
	}
	if w.scriptFormatKey() != "my-radio" || w.lex().format("my-radio").Name != "My Radio" || w.currentScriptFormat()["action"].Style != "b" {
		t.Errorf("format %q: %+v", w.scriptFormatKey(), w.lex().format("my-radio"))
	}
	if !isSceneHeadingIn("ENDE. KAMPO", w.sceneStarts()) {
		t.Error("xq's headings are not headings here")
	}
	if s := w.statusText(); !strings.Contains(s, "Add to My Settings") || !strings.Contains(s, "My Radio") {
		t.Errorf("status: %q", s)
	}
	if lexSettings.Scenes["xq"] != nil || machineLex().hasFormat("my-radio") || fileExists(filepath.Join(conf, "lexington.toml")) ||
		fileExists(filepath.Join(conf, "formats", "my-radio.toml")) {
		t.Error("opening changed the machine's settings")
	}
	if w.settings.GetString("script-format") != rules.ScreenplayPreset {
		t.Error("the script's format became the machine's")
	}
	// exports read the script's settings: its format, its language
	o := w.layoutJob().options("pdf", "out.pdf")
	if o.Conf == nil || o.Conf.Elements["my-radio"]["action"].Style != "b" || o.Conf.Scenes["xq"] == nil || o.Elements != "my-radio" {
		t.Errorf("export options: %+v", o)
	}
	if !cmd("file.addSettings").applies(w.cmdEnv()) {
		t.Error("no Add to My Settings")
	}
	// saved here: still carried, the text as before
	w.textEditor.SetText(xqScript + "\nAUSE. KAMPO - NOKTO\n\nVento.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	scratchSettings(t)
	w2 := againWindow(t, w)
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w2.language != "xq" || !strings.HasSuffix(w2.textEditor.Text(), "AUSE. KAMPO - NOKTO\n\nVento.\n") || w2.scriptFormatKey() != "my-radio" {
		t.Errorf("saved where it was not: %q, %q\n%s", w2.language, w2.scriptFormatKey(), w2.textEditor.Text())
	}
	data, _ = os.ReadFile(p)
	if std, _ := zipFile(t, data, pre+"text.fountain"); !strings.Contains(std, "\n.AUSE. KAMPO - NOKTO") {
		t.Errorf("text.fountain:\n%s", std)
	}
	// a window with another script reads the machine's
	if w3 := againWindow(t, w); w3.lex().knows("xq") {
		t.Error("the script's language reached another window")
	}
}

// Only an .accolade carries settings: a .highland from it leaves them
// out; a stock script carries its language in full, which matches the
// machine's, so there is nothing to add.
func TestCarriedConfOnlyAccolade(t *testing.T) {
	p := xqAccolade(t)
	w := draftsWindow(t)
	withXQ(t)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if err := w.renameTo("film.highland"); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	data, _ := os.ReadFile(filepath.Join(filepath.Dir(p), "film.highland"))
	for _, e := range rawEntries(t, data) {
		if strings.Contains(e.name, "assets/lexington") {
			t.Errorf("a .highland got %s", e.name)
		}
	}
	info, _ := zipFile(t, data, "film.textbundle/info.json")
	if a := oursIn(t, info); a.Format != "" || a.Language != "xq" {
		t.Errorf("info.json: %+v", a)
	}

	w2 := draftsWindow(t)
	scratchSettings(t)
	plain := filepath.Join(t.TempDir(), "plain.fountain")
	os.WriteFile(plain, []byte(bundleScript), 0o644)
	if err := w2.LoadFile(plain); err != nil {
		t.Fatal(err)
	}
	if err := w2.renameTo("plain.accolade"); err != nil {
		t.Fatal(err)
	}
	w2.flushBundleWrites()
	data, _ = os.ReadFile(filepath.Join(filepath.Dir(plain), "plain.accolade"))
	// a stock script carries its language in full all the same (Joop: it
	// reads alike where that language's words were changed), and no format
	var carried []string
	for _, e := range rawEntries(t, data) {
		if strings.Contains(e.name, "lexington") {
			carried = append(carried, e.name)
		}
	}
	if len(carried) != 1 || !strings.HasSuffix(carried[0], "assets/lexington/lexington.toml") {
		t.Errorf("a stock script carries %v", carried)
	}
	if conf, _ := zipFile(t, data, carried[0]); !strings.Contains(conf, "INT") {
		t.Errorf("its language not in full:\n%s", conf)
	}
	if cmd("file.addSettings").applies(w2.cmdEnv()) {
		t.Error("Add to My Settings for a script whose settings are the machine's")
	}
}

// What someone else's bundle names cannot reach outside the formats
// folder, and a file that does not read is left out.
func TestReadCarriedNames(t *testing.T) {
	files := map[string]string{
		bundleFormatsDir + "../../evil.toml": "Name = \"x\"",
		bundleFormatsDir + "Bad Name.toml":   "Name = \"x\"",
		bundleFormatsDir + "broken.toml":     "Name = ",
		bundleFormatsDir + "fine.toml":       "Name = \"Fine\"",
		bundleLexFile:                        "[Scenes]\nzz = [\"A\"]\n",
	}
	var names []string
	for n := range files {
		names = append(names, n)
	}
	c := readCarried(names, func(n string) ([]byte, error) { return []byte(files[n]), nil })
	if c == nil || len(c.formats) != 1 || c.formats["fine"] == nil || c.lex.Scenes["zz"] == nil {
		t.Fatalf("read: %+v", c)
	}
}

// Add to My Settings on a machine without them: listed as new, added on
// Add — the language into lexington.toml, the format into one's own
// folder — after which nothing is left to add.
func TestAddToMySettings(t *testing.T) {
	p := xqAccolade(t)
	w := draftsWindow(t)
	conf := scratchSettings(t)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	w.addToMySettings()
	a := w.adoptAsked
	if a == nil || len(a.adds) != 2 || len(a.picks) != 0 {
		t.Fatalf("listed: %+v", a)
	}
	a.apply()
	if lexSettings.Scenes["xq"] == nil || !machineLex().hasFormat("my-radio") {
		t.Fatal("not added")
	}
	if b, _ := os.ReadFile(filepath.Join(conf, "formats", "my-radio.toml")); string(b) != myRadio {
		t.Errorf("format file: %q", b)
	}
	if len(w.carried.toAdd()) != 0 {
		t.Errorf("left to add: %+v", w.carried.toAdd())
	}
	if !strings.Contains(w.statusText(), "Added to your settings") {
		t.Errorf("status: %q", w.statusText())
	}
}

// With a clash: the machine has another xq and another my-radio. Keep
// mine is chosen to start with and changes nothing; keep both adds theirs
// as xq-2, which the script then uses; take theirs replaces the format.
// lexington.toml keeps its comment and the rest.
func TestAddToMySettingsClash(t *testing.T) {
	p := xqAccolade(t)
	w := draftsWindow(t)
	conf := scratchSettings(t)
	mine := "# mine, keep this\nStandard = \"en\"\n\n[Scenes]\nxq = [\"INNE\", \"UTE\"] # my words\nnl = [\"BIN\", \"BUI\"]\n"
	os.MkdirAll(filepath.Join(conf, "formats"), 0o755)
	os.WriteFile(filepath.Join(conf, "lexington.toml"), []byte(mine), 0o644)
	os.WriteFile(filepath.Join(conf, "formats", "my-radio.toml"), []byte("Name = \"Radio, mine\"\nBase = \"radio\"\n"), 0o644)
	reloadLexingtonSettings()
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if !isSceneHeadingIn("ENDE. DOMO", w.sceneStarts()) || isSceneHeadingIn("INNE. DOMO", w.sceneStarts()) {
		t.Error("the script reads the machine's xq, not its own")
	}
	if s := w.statusText(); !strings.Contains(s, "not as yours") {
		t.Errorf("status: %q", s)
	}
	w.addToMySettings()
	a := w.adoptAsked
	lang, format := a.picks["language:xq"], a.picks["format:my-radio"]
	if lang == nil || format == nil || lang.Selected != pickMine || format.Selected != pickMine {
		t.Fatalf("picks: %+v", a.picks)
	}
	a.apply() // all kept as mine
	if b, _ := os.ReadFile(filepath.Join(conf, "lexington.toml")); string(b) != mine {
		t.Errorf("keep mine changed lexington.toml:\n%s", b)
	}
	w.addToMySettings()
	a = w.adoptAsked
	a.picks["language:xq"].SetSelected(pickBoth)
	if a.names["language:xq"].Text != "xq-2" {
		t.Errorf("both name %q", a.names["language:xq"].Text)
	}
	a.picks["format:my-radio"].SetSelected(pickTheirs)
	a.apply()
	b, _ := os.ReadFile(filepath.Join(conf, "lexington.toml"))
	text := string(b)
	if !strings.Contains(text, "# mine, keep this") || !strings.Contains(text, "# my words") || !strings.Contains(text, "nl = ") {
		t.Errorf("lexington.toml lost something:\n%s", text)
	}
	if !slices.Equal(lexSettings.Scenes["xq"], []string{"INNE", "UTE"}) || !slices.Equal(lexSettings.Scenes["xq-2"], []string{"ENDE", "AUSE"}) ||
		lexSettings.Matchups["xq-2"] == nil {
		t.Errorf("languages: xq %v, xq-2 %v\n%s", lexSettings.Scenes["xq"], lexSettings.Scenes["xq-2"], text)
	}
	if f, _ := os.ReadFile(filepath.Join(conf, "formats", "my-radio.toml")); string(f) != myRadio {
		t.Errorf("take theirs: %q", f)
	}
	if w.language != "xq-2" || !w.hasChanges || !isSceneHeadingIn("ENDE. DOMO", w.sceneStarts()) {
		t.Errorf("the script is in %q now", w.language)
	}
}

// lexington.toml edited as text: a key set in its table, a table added,
// a root key set; a file written otherwise (an inline table) is refused,
// nothing changed.
func TestAdoptTOMLText(t *testing.T) {
	scratchSettings(t)
	text := "[Scenes]\nen = [\n  \"INT\",\n  \"EXT\",\n]\n\n[Elements.mine.action]\nStyle = \"b\"\n"
	got := setTableKey(text, "Scenes", "en", `["A"]`)
	if got != "[Scenes]\nen = [\"A\"]\n\n[Elements.mine.action]\nStyle = \"b\"\n" {
		t.Errorf("replaced:\n%s", got)
	}
	got = setTableKey(text, "Matchups", "xx", `["A."]`)
	if !strings.HasSuffix(got, "\n\n[Matchups]\nxx = [\"A.\"]\n") {
		t.Errorf("added:\n%s", got)
	}
	got = setTableKey(got, "Scenes", "xx", `["B"]`)
	if !strings.Contains(got, "]\nxx = [\"B\"]\n\n[Elements") {
		t.Errorf("added to the table:\n%s", got)
	}
	if got := setRootKey(text, "Standard", `"eo"`); !strings.HasPrefix(got, "Standard = \"eo\"\n[Scenes]") {
		t.Errorf("root:\n%s", got)
	}
	if got := dropTables(text, "Elements.mine"); strings.Contains(got, "Style") {
		t.Errorf("dropped:\n%s", got)
	}
	conf := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "accolade")
	os.MkdirAll(conf, 0o755)
	inline := "Scenes = { en = [\"INT\", \"EXT\"] }\n"
	os.WriteFile(filepath.Join(conf, "lexington.toml"), []byte(inline), 0o644)
	theirs := rules.TOMLConf{Scenes: map[string][]string{"zz": {"Z"}}}
	_, err := adopt([]adoptChoice{{item: carriedItem{kind: "language", key: "zz", state: itemNew, theirs: theirs}, how: adoptAdd}})
	if err == nil {
		t.Error("an inline table was added to")
	}
	if b, _ := os.ReadFile(filepath.Join(conf, "lexington.toml")); string(b) != inline {
		t.Errorf("changed:\n%s", b)
	}
}

// The Add to My Settings dialog with a clash, on a desktop and a phone
// (SHOT_DIR=… go test -run TestAdoptScreenshots): bundleconf-*.png.
func TestAdoptScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	p := xqAccolade(t)
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 760)}, {"phone", fyne.NewSize(380, 800)}} {
		w := draftsWindow(t)
		conf := scratchSettings(t)
		os.MkdirAll(conf, 0o755)
		os.WriteFile(filepath.Join(conf, "lexington.toml"), []byte("[Scenes]\nxq = [\"INNE\", \"UTE\"]\n"), 0o644)
		reloadLexingtonSettings()
		w.fyneWindow.Resize(size.size)
		if err := w.LoadFile(p); err != nil {
			t.Fatal(err)
		}
		capture(t, w.fyneWindow.Canvas(), "bundleconf-status-"+size.name+".png")
		w.addToMySettings()
		capture(t, w.fyneWindow.Canvas(), "bundleconf-adopt-"+size.name+".png")
		w.adoptAsked.picks["language:xq"].SetSelected(pickBoth)
		capture(t, w.fyneWindow.Canvas(), "bundleconf-adopt-both-"+size.name+".png")
	}
}
