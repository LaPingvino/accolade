package help

import (
	"fmt"
	"strings"

	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/report"
)

// The cheat sheets made from Lexington's own tables, so they say what
// Accolade does.

// revisionsSheet: the revision colours in their order (lex.RevisionName).
func revisionsSheet() string {
	var b strings.Builder
	b.WriteString("# Revision Colours\n\n")
	b.WriteString("Each draft issued after the lock (Production > Issue Draft…) takes the next colour. " +
		"Changed lines show a bar in the editor's margin and an asterisk at the right margin of the PDF; by custom the pages go out on paper of that colour.\n\n")
	b.WriteString("## The order\n\n")
	for n := 0; n < 17; n++ {
		name := lex.RevisionName(n)
		switch n {
		case 0:
			fmt.Fprintf(&b, "- **%s** — the lock: the first issue, scene numbers locked\n", name)
		case 9:
			fmt.Fprintf(&b, "- **%s** — after %s the round starts again (no second White)\n", name, lex.RevisionName(8))
		default:
			fmt.Fprintf(&b, "- **%s**\n", name)
		}
	}
	fmt.Fprintf(&b, "- … then %s, %s and on\n\n", lex.RevisionName(17), lex.RevisionName(18))
	b.WriteString("## Good to know\n\n")
	b.WriteString("- A script that comes in with names of its own (an FDX's) keeps them.\n")
	b.WriteString("- Lock Pages keeps page breaks: text that grows goes onto A-pages (12A), never renumbering the rest.\n")
	b.WriteString("- See [Production](production.md) for locking, issuing and the drafts.\n")
	return b.String()
}

// breakdownSheet: the breakdown categories and their colours
// (lex.TagCategories, in a breakdown sheet's order).
func breakdownSheet() string {
	var b strings.Builder
	b.WriteString("# Breakdown Categories\n\n")
	b.WriteString("Select words in the script and Production > Tag Selection… (Ctrl+Shift+B) to tag them. " +
		"The colours are Final Draft's, so an FDX keeps them.\n\n")
	b.WriteString("## The categories\n\n")
	for _, c := range lex.TagCategories {
		fmt.Fprintf(&b, "- **%s** `%s`", c.Name, c.Color)
		if len(c.Aliases) > 0 {
			var also []string
			for _, a := range c.Aliases {
				if strings.Count(a, "-") == 4 || strings.ToLower(a) == a && strings.EqualFold(a, c.Name) {
					continue // an Id, or the name in small letters
				}
				if !containsFold(also, a) {
					also = append(also, a)
				}
			}
			if len(also) > 0 {
				fmt.Fprintf(&b, " — also read: %s", strings.Join(also, ", "))
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("\nA category of your own gets a colour of its own; it is kept with the script's breakdown.\n")
	return b.String()
}

func containsFold(list []string, s string) bool {
	for _, l := range list {
		if strings.EqualFold(l, s) {
			return true
		}
	}
	return false
}

// doodSheet: the Day Out of Days codes (report's).
func doodSheet() string {
	var b strings.Builder
	b.WriteString("# Day Out of Days Codes\n\n")
	b.WriteString("The Day Out of Days (Production > Reports…, or the schedule's Day Out of Days…) shows for each cast member " +
		"which shoot days they work. The codes are Movie Magic Scheduling's.\n\n")
	b.WriteString("## The codes\n\n")
	for _, c := range []struct{ code, what string }{
		{report.CodeStartWork, "start work: the first day they work"},
		{report.CodeWork, "work: a day in between that they work"},
		{report.CodeWorkFinish, "work finish: the last day they work"},
		{report.CodeStartWorkFinish, "start, work and finish: their only day"},
		{report.CodeHold, "hold: a day between their first and last that they do not work (paid to stay)"},
	} {
		fmt.Fprintf(&b, "- **%s** — %s\n", c.code, c.what)
	}
	b.WriteString("\n## Reading it\n\n")
	b.WriteString("- One row per cast member, one column per shoot day, in the schedule's order.\n")
	b.WriteString("- The totals count work and hold days; drops, travel, fittings and rehearsals are not made.\n")
	b.WriteString("- It follows the schedule: move a strip on the stripboard and the codes follow.\n")
	return b.String()
}
