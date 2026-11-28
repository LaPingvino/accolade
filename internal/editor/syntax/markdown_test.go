// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package syntax

import (
	"reflect"
	"strings"
	"testing"
)

func TestClassifyMarkdown(t *testing.T) {
	doc := []struct {
		line string
		kind Kind
	}{
		{"---", MdFrontMatter},
		{"title: The Barn", MdFrontMatter},
		{"---", MdFrontMatter},
		{"# Chapter One", MdHeading},
		{"", Empty},
		{"Rain hammered the roof.", Action},
		{"", Empty},
		{"> It's coming, she said.", MdQuote},
		{"", Empty},
		{"- the dog", MdListItem},
		{"    still barking", Action}, // the item's, not code
		{"2. the cat", MdListItem},
		{"", Empty},
		{"```go", MdCodeFence},
		{"# not a heading", MdCode},
		{"```", MdCodeFence},
		{"", Empty},
		{"The Return", MdHeading},
		{"----------", MdHeading},
		{"", Empty},
		{"* * *", MdRule},
		{"", Empty},
		{"    indented code", MdCode},
		{"", Empty},
		{"<!-- Notes", MdComment},
		{"call Bram", MdComment},
		{"-->", MdComment},
		{"#hashtag is no heading", Action},
		{"INT. BARN - DAY", Action}, // no Fountain in Markdown
	}
	var lines []string
	for _, l := range doc {
		lines = append(lines, l.line)
	}
	got := ClassifyMarkdown(lines)
	for i, l := range doc {
		if got[i] != l.kind {
			t.Errorf("line %d %q: kind %d, want %d", i, l.line, got[i], l.kind)
		}
	}
}

func TestMarkdownHeadings(t *testing.T) {
	text := "# One #\n\nText\n\n```\n## fenced\n```\n\nTwo\n===\n\n### Three\nSub\n---"
	got := MarkdownHeadings(strings.Split(text, "\n"))
	want := []Heading{{0, 1, "One"}, {8, 1, "Two"}, {11, 3, "Three"}, {12, 2, "Sub"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("headings %v, want %v", got, want)
	}
}

func TestMarkdownInline(t *testing.T) {
	for _, c := range []struct {
		line string
		want []Span
	}{
		{"a **bold** b", []Span{{2, 10, Strong}}},
		{"an *italic* and _this_", []Span{{3, 11, Emphasis}, {16, 22, Emphasis}}},
		{"`a * b` [link](http://x) <https://y>", []Span{{0, 7, CodeSpan}, {8, 24, Link}, {25, 36, Link}}},
		{"snake_case_name and 2 * 3 * 4", nil},
		{`\*not\*`, nil},
	} {
		if got := MarkdownInline(c.line); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: %v, want %v", c.line, got, c.want)
		}
	}
}
