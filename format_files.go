package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"

	"github.com/LaPingvino/lexington/rules"
)

// Editing the format files (format_buttons.go): one's own are in
// userFormatDir; editing Accolade's or Lexington's keeps a changed copy
// there, which removing brings back to the built-in one.

// formatSpec is what a format file holds.
type formatSpec struct {
	Name, Description, Page, Base string
	Elements                      rules.Set
	Accolade                      struct{ Buttons []formatButton }
}

// userFormat reports whether one's own folder has the format.
func userFormat(key string) bool { return fileExists(filepath.Join(userFormatDir(), key+".toml")) }

// builtinFormat reports whether the format comes with Lexington or Accolade.
func builtinFormat(key string) bool {
	_, preset := rules.GetPreset(key)
	return preset || fileExists(filepath.Join(builtinFormatDir(), key+".toml"))
}

// formatText is a format's file to edit: one's own, else Accolade's, else
// one on Lexington's preset; with the elements it starts from, as
// comments, if it changes none.
func formatText(key string) string {
	text := fmt.Sprintf("Base = %q # Lexington's preset, as it is\n", key)
	if p := formatFile(key); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			text = string(b)
		}
	}
	var spec formatSpec
	if _, err := toml.Decode(text, &spec); err != nil || len(spec.Elements) > 0 {
		return text
	}
	base := spec.Base
	if base == "" {
		base = rules.ScreenplayPreset
	}
	src, ok := rules.PresetSource(base)
	if !ok {
		return text
	}
	var b strings.Builder
	b.WriteString(text)
	fmt.Fprintf(&b, "\n# The elements it starts from (%s), to copy here, without the #, and change:\n", base)
	for _, l := range strings.Split(strings.TrimRight(src, "\n"), "\n") {
		if strings.HasPrefix(l, "Name") || strings.HasPrefix(l, "Description") || strings.HasPrefix(l, "Page") {
			continue
		}
		b.WriteString("# " + l + "\n")
	}
	return b.String()
}

// saveFormat checks a format's file and keeps it in one's own folder.
func saveFormat(key, text string) error {
	var spec formatSpec
	if _, err := toml.Decode(text, &spec); err != nil {
		return err
	}
	if spec.Base != "" && !slices.ContainsFunc(scriptFormats(), func(p rules.Preset) bool { return p.Key == spec.Base }) {
		return fmt.Errorf("no format %q to start from (Base)", spec.Base)
	}
	if spec.Base == key && !builtinFormat(key) {
		return errors.New("a format cannot start from itself (Base)")
	}
	if err := os.MkdirAll(userFormatDir(), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(userFormatDir(), key+".toml"), []byte(text))
}

// newFormat makes a format of one's own named name, starting from the
// format from with its buttons; it returns the new format's key.
func newFormat(name, from string) (string, error) {
	key := formatKey(name)
	if key == "" {
		return "", errors.New("give the format a name")
	}
	if slices.ContainsFunc(scriptFormats(), func(p rules.Preset) bool { return p.Key == key || p.Name == name }) {
		return "", fmt.Errorf("there is a format %q already", name)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Name = %q\nBase = %q\n", name, from)
	if buttons := formatButtons(from); len(buttons) > 0 {
		b.WriteString("\n")
		if err := toml.NewEncoder(&b).Encode(struct {
			Accolade struct{ Buttons []formatButton }
		}{struct{ Buttons []formatButton }{buttons}}); err != nil {
			return "", err
		}
	}
	return key, saveFormat(key, b.String())
}

// removeFormat removes one's own format (a changed built-in one is the
// built-in one again).
func removeFormat(key string) error {
	return os.Remove(filepath.Join(userFormatDir(), key+".toml"))
}

// formatKey is a format's file name from its name: lower case, words
// joined with -.
func formatKey(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		default:
			dash = true
		}
	}
	return b.String()
}
