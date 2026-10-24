package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Ex commands, as vim's command line has them (":" in vim and helix
// keys): a range of lines, then a command.
//
//	ranges     12  .  $  %  'a,'b ('< '> the visual lines)  /re/  ?re?  each +n -n
//	:s/re/to/gi    substitute (& the match, \1 a group; re in vim's style, \v Go's)
//	:g/re/cmd      run d, s or j on the lines that match; :v or :g! on the others
//	:d  :y  :j  :m addr  :t addr (:co)  :u  :12 (go to)
//	:w [file]  :q  :q!  :wq  :x
//
// run works on the lines alone; the window applies what it returns.

// exState is what ex commands keep between them.
type exState struct {
	lastPattern string
	history     []string
}

// exResult is what a command did.
type exResult struct {
	lines      []string
	line       int    // where the cursor goes (0-based)
	msg        string // to show
	yank       string // lines put in the register
	undo       bool
	write      bool
	file       string // :w file
	quit, keep bool   // keep: :q! (close without saving)
}

var errNotCommand = errors.New("not an editor command")

// run runs a command on lines with the cursor on line cur; visual is the
// last visual selection's lines ('< and '>), -1 if none.
func (st *exState) run(cmd string, lines []string, cur int, visual [2]int) (exResult, error) {
	res := exResult{lines: lines, line: cur}
	cmd = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(cmd), ":"))
	if cmd == "" {
		return res, nil
	}
	p := &exParser{s: cmd, lines: lines, cur: cur, visual: visual, st: st}
	from, to, given, err := p.rangeOf()
	if err != nil {
		return res, err
	}
	name, rest := p.command()
	if !given {
		from, to = cur, cur
		if name == "g" || name == "v" || name == "g!" || name == "w" || name == "wq" || name == "x" {
			from, to = 0, len(lines)-1
		}
	}
	if from > to {
		from, to = to, from
	}
	if from < 0 || to >= len(lines) {
		return res, fmt.Errorf("invalid range")
	}
	switch name {
	case "":
		res.line = to // :12 goes there
		return res, nil
	case "s":
		n, err := st.substitute(rest, lines, from, to)
		if err != nil {
			return res, err
		}
		res.lines, res.line = n.lines, n.line
		res.msg = n.msg
		return res, nil
	case "g", "g!", "v":
		return st.global(rest, name != "g", lines, from, to)
	case "d", "de", "del", "delete":
		res.yank = strings.Join(lines[from:to+1], "\n") + "\n"
		res.lines = append(append([]string(nil), lines[:from]...), lines[to+1:]...)
		if len(res.lines) == 0 {
			res.lines = []string{""}
		}
		res.line = min(from, len(res.lines)-1)
		res.msg = fmt.Sprintf("%d fewer lines", to-from+1)
		return res, nil
	case "y", "ya", "yank":
		res.yank = strings.Join(lines[from:to+1], "\n") + "\n"
		res.msg = fmt.Sprintf("%d lines yanked", to-from+1)
		return res, nil
	case "j", "join":
		if !given || from == to {
			to = min(from+1, len(lines)-1)
		}
		res.lines = join(lines, from, to)
		res.line = from
		return res, nil
	case "m", "mo", "move", "t", "co", "copy":
		dest, err := p.address(strings.TrimSpace(rest))
		if err != nil {
			return res, err
		}
		block := append([]string(nil), lines[from:to+1]...)
		if name == "t" || name == "co" || name == "copy" {
			res.lines = insertLines(lines, dest+1, block)
			res.line = dest + len(block)
			return res, nil
		}
		if dest >= from && dest < to {
			return res, errors.New("cannot move a range into itself")
		}
		without := append(append([]string(nil), lines[:from]...), lines[to+1:]...)
		if dest > to {
			dest -= len(block)
		}
		res.lines = insertLines(without, dest+1, block)
		res.line = dest + len(block)
		return res, nil
	case "u", "undo":
		res.undo = true
		return res, nil
	case "w", "write":
		res.write, res.file = true, strings.TrimSpace(rest)
		return res, nil
	case "wq", "x", "xit", "exit":
		res.write, res.quit = true, true
		return res, nil
	case "q", "quit":
		res.quit = true
		return res, nil
	case "q!", "quit!":
		res.quit, res.keep = true, true
		return res, nil
	}
	return res, fmt.Errorf("%w: %s", errNotCommand, name)
}

// exParser reads a command's range and name.
type exParser struct {
	s      string
	i      int
	lines  []string
	cur    int
	visual [2]int
	st     *exState
}

// rangeOf reads a range: from and to (0-based), and whether one was given.
func (p *exParser) rangeOf() (from, to int, given bool, err error) {
	if strings.HasPrefix(p.s, "%") {
		p.i = 1
		return 0, len(p.lines) - 1, true, nil
	}
	from, ok, err := p.addr()
	if err != nil || !ok {
		return 0, 0, false, err
	}
	to = from
	if p.i < len(p.s) && (p.s[p.i] == ',' || p.s[p.i] == ';') {
		if p.s[p.i] == ';' {
			p.cur = from
		}
		p.i++
		if to, ok, err = p.addr(); err != nil {
			return 0, 0, false, err
		} else if !ok {
			to = from
		}
	}
	return from, to, true, nil
}

