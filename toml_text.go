package main

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// TOML edited as text, so that what a person wrote stays: the comments,
// the order, the keys Accolade does not know. Every change is one
// operation on one path (a key set or taken out, a table taken out, a
// [[block]] moved); the text around it is left as it was. The editing is
// by lines, as people write TOML: a [table] or [[array]] header on a line
// of its own, a key's value starting on its line and going on until its
// brackets and strings close. What calls these reads the result back
// (tomlCheck) before it keeps it.

// tomlSection is a table of the text: its header's path, whether it is
// an [[array]] block, the header's line (-1 for the root, the keys before
// any header), where it starts (with the comments right above its
// header, which go with it) and where it ends (where the next starts).
type tomlSection struct {
	path               []string
	array              bool
	header, start, end int
}

// body are the lines of the section's keys.
func (s tomlSection) body() (int, int) { return s.header + 1, s.end }

// splitTOMLKey is a dotted key's parts: a."b.c".d is a, b.c, d.
func splitTOMLKey(s string) []string {
	var parts []string
	var cur strings.Builder
	quote := byte(0)
	quoted := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case quote != 0:
			if ch == '\\' && quote == '"' && i+1 < len(s) {
				i++
				cur.WriteByte(s[i])
			} else if ch == quote {
				quote = 0
			} else {
				cur.WriteByte(ch)
			}
		case ch == '"' || ch == '\'':
			quote, quoted = ch, true
		case ch == '.':
			parts = append(parts, strings.TrimSpace(cur.String()))
			cur.Reset()
			quoted = false
		case ch == ' ' || ch == '\t':
			if quoted {
				continue
			}
			cur.WriteByte(ch)
		default:
			cur.WriteByte(ch)
		}
	}
	return append(parts, strings.TrimSpace(cur.String()))
}

// tomlBareKey reports whether a key can be written without quotes.
func tomlBareKey(k string) bool {
	if k == "" {
		return false
	}
	for _, r := range k {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// tomlKeyText is a key as TOML writes it: bare, else quoted.
func tomlKeyText(k string) string {
	if tomlBareKey(k) {
		return k
	}
	return tomlValue(k)
}

// joinTOMLKey is a path written as a dotted key.
func joinTOMLKey(path []string) string {
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = tomlKeyText(p)
	}
	return strings.Join(parts, ".")
}

// tomlHeaderOf reads a header line: its path and whether it is an
// [[array]] block; ok false for another line.
func tomlHeaderOf(line string) (path []string, array, ok bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "[") {
		return nil, false, false
	}
	array = strings.HasPrefix(t, "[[")
	inner := t[1:]
	if array {
		inner = t[2:]
	}
	end := closingBracket(inner)
	if end < 0 {
		return nil, false, false
	}
	return splitTOMLKey(inner[:end]), array, true
}

// closingBracket is where the header's ] is, outside quotes.
func closingBracket(s string) int {
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		switch ch := s[i]; {
		case quote != 0:
			if ch == quote {
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == ']':
			return i
		}
	}
	return -1
}

// tomlHeader is the table a header line opens ("Elements.mine" for
// [Elements.mine]), ok false for another line and for an [[array]].
func tomlHeader(line string) (string, bool) {
	p, array, ok := tomlHeaderOf(line)
	if !ok || array {
		return "", false
	}
	return strings.Join(p, "."), true
}

// tomlLineKey is the key a line sets (its parts: a.b = 1 is a, b), nil
// for a line that sets none.
func tomlLineKey(line string) []string {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "[") {
		return nil
	}
	quote := byte(0)
	for i := 0; i < len(t); i++ {
		switch ch := t[i]; {
		case quote != 0:
			if ch == '\\' && quote == '"' {
				i++
			} else if ch == quote {
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == '=':
			return splitTOMLKey(t[:i])
		}
	}
	return nil
}

// tomlKeyOf is the key a line sets ("eo" for eo = [...]), "" for none.
func tomlKeyOf(line string) string {
	if k := tomlLineKey(line); k != nil {
		return strings.Join(k, ".")
	}
	return ""
}

