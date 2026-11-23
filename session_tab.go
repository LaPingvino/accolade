package main

import (
	"fmt"
	"hash/fnv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The Session tab (LOOP.md, CONTEXTUAL RIBBON TABS (4)): the ribbon's
// contextual tab while the script is shared (collab_window.go), beside
// the others, not brought forward (the Collaboration tab stays): who is
// here (each in a colour of their own; tapped, followed), Follow, Copy
// the Link, Leave.
//
// Who is here is the session's presence (internal/collab presence.go):
// each window says it is here, and the line its cursor is on, as it comes,
// as the line changes, and every collab.HereEvery. Following someone, the
// cursor goes to their line each time theirs moves, until you type, they
// leave, or Stop Following.

// The rules of the Session tab's commands.
func sharedNow(e cmdEnv) bool    { return e.shared }
func othersHere(e cmdEnv) bool   { return e.others }
func followingNow(e cmdEnv) bool { return e.following }

func (w *MainWindow) sessionEnv(e *cmdEnv) {
	s := w.shared
	e.shared = s != nil
	if s != nil {
		e.others = len(s.session.Here()) > 0
		e.following = s.follow != ""
	}
}

// peerColors are the colours the people here are told apart by.
var peerColors = []string{"#e8590c", "#1c7ed6", "#2f9e44", "#ae3ec9", "#f08c00", "#0c8599", "#e03131", "#5f3dc4"}

// peerColor is replica's colour (the same on every side).
func peerColor(replica string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(replica))
	return peerColors[h.Sum32()%uint32(len(peerColors))]
}

