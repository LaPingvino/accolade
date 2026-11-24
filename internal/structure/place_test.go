package structure

import (
	"strings"
	"testing"
)

// A synopsis or note paragraph put by a unit, as a synopsis card is
// dragged on the cards: before or after a beat, before a scene (the end
// of what comes before it), into a section's start (after its own
// synopsis, which is its heading's) or a scene's end, to the body's end.
func TestPlace(t *testing.T) {
	const s = "Title: T\n\n# One\n= own one\n\n= a\n\n= b\n\nINT. X - DAY\n= own x\n\nAction.\n\n# Two\n= own two\n\n= c\n\nINT. Y - DAY\n\nMore.\n"
	place := func(from string, to string, where Where) (string, error) {
		tr := Parse(Plain(Lines(s)))
		toLine := -1
		if to != "" {
			toLine = lineOf(t, s, to)
		}
		res, err := tr.Place(lineOf(t, s, from), toLine, where, Plain)
		if err != nil {
			return s, err
		}
		return Text(res.Lines), nil
	}
	for _, c := range []struct {
		name, from, to string
		where          Where
		want, refuse   string
	}{
		{name: "after the next beat", from: "= a", to: "= b", where: After,
			want: "Title: T\n\n# One\n= own one\n\n= b\n\n= a\n\nINT. X - DAY\n= own x\n\nAction.\n\n# Two\n= own two\n\n= c\n\nINT. Y - DAY\n\nMore.\n"},
		{name: "before a beat in another section", from: "= a", to: "= c", where: Before,
			want: "Title: T\n\n# One\n= own one\n\n= b\n\nINT. X - DAY\n= own x\n\nAction.\n\n# Two\n= own two\n\n= a\n\n= c\n\nINT. Y - DAY\n\nMore.\n"},
		{name: "into a section's start, after its own", from: "= b", to: "# Two", where: Start,
			want: "Title: T\n\n# One\n= own one\n\n= a\n\nINT. X - DAY\n= own x\n\nAction.\n\n# Two\n= own two\n\n= b\n\n= c\n\nINT. Y - DAY\n\nMore.\n"},
		{name: "into a scene's end", from: "= c", to: "INT. X - DAY", where: End,
			want: "Title: T\n\n# One\n= own one\n\n= a\n\n= b\n\nINT. X - DAY\n= own x\n\nAction.\n\n= c\n\n# Two\n= own two\n\nINT. Y - DAY\n\nMore.\n"},
		{name: "before a scene: the end of what comes before", from: "= c", to: "INT. X - DAY", where: Before,
			want: "Title: T\n\n# One\n= own one\n\n= a\n\n= b\n\n= c\n\nINT. X - DAY\n= own x\n\nAction.\n\n# Two\n= own two\n\nINT. Y - DAY\n\nMore.\n"},
		{name: "to the body's end", from: "= a", to: "", where: End,
			want: "Title: T\n\n# One\n= own one\n\n= b\n\nINT. X - DAY\n= own x\n\nAction.\n\n# Two\n= own two\n\n= c\n\nINT. Y - DAY\n\nMore.\n\n= a\n"},
		{name: "where it is", from: "= a", to: "= a", where: Before, refuse: "there already"},
		{name: "a heading's own synopsis is its heading's", from: "= own one", to: "= c", where: Before, refuse: "Only a synopsis or a note"},
		{name: "action is not placed so", from: "Action.", to: "= c", where: Before, refuse: "Only a synopsis or a note"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := place(c.from, c.to, c.where)
			if c.refuse != "" {
				if err == nil || !strings.Contains(err.Error(), c.refuse) {
					t.Fatalf("want a refusal %q, got %v:\n%s", c.refuse, err, got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
}