// addr reads one address with its offsets.
func (p *exParser) addr() (line int, ok bool, err error) {
	s := p.s
	switch {
	case p.i >= len(s):
		return 0, false, nil
	case s[p.i] == '.':
		line, ok = p.cur, true
		p.i++
	case s[p.i] == '$':
		line, ok = len(p.lines)-1, true
		p.i++
	case s[p.i] >= '0' && s[p.i] <= '9':
		j := p.i
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		n, _ := strconv.Atoi(s[p.i:j])
		line, ok, p.i = max(n-1, 0), true, j
	case s[p.i] == '\'' && p.i+1 < len(s) && (s[p.i+1] == '<' || s[p.i+1] == '>'):
		if p.visual[0] < 0 {
			return 0, false, errors.New("no visual selection")
		}
		line, ok = p.visual[0], true
		if s[p.i+1] == '>' {
			line = p.visual[1]
		}
		p.i += 2
	case s[p.i] == '/' || s[p.i] == '?':
		d := s[p.i]
		j := strings.IndexByte(s[p.i+1:], d)
		pat := s[p.i+1:]
		if j >= 0 {
			pat = s[p.i+1 : p.i+1+j]
			p.i += j + 2
		} else {
			p.i = len(s)
		}
		re, err := p.st.compile(pat)
		if err != nil {
			return 0, false, err
		}
		n := len(p.lines)
		for k := 1; k <= n; k++ {
			l := (p.cur + k) % n
			if d == '?' {
				l = ((p.cur-k)%n + n) % n
			}
			if re.MatchString(p.lines[l]) {
				line, ok = l, true
				break
			}
		}
		if !ok {
			return 0, false, fmt.Errorf("pattern not found: %s", pat)
		}
	}
	for p.i < len(s) && (s[p.i] == '+' || s[p.i] == '-') { // offsets
		sign := 1
		if s[p.i] == '-' {
			sign = -1
		}
		j := p.i + 1
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		n := 1
		if j > p.i+1 {
			n, _ = strconv.Atoi(s[p.i+1 : j])
		}
		if !ok {
			line, ok = p.cur, true
		}
		line += sign * n
		p.i = j
	}
	return line, ok, nil
}

// address is a whole address (m and t's destination); 0 puts it first (-1).
func (p *exParser) address(s string) (int, error) {
	if s == "0" {
		return -1, nil
	}
	q := &exParser{s: s, lines: p.lines, cur: p.cur, visual: p.visual, st: p.st}
	l, ok, err := q.addr()
	if err == nil && (!ok || q.i != len(s)) {
		err = errors.New("invalid address")
	}
	return l, err
}

// command reads the command's name: letters (s and g end at their
// delimiter), or ! after q and g.
func (p *exParser) command() (name, rest string) {
	s := strings.TrimSpace(p.s[p.i:])
	j := 0
	for j < len(s) && unicode.IsLetter(rune(s[j])) {
		j++
	}
	name = s[:j]
	if name == "su" || name == "substitute" {
		name = "s"
	}
	if name == "global" {
		name = "g"
	}
	if name == "vglobal" {
		name = "v"
	}
	if j < len(s) && s[j] == '!' && (name == "g" || name == "q" || name == "quit") {
		name += "!"
		j++
	}
	return name, s[j:]
}

// substitute runs s/re/to/flags over lines from..to.
func (st *exState) substitute(arg string, lines []string, from, to int) (exResult, error) {
	res := exResult{lines: lines, line: from}
	if arg == "" {
		return res, errors.New("no pattern")
	}
	parts := splitDelimited(arg[1:], arg[0])
	if len(parts) < 1 {
		return res, errors.New("no pattern")
	}
	pat, rep, flags := parts[0], "", ""
	if len(parts) > 1 {
		rep = parts[1]
	}
	if len(parts) > 2 {
		flags = parts[2]
	}
	if pat == "" {
		pat = st.lastPattern
	}
	if strings.Contains(flags, "i") {
		pat = "\\c" + pat
	}
	re, err := st.compile(pat)
	if err != nil {
		return res, err
	}
	repl := vimReplacement(rep)
	all := strings.Contains(flags, "g")
	out := append([]string(nil), lines[:from]...)
	count := 0
	for i := from; i <= to; i++ {
		l := lines[i]
		if all {
			count += len(re.FindAllStringIndex(l, -1))
			l = re.ReplaceAllString(l, repl)
		} else if loc := re.FindStringSubmatchIndex(l); loc != nil {
			count++
			l = l[:loc[0]] + string(re.ExpandString(nil, repl, l, loc)) + l[loc[1]:]
		}
		out = append(out, strings.Split(l, "\n")...) // \r in the replacement breaks the line
		if l != lines[i] {
			res.line = len(out) - 1
		}
	}
	out = append(out, lines[to+1:]...)
	if count == 0 {
		return res, fmt.Errorf("pattern not found: %s", parts[0])
	}
	res.lines = out
	res.msg = fmt.Sprintf("%d substitutions", count)
	return res, nil
}