// valueEnd is the line after the value started on lines[i] ends: its
// brackets closed, its strings (also """ over several lines) ended.
func valueEnd(lines []string, i int) int {
	depth := 0
	multi := "" // the multi-line string we are in: """ or '''
	started := false
	for j := i; j < len(lines); j++ {
		l := lines[j]
		k := 0
		if j == i { // after the =
			if eq := strings.Index(l, "="); eq >= 0 && tomlLineKey(l) != nil {
				k = eq + 1
			}
		}
		for ; k < len(l); k++ {
			ch := l[k]
			if multi != "" {
				if ch == '\\' && multi == `"""` {
					k++
					continue
				}
				if strings.HasPrefix(l[k:], multi) {
					k += 2
					multi = ""
				}
				continue
			}
			switch {
			case strings.HasPrefix(l[k:], `"""`) || strings.HasPrefix(l[k:], `'''`):
				multi = l[k : k+3]
				k += 2
				started = true
			case ch == '"' || ch == '\'':
				quote := ch
				for k++; k < len(l) && l[k] != quote; k++ {
					if l[k] == '\\' && quote == '"' {
						k++
					}
				}
				started = true
			case ch == '#':
				k = len(l)
			case ch == '[' || ch == '{':
				depth++
				started = true
			case ch == ']' || ch == '}':
				depth--
			case ch != ' ' && ch != '\t':
				started = true
			}
		}
		if multi == "" && depth <= 0 && (started || j > i) {
			return j + 1
		}
	}
	return len(lines)
}

// tomlSections are the text's tables, the root first.
func tomlSections(lines []string) []tomlSection {
	root := tomlSection{header: -1, start: 0, end: len(lines)}
	out := []tomlSection{root}
	for i := 0; i < len(lines); i++ {
		if p, array, ok := tomlHeaderOf(lines[i]); ok {
			start := i
			for start > out[len(out)-1].header+1 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), "#") {
				start--
			}
			out[len(out)-1].end = start
			out = append(out, tomlSection{path: p, array: array, header: i, start: start, end: len(lines)})
			continue
		}
		if tomlLineKey(lines[i]) != nil {
			i = valueEnd(lines, i) - 1
		}
	}
	return out
}

// tomlTable is the section that is the [table] with path (the root for
// none), ok false if the text has no header for it.
func tomlTable(secs []tomlSection, path []string) (tomlSection, bool) {
	for _, s := range secs {
		if !s.array && slices.Equal(s.path, path) {
			return s, true
		}
	}
	return tomlSection{}, false
}

// keyLine is the line in s that sets key (one part), -1 for none.
func keyLine(lines []string, s tomlSection, key string) int {
	from, to := s.body()
	for i := from; i < to; i++ {
		k := tomlLineKey(lines[i])
		if k == nil {
			continue
		}
		if len(k) == 1 && k[0] == key {
			return i
		}
		i = valueEnd(lines, i) - 1
	}
	return -1
}

// dottedIn reports whether s sets keys under key with dotted keys
// (key.sub = 1), which these edits leave alone.
func dottedIn(lines []string, s tomlSection, key string) bool {
	from, to := s.body()
	for i := from; i < to; i++ {
		if k := tomlLineKey(lines[i]); len(k) > 1 && k[0] == key {
			return true
		}
	}
	return false
}

// lastFilled is where a line goes at the end of a table ending before
// line end: after its last line that is not blank.
func lastFilled(lines []string, end int) int {
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return end
}

// indentOf is the white space a section's keys start with.
func indentOf(lines []string, s tomlSection) string {
	from, to := s.body()
	for i := from; i < to; i++ {
		if tomlLineKey(lines[i]) != nil {
			return lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
		}
	}
	if s.header >= 0 {
		h := lines[s.header]
		return h[:len(h)-len(strings.TrimLeft(h, " \t"))]
	}
	return ""
}

// replaceValue puts line (key = value) in the place of the key's line(s)
// at i, with their indent, and the comment at the end of a value on one
// line (Base = "bd" # Lexington's).
func replaceValue(lines []string, i int, line string) []string {
	indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
	end := valueEnd(lines, i)
	if end == i+1 {
		line += trailingComment(lines[i])
	}
	return slices.Replace(lines, i, end, indent+line)
}