// peerDot is a dot in replica's colour, as an icon.
func peerDot(replica string) fyne.Resource {
	c := peerColor(replica)
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><circle cx="12" cy="12" r="8" fill="` + c + `"/></svg>`
	return fyne.NewStaticResource("peer-"+strings.TrimPrefix(c, "#")+".svg", []byte(svg))
}

// hereNow is who else is here ("": not shared).
func (w *MainWindow) hereNow() []collab.Presence {
	if w.shared == nil {
		return nil
	}
	return w.shared.session.Here()
}

// hereKey is who is here and who is followed, for the ribbon's groups.
func (w *MainWindow) hereKey() string {
	if w.shared == nil {
		return ""
	}
	var b strings.Builder
	for _, p := range w.hereNow() {
		b.WriteString(p.Replica + "=" + p.Name + ";")
	}
	return b.String() + "→" + w.shared.follow
}

// scriptCursorLine is the script's cursor's line.
func (w *MainWindow) scriptCursorLine() int {
	if w.textEditor == nil {
		return 0
	}
	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	return line
}

// sayHere tells the others this side is here, and its place (its
// cursor's line, its selection): when it changed, or always (again).
func (w *MainWindow) sayHere(always bool) { w.sayPlace(always) }

// hereBeat says this side is still here every collab.HereEvery, and
// lets the ribbon see who went quiet.
func (w *MainWindow) hereBeat(s *shared) {
	s.beat = w.after(collab.HereEvery, func() {
		if w.shared != s {
			return
		}
		w.sayHere(true)
		w.hereChanged()
		w.hereBeat(s)
	})
}

// hereChanged follows who is here: the one followed, the ribbon.
func (w *MainWindow) hereChanged() {
	s := w.shared
	if s == nil {
		return
	}
	if s.follow != "" {
		var them *collab.Presence
		for _, p := range s.session.Here() {
			if p.Replica == s.follow {
				them = &p
			}
		}
		switch {
		case them == nil:
			w.setStatus(s.followName + " left: not following any more")
			s.follow = ""
		case them.Line != s.followLine:
			s.followLine = them.Line
			w.followTo(them.Line)
		}
	}
	w.tableReadFollowHere() // the Table Read following someone (tableread_follow.go)
	w.showPeers()           // their places in the editor, the outline and the cards (collab_cursors.go)
	w.updateStatusBar()     // "Shared · 1 other"
	w.ribbonFollow()
}

// followTo puts the script's cursor on line, as followed (the script's
// tab in front; the focus where it is).
func (w *MainWindow) followTo(line int) {
	e := w.textEditor
	if e == nil {
		return
	}
	if !w.scriptInMain() {
		w.showMiddle("")
	}
	e.Navigate(func(b *buffer.Buffer) {
		n, _ := b.Position(b.Len())
		b.SetCursor(b.LineStart(max(0, min(line, n))), false)
	})
}

// follow follows replica (again: no longer).
func (w *MainWindow) follow(replica string) {
	s := w.shared
	if s == nil {
		return
	}
	if s.follow == replica {
		w.stopFollowing()
		return
	}
	for _, p := range s.session.Here() {
		if p.Replica == replica {
			s.follow, s.followName, s.followLine = replica, p.Name, p.Line
			w.followTo(p.Line)
			w.setStatus("Following " + p.Name + ": typing stops it")
			w.ribbonRefresh()
			return
		}
	}
	w.setStatus("They are not here any more")
}

// stopFollowing is Stop Following.
func (w *MainWindow) stopFollowing() {
	s := w.shared
	if s == nil || s.follow == "" {
		return
	}
	w.setStatus("Not following " + s.followName + " any more")
	s.follow = ""
	w.ribbonRefresh()
}

// typedWhileFollowing: an edit of this side's own stops following.
func (w *MainWindow) typedWhileFollowing() {
	if s := w.shared; s != nil && s.follow != "" {
		w.setStatus("You typed: not following " + s.followName + " any more")
		s.follow = ""
		w.ribbonFollow()
	}
}

// followMenu is Follow's menu: each one here, the one followed checked.
func (w *MainWindow) followMenu() *fyne.Menu {
	var items []*fyne.MenuItem
	for _, p := range w.hereNow() {
		item := fyne.NewMenuItem(p.Name, func() { w.follow(p.Replica) })
		item.Icon = peerDot(p.Replica)
		item.Checked = w.shared != nil && w.shared.follow == p.Replica
		items = append(items, item)
	}
	if len(items) == 0 {
		none := fyne.NewMenuItem("Nobody else is here yet", nil)
		none.Disabled = true
		items = append(items, none)
	}
	return fyne.NewMenu("Follow", items...)
}

// copyShareLink is Copy the Link.
func (w *MainWindow) copyShareLink() {
	if w.shared == nil {
		return
	}
	fyne.CurrentApp().Clipboard().SetContent(shareLink(w.shared.key))
	w.setStatus("Link copied: whoever has it can read and edit the script with you")
}

// leaveSession is Leave: the script stays here, as it is.
func (w *MainWindow) leaveSession() {
	if w.shared == nil {
		return
	}
	w.leaveSharing()
	w.setStatus("You left the shared script: it stays here as it is")
}

// peopleHere are the Session tab's people: a button each, in their
// colour, tapped to follow them; a word when nobody else is here.
func (r *ribbonBar) peopleHere() []fyne.CanvasObject {
	w := r.w
	var out []fyne.CanvasObject
	for _, p := range w.hereNow() {
		b := newRibbonButton(r, nil, false)
		b.label, b.icon = p.Name, peerDot(p.Replica)
		followed := w.shared.follow == p.Replica
		b.name = fmt.Sprintf("%s, on line %d: tap to follow", p.Name, p.Line+1)
		if followed {
			b.name = p.Name + ", followed: tap to stop"
		}
		b.onTap = func() { w.follow(p.Replica) }
		b.setSelected(followed)
		r.add(b)
		out = append(out, b)
	}
	if len(out) == 0 {
		l := widget.NewLabel("Only you so far")
		l.Importance = widget.LowImportance
		l.SizeName = theme.SizeNameCaptionText
		out = append(out, l)
	}
	return out
}

// sessionCommands are the Session tab's.
func sessionCommands() []*command {
	return []*command{
		{id: "session.follow", label: "Follow", tip: "Your cursor goes where theirs is, until you type", icon: theme.VisibilityIcon,
			when: sharedNow, enabled: othersHere, submenu: (*MainWindow).followMenu},
		{id: "session.stopFollowing", label: "Stop Following", icon: theme.VisibilityOffIcon, when: sharedNow, enabled: followingNow,
			run: (*MainWindow).stopFollowing},
		{id: "session.copyLink", label: "Copy the Link", ribbonLabel: "Copy Link", tip: "The shared script's link, to send: whoever has it can edit with you",
			icon: theme.ContentCopyIcon, when: sharedNow, run: (*MainWindow).copyShareLink},
		{id: "session.leave", label: "Leave the Shared Script", ribbonLabel: "Leave", tip: "Stop editing together: the script stays here as it is, and with the others",
			icon: theme.LogoutIcon, when: sharedNow, run: (*MainWindow).leaveSession},
	}
}

// sessionTab is the Session tab's layout.
func sessionTab() ribbonTab {
	return ribbonTab{title: "Session", when: sharedNow, contextual: true, groups: []ribbonGroup{
		{title: "Here", homes: []home{{id: "sessionPeople"}}},
		{title: "Follow", homes: []home{at("session.follow"), at("session.stopFollowing")}},
		{title: "Link", homes: []home{big("session.copyLink"), at("tableread.followLink")}},
		{title: "Leave", homes: []home{big("session.leave")}},
	}}
}

// sessionMenu is the Session menu, while the script is shared (the Menus look).
func sessionMenu() []home {
	return []home{at("session.follow"), at("session.stopFollowing"), sep(), at("session.copyLink"), at("tableread.followLink"), at("session.leave")}
}