// global runs a command on the lines that match (or not) in from..to.
func (st *exState) global(arg string, invert bool, lines []string, from, to int) (exResult, error) {
	res := exResult{lines: lines, line: from}
	if arg == "" {
		return res, errors.New("no pattern")
	}
	parts := splitDelimited(arg[1:], arg[0])
	re, err := st.compile(parts[0])
	if err != nil {
		return res, err
	}
	cmd := "" // the cursor to the last line that matches
	if len(parts) > 1 {
		cmd = strings.TrimSpace(strings.Join(parts[1:], string(arg[0])))
	}
	marked := map[int]bool{}
	for i := from; i <= to; i++ {
		if re.MatchString(lines[i]) != invert {
			marked[i] = true
		}
	}
	if len(marked) == 0 {
		return res, fmt.Errorf("pattern not found: %s", parts[0])
	}
	// the marked lines, last first: commands on one do not move the
	// earlier ones
	out := append([]string(nil), lines...)
	for i := to; i >= from; i-- {
		if !marked[i] {
			continue
		}
		r, err := st.run(strconv.Itoa(i+1)+cmd, out, i, [2]int{-1, -1})
		if err != nil && !strings.Contains(err.Error(), "pattern not found") {
			return res, err
		}
		if err == nil {
			out = r.lines
		}
		res.line = min(i, len(out)-1)
	}
	res.lines = out
	res.msg = fmt.Sprintf("%d lines", len(marked))
	return res, nil
}

// splitDelimited splits s at unescaped d (a\/b stays one part, as a/b).
func splitDelimited(s string, d byte) []string {
	var parts []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == d:
			cur.WriteByte(d)
			i++
		case s[i] == '\\' && i+1 < len(s):
			cur.WriteByte(s[i])
			cur.WriteByte(s[i+1])
			i++
		case s[i] == d:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(s[i])
		}
	}
	return append(parts, cur.String())
}

// compile is a pattern in vim's style as Go's: \( \) \| \{ \} groups,
// alternatives and counts, \< \> word ends, ( ) | { } + ? themselves;
// \v (very magic) takes Go's syntax as it is, \c ignores case. An empty
// pattern is the last one.
func (st *exState) compile(pat string) (*regexp.Regexp, error) {
	if pat == "" {
		pat = st.lastPattern
	}
	if pat == "" {
		return nil, errors.New("no previous pattern")
	}
	st.lastPattern = pat
	fold := false
	if strings.Contains(pat, "\\c") {
		fold, pat = true, strings.ReplaceAll(pat, "\\c", "")
	}
	var b strings.Builder
	if fold {
		b.WriteString("(?i)")
	}
	if strings.HasPrefix(pat, "\\v") {
		b.WriteString(pat[2:])
	} else {
		for i := 0; i < len(pat); i++ {
			c := pat[i]
			if c == '\\' && i+1 < len(pat) {
				i++
				switch n := pat[i]; n {
				case '(', ')', '|', '{', '}', '+', '?':
					b.WriteByte(n)
				case '=':
					b.WriteByte('?')
				case '<', '>':
					b.WriteString(`\b`)
				case 'n':
					b.WriteString(`\n`)
				case 't':
					b.WriteString(`\t`)
				default:
					b.WriteByte('\\')
					b.WriteByte(n)
				}
				continue
			}
			if strings.IndexByte("()|{}+?", c) >= 0 {
				b.WriteByte('\\')
			}
			b.WriteByte(c)
		}
	}
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil, fmt.Errorf("pattern %s: %v", pat, err)
	}
	return re, nil
}

// vimReplacement is a replacement in vim's style as Go's: & the match,
// \1 a group, \& and \\ themselves, \r a new line.
func vimReplacement(rep string) string {
	var b strings.Builder
	for i := 0; i < len(rep); i++ {
		c := rep[i]
		switch {
		case c == '\\' && i+1 < len(rep):
			i++
			n := rep[i]
			switch {
			case n >= '0' && n <= '9':
				b.WriteString("${" + string(n) + "}")
			case n == 'r' || n == 'n':
				b.WriteByte('\n')
			case n == 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(n)
			}
		case c == '&':
			b.WriteString("${0}")
		case c == '$':
			b.WriteString("$$")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func join(lines []string, from, to int) []string {
	parts := []string{strings.TrimRight(lines[from], " ")}
	for _, l := range lines[from+1 : to+1] {
		if t := strings.TrimSpace(l); t != "" {
			parts = append(parts, t)
		}
	}
	out := append(append([]string(nil), lines[:from]...), strings.Join(parts, " "))
	return append(out, lines[to+1:]...)
}

func insertLines(lines []string, at int, block []string) []string {
	out := append([]string(nil), lines[:at]...)
	out = append(out, block...)
	return append(out, lines[at:]...)
}