// trailingComment is the comment ending a key's line, with the space
// before it ("" for none).
func trailingComment(l string) string {
	eq := strings.Index(l, "=")
	if eq < 0 {
		return ""
	}
	for k := eq + 1; k < len(l); k++ {
		switch ch := l[k]; ch {
		case '"', '\'':
			if strings.HasPrefix(l[k:], `"""`) || strings.HasPrefix(l[k:], "'''") {
				return "" // a string over lines: not here
			}
			for k++; k < len(l) && l[k] != ch; k++ {
				if l[k] == '\\' && ch == '"' {
					k++
				}
			}
		case '#':
			j := k
			for j > eq+1 && (l[j-1] == ' ' || l[j-1] == '\t') {
				j--
			}
			return l[j:]
		}
	}
	return ""
}

// errInline: the path goes through a value written inline (a = {…}),
// which is set as a whole (setPath does).
type errInline struct{ at int }

func (e errInline) Error() string { return "written inline" }

// inlineAt is how much of path is the table or key that holds the rest
// inline: a key path[n-1] set in table path[:n-1] whose value is
// path[n:] (n > 0), or 0 if none.
func inlineAt(lines []string, secs []tomlSection, path []string) int {
	for n := len(path) - 1; n >= 1; n-- {
		s, ok := tomlTable(secs, path[:n-1])
		if !ok && n-1 > 0 {
			continue
		}
		if keyLine(lines, s, path[n-1]) >= 0 {
			return n
		}
	}
	return 0
}

// setKey sets path's key to value (written TOML): its line(s) replaced,
// else added at the end of its table (the root: at the top), else the
// table added at the end of the text. A path through a value written
// inline is errInline; one set with dotted keys an error.
func setKey(text string, path []string, value string) (string, error) {
	lines := strings.Split(text, "\n")
	secs := tomlSections(lines)
	table, key := path[:len(path)-1], path[len(path)-1]
	line := tomlKeyText(key) + " = " + value
	if s, ok := tomlTable(secs, table); ok || len(table) == 0 {
		if dottedIn(lines, s, key) {
			return "", fmt.Errorf("%s is written with dotted keys", joinTOMLKey(path))
		}
		if i := keyLine(lines, s, key); i >= 0 {
			return strings.Join(replaceValue(lines, i, line), "\n"), nil
		}
		if len(table) == 0 {
			from, to := s.body()
			// after the root's last key, else at the very top
			at := 0
			for i := from; i < to; i++ {
				if tomlLineKey(lines[i]) != nil {
					i = valueEnd(lines, i) - 1
					at = i + 1
				}
			}
			return strings.Join(slices.Insert(lines, at, line), "\n"), nil
		}
		at := lastFilled(lines, s.end)
		if at <= s.header {
			at = s.header + 1
		}
		return strings.Join(slices.Insert(lines, at, indentOf(lines, s)+line), "\n"), nil
	}
	if n := inlineAt(lines, secs, path); n > 0 {
		return "", errInline{at: n}
	}
	return appendTables(text, "["+joinTOMLKey(table)+"]\n"+line+"\n"), nil
}

// dropKey takes path's key (its line(s)) out of its table, or the
// tables at path if it names tables; nothing if the text has neither.
func dropKey(text string, path []string) (string, error) {
	lines := strings.Split(text, "\n")
	secs := tomlSections(lines)
	table, key := path[:len(path)-1], path[len(path)-1]
	if s, ok := tomlTable(secs, table); ok || len(table) == 0 {
		if i := keyLine(lines, s, key); i >= 0 {
			return strings.Join(slices.Delete(lines, i, valueEnd(lines, i)), "\n"), nil
		}
		if dottedIn(lines, s, key) {
			return "", fmt.Errorf("%s is written with dotted keys", joinTOMLKey(path))
		}
	}
	if n := inlineAt(lines, secs, path); n > 0 {
		return "", errInline{at: n}
	}
	return dropTablesAt(text, path), nil
}

