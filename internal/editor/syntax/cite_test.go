package syntax

import (
	"reflect"
	"strings"
	"testing"
)

// keysOf is each citation's text and keys ("-" before a suppressed one).
func keysOf(line string) []string {
	r := []rune(line)
	var out []string
	for _, c := range Citations(line) {
		var ks []string
		for _, k := range c.Keys {
			s := k.Key
			if k.Suppress {
				s = "-" + s
			}
			if string(r[k.Start:k.End]) != "@"+k.Key && string(r[k.Start:k.End]) != "@{"+k.Key+"}" {
				s += "!span"
			}
			ks = append(ks, s)
		}
		out = append(out, string(r[c.Start:c.End])+" = "+strings.Join(ks, ","))
	}
	return out
}

func TestCitations(t *testing.T) {
	for _, tc := range []struct {
		line string
		want []string
	}{
		{"As [@doe99] says.", []string{"[@doe99] = doe99"}},
		{"Seen [@doe99, p. 12].", []string{"[@doe99, p. 12] = doe99"}},
		{"[see @a; @b, ch. 2; also @c]", []string{"[see @a; @b, ch. 2; also @c] = a,b,c"}},
		{"Doe says [-@doe99].", []string{"[-@doe99] = -doe99"}},
		{"@doe99 says so.", []string{"@doe99 = doe99"}},
		{"@doe99 [p. 33] says so.", []string{"@doe99 [p. 33] = doe99"}},
		{"As @doe99, and @roe:2001.", []string{"@doe99 = doe99", "@roe:2001 = roe:2001"}}, // the comma and full stop are not the key's
		{"Ask me@example.org.", nil},
		{"Not \\@doe99.", nil},
		{"Braces [@{a key, with comma}]", []string{"[@{a key, with comma}] = a key, with comma"}},
		{"A [@link](https://x.org) stays a link.", nil},
		{"A [ref][@x] stays a reference.", nil},
		{"A [ref] [@x] is one.", []string{"[@x] = x"}},
		{"A note[^1] and [^@x] are not.", nil},
		{"In `code @doe99` none.", nil},
		{"[no citation here]", nil},
		{"[see @a; and no key]", []string{"@a = a"}}, // no bracketed citation: @a is one in the text
		{"Unicode @Müller2020 too.", []string{"@Müller2020 = Müller2020"}},
		{"*Emphasis with [@a] inside*", []string{"[@a] = a"}}, // Pandoc cites there too
		{"**Bold @b** and [see @c](https://x.org/@d) and <https://x.org/@e>", []string{"@b = b"}},
		{"`` code @x `` then @y", []string{"@y = y"}},
		{"Before *it* [@a].", []string{"[@a] = a"}},
	} {
		if got := keysOf(tc.line); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: %q, want %q", tc.line, got, tc.want)
		}
	}
}

func TestCiteTyped(t *testing.T) {
	for _, tc := range []struct {
		line   string
		cursor int
		start  int
		typed  string
		ok     bool
	}{
		{"See @", 5, 4, "", true},
		{"See [@do", 8, 5, "do", true},
		{"See [@doe99, p", 14, 0, "", false},
		{"me@x", 4, 0, "", false},
		{"See @doe99", 7, 4, "do", true}, // in the key: what is before the cursor
		{"`@do`", 4, 0, "", false},
		{"[-@ro", 5, 2, "ro", true},
	} {
		start, typed, ok := CiteTyped(tc.line, tc.cursor)
		if ok != tc.ok || ok && (start != tc.start || typed != tc.typed) {
			t.Errorf("%q at %d: %d %q %v, want %d %q %v", tc.line, tc.cursor, start, typed, ok, tc.start, tc.typed, tc.ok)
		}
	}
	if e := CiteKeyEnd("See @doe99, p", 4); e != 10 {
		t.Errorf("key end %d, want 10", e)
	}
}

// TestCitationsAreSpans: the editor colours a citation as one span, and
// links and emphasis around it stay what they were.
func TestCitationsAreSpans(t *testing.T) {
	got := MarkdownInline("**Bold** [@a] and [a link](x) and @b.")
	var kinds []InlineKind
	for _, s := range got {
		kinds = append(kinds, s.Kind)
	}
	if want := []InlineKind{Strong, Cite, Link, Cite}; !reflect.DeepEqual(kinds, want) {
		t.Errorf("kinds %v, want %v", kinds, want)
	}
}
