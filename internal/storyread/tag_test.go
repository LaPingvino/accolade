// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package storyread

import (
	"strings"
	"testing"
)

// The sentence that says who speaks is a piece of its own, a tag, so the
// narration around it can be taken without it.
func TestTags(t *testing.T) {
	r := Read([]Passage{
		{Text: "“Out,” said Tom, shaking his coat. He sat down. “Is there tea?”"},
		{Text: "Tom said: “No.”"},
	}, "en", Hints{})
	var got []string
	for _, p := range r.Passages {
		for _, pc := range p.Pieces {
			kind := "narration"
			switch {
			case pc.Speech:
				kind = "speech"
			case pc.Tag:
				kind = "tag"
			}
			got = append(got, kind+": "+pc.Text)
		}
	}
	want := []string{"speech: Out,", "tag: said Tom, shaking his coat.", "narration: He sat down.",
		"speech: Is there tea?", "tag: Tom said:", "speech: No."}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("pieces:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