// dropTablesAt takes out the table at path and every one under it
// ([path.…] and [[path.…]]), with the comments right above them.
func dropTablesAt(text string, path []string) string {
	lines := strings.Split(text, "\n")
	var out []string
	last := 0
	for _, s := range tomlSections(lines)[1:] {
		if len(s.path) >= len(path) && slices.Equal(s.path[:len(path)], path) {
			out = append(out, lines[last:s.start]...)
			last = s.end
		}
	}
	out = append(out, lines[last:]...)
	return strings.Join(out, "\n")
}

// arrayBlocks are the [[path]] blocks, in order.
func arrayBlocks(lines []string, path []string) []tomlSection {
	var out []tomlSection
	secs := tomlSections(lines)
	for i, s := range secs {
		if s.array && slices.Equal(s.path, path) {
			// a block goes on over the tables under it ([path.sub])
			for j := i + 1; j < len(secs) && len(secs[j].path) > len(path) && slices.Equal(secs[j].path[:len(path)], path); j++ {
				s.end = secs[j].end
			}
			out = append(out, s)
		}
	}
	return out
}

// moveBlock moves the i-th [[path]] block to be the j-th, its comments
// with it.
func moveBlock(text string, path []string, i, j int) (string, error) {
	lines := strings.Split(text, "\n")
	blocks := arrayBlocks(lines, path)
	if i < 0 || j < 0 || i >= len(blocks) || j >= len(blocks) {
		return "", errors.New("no such block")
	}
	if i == j {
		return text, nil
	}
	chunk := func(b tomlSection) []string {
		c := slices.Clone(lines[b.start:lastFilled(lines, b.end)])
		return append(c, "")
	}
	order := make([]int, len(blocks))
	for k := range order {
		order[k] = k
	}
	moved := order[i]
	order = slices.Delete(order, i, i+1)
	order = slices.Insert(order, j, moved)
	var out []string
	last := 0
	for k, b := range blocks {
		out = append(out, lines[last:b.start]...)
		out = append(out, chunk(blocks[order[k]])...)
		last = b.end
		if k == len(blocks)-1 && b.end == len(lines) {
			out = out[:lastFilled(out, len(out))]
			out = append(out, "")
		}
	}
	out = append(out, lines[last:]...)
	return strings.Join(out, "\n"), nil
}

// dropBlock takes the i-th [[path]] block out.
func dropBlock(text string, path []string, i int) (string, error) {
	lines := strings.Split(text, "\n")
	blocks := arrayBlocks(lines, path)
	if i < 0 || i >= len(blocks) {
		return "", errors.New("no such block")
	}
	return strings.Join(slices.Delete(lines, blocks[i].start, blocks[i].end), "\n"), nil
}

// addBlock adds a [[path]] block with the keys of m after the last one
// (at the end if none).
func addBlock(text string, path []string, m map[string]any, order []string) string {
	lines := strings.Split(text, "\n")
	blocks := arrayBlocks(lines, path)
	indent := ""
	var b []string
	if len(blocks) > 0 {
		last := blocks[len(blocks)-1]
		indent = indentOf(lines, last)
		h := lines[last.header]
		b = append(b, h[:len(h)-len(strings.TrimLeft(h, " \t"))]+"[["+joinTOMLKey(path)+"]]")
	} else {
		b = append(b, "[["+joinTOMLKey(path)+"]]")
	}
	for _, k := range keysInOrder(m, order) {
		b = append(b, indent+tomlKeyText(k)+" = "+tomlInline(m[k]))
	}
	if len(blocks) == 0 {
		return appendTables(text, strings.Join(b, "\n")+"\n")
	}
	at := lastFilled(lines, blocks[len(blocks)-1].end)
	ins := append([]string{""}, b...)
	return strings.Join(slices.Insert(lines, at, ins...), "\n")
}

// setInBlock sets key in the i-th [[path]] block ("" value: takes it out).
func setInBlock(text string, path []string, i int, key, value string) (string, error) {
	lines := strings.Split(text, "\n")
	blocks := arrayBlocks(lines, path)
	if i < 0 || i >= len(blocks) {
		return "", errors.New("no such block")
	}
	s := blocks[i]
	s.end = tableEnd(lines, s)
	if at := keyLine(lines, s, key); at >= 0 {
		if value == "" {
			return strings.Join(slices.Delete(lines, at, valueEnd(lines, at)), "\n"), nil
		}
		return strings.Join(replaceValue(lines, at, tomlKeyText(key)+" = "+value), "\n"), nil
	}
	if value == "" {
		return text, nil
	}
	at := lastFilled(lines, s.end)
	return strings.Join(slices.Insert(lines, at, indentOf(lines, s)+tomlKeyText(key)+" = "+value), "\n"), nil
}

// tableEnd is where a block's own keys end (before a table under it).
func tableEnd(lines []string, s tomlSection) int {
	for i := s.header + 1; i < s.end; i++ {
		if _, _, ok := tomlHeaderOf(lines[i]); ok {
			return i
		}
		if tomlLineKey(lines[i]) != nil {
			i = valueEnd(lines, i) - 1
		}
	}
	return s.end
}

// keysInOrder are m's keys: those of order first, in that order, then
// the rest sorted.
func keysInOrder(m map[string]any, order []string) []string {
	var out []string
	for _, k := range order {
		if _, ok := m[k]; ok {
			out = append(out, k)
		}
	}
	var rest []string
	for k := range m {
		if !slices.Contains(order, k) {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// tomlValue is v written as a TOML value.
func tomlValue(v any) string {
	var b strings.Builder
	_ = toml.NewEncoder(&b).Encode(struct{ V any }{v})
	return strings.TrimSpace(strings.TrimPrefix(b.String(), "V = "))
}

// tomlInline is v written as a TOML value on one line, a table as an
// inline table.
func tomlInline(v any) string {
	switch x := v.(type) {
	case map[string]any:
		var parts []string
		for _, k := range keysInOrder(x, nil) {
			parts = append(parts, tomlKeyText(k)+" = "+tomlInline(x[k]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case map[string]string:
		m := map[string]any{}
		for k, s := range x {
			m[k] = s
		}
		return tomlInline(m)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = tomlInline(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []string:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = tomlValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return tomlValue(v)
}

// tomlMap is the text read as TOML.
func tomlMap(text string) (map[string]any, error) {
	m := map[string]any{}
	_, err := toml.Decode(text, &m)
	return m, err
}

// mapAt is the value at path in m, and whether there is one.
func mapAt(m map[string]any, path []string) (any, bool) {
	var cur any = m
	for _, p := range path {
		t, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		if cur, ok = t[p]; !ok {
			return nil, false
		}
	}
	return cur, true
}

// withAt is a copy of m with path set to v (deleted for nil), the tables
// on the way made.
func withAt(m map[string]any, path []string, v any) map[string]any {
	out := make(map[string]any, len(m))
	for k, x := range m {
		out[k] = x
	}
	if len(path) == 1 {
		if v == nil {
			delete(out, path[0])
		} else {
			out[path[0]] = v
		}
		return out
	}
	sub, _ := out[path[0]].(map[string]any)
	if sub == nil {
		if v == nil {
			return out
		}
		sub = map[string]any{}
	}
	out[path[0]] = withAt(sub, path[1:], v)
	return out
}

// setPath sets path to v (a Go value) in the text: as a key of its
// table, or the inline value holding it rewritten whole.
func setPath(text string, path []string, v any) (string, error) {
	out, err := setKey(text, path, tomlInline(v))
	var in errInline
	if errors.As(err, &in) {
		return rewriteInline(text, path, in.at, v)
	}
	return out, err
}

// dropPath takes path out of the text (nothing if it is not there).
func dropPath(text string, path []string) (string, error) {
	out, err := dropKey(text, path)
	var in errInline
	if errors.As(err, &in) {
		return rewriteInline(text, path, in.at, nil)
	}
	return out, err
}

// rewriteInline sets path to v (nil: takes it out) inside the value of
// path[:at], written inline, which is written again whole.
func rewriteInline(text string, path []string, at int, v any) (string, error) {
	m, err := tomlMap(text)
	if err != nil {
		return "", err
	}
	holder, _ := mapAt(m, path[:at])
	t, ok := holder.(map[string]any)
	if !ok {
		return "", fmt.Errorf("%s is not a table", joinTOMLKey(path[:at]))
	}
	t = withAt(t, path[at:], v)
	return setKey(text, path[:at], tomlInline(t))
}
