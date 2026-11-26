package cite

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseCSLJSON reads a CSL-JSON library: an array of items (Zotero's
// and Better BibTeX's export), or one item alone. Fields that are
// numbers (a volume, a numeric id) are taken as text; an item without an
// id is left out.
func ParseCSLJSON(data []byte) ([]*Entry, error) {
	data = []byte(strings.TrimPrefix(string(data), "\ufeff"))
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		var one map[string]json.RawMessage
		if json.Unmarshal(data, &one) != nil {
			return nil, fmt.Errorf("not CSL-JSON: %v", err)
		}
		items = []map[string]json.RawMessage{one}
	}
	var out []*Entry
	for _, it := range items {
		e := &Entry{
			Key:       jsonText(it["id"]),
			Type:      jsonText(it["type"]),
			Title:     cleanHTML(jsonText(it["title"])),
			Authors:   jsonNames(it["author"]),
			Editors:   jsonNames(it["editor"]),
			Year:      jsonYear(it["issued"]),
			Container: cleanHTML(jsonText(it["container-title"])),
			Publisher: jsonText(it["publisher"]),
			Place:     jsonText(it["publisher-place"]),
			Volume:    jsonText(it["volume"]),
			Issue:     jsonText(it["issue"]),
			Pages:     jsonText(it["page"]),
			Edition:   jsonText(it["edition"]),
			Genre:     jsonText(it["genre"]),
			DOI:       jsonText(it["DOI"]),
			URL:       jsonText(it["URL"]),
			Note:      jsonText(it["note"]),
		}
		if e.Key == "" {
			continue
		}
		e.setDate("issued", jsonDate(it["issued"]))
		for k, raw := range it {
			if jsonUsed[k] {
				continue
			}
			if ns := jsonNames(raw); len(ns) > 0 {
				e.setNames(k, ns)
			} else if d := jsonDate(raw); d[0] != 0 {
				e.setDate(k, d)
			} else {
				e.setMore(k, cleanHTML(jsonText(raw)))
			}
		}
		if e.Container == "" {
			e.Container = cleanHTML(jsonText(it["collection-title"]))
		}
		if e.Place == "" {
			e.Place = jsonText(it["event-place"])
		}
		out = append(out, e)
	}
	if len(out) == 0 && len(items) > 0 {
		return nil, fmt.Errorf("not CSL-JSON: no item has an id")
	}
	return out, nil
}

// jsonText is a string or a number as text ("" for anything else).
func jsonText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}

// jsonNames reads CSL's names: family/given (with particles and a
// suffix), or a literal name.
func jsonNames(raw json.RawMessage) []Name {
	if len(raw) == 0 {
		return nil
	}
	var list []map[string]json.RawMessage
	if json.Unmarshal(raw, &list) != nil {
		return nil
	}
	var out []Name
	for _, m := range list {
		n := Name{
			Family:  jsonText(m["family"]),
			Given:   jsonText(m["given"]),
			Suffix:  jsonText(m["suffix"]),
			Literal: jsonText(m["literal"]),
		}
		n.Particle = strings.TrimSpace(jsonText(m["non-dropping-particle"]))
		if d := jsonText(m["dropping-particle"]); d != "" {
			n.Given = strings.TrimSpace(n.Given + " " + d)
		}
		if n.Literal == "" && n.Family == "" && n.Given != "" {
			n.Literal, n.Given = n.Given, ""
		}
		if n.Literal != "" || n.Family != "" {
			out = append(out, n)
		}
	}
	return out
}

// jsonYear is a CSL date's year: from date-parts (numbers or strings),
// else the first four digits of raw or literal.
func jsonYear(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var d struct {
		Parts   [][]json.RawMessage `json:"date-parts"`
		Raw     string              `json:"raw"`
		Literal string              `json:"literal"`
	}
	if json.Unmarshal(raw, &d) != nil {
		return yearIn(jsonText(raw))
	}
	if len(d.Parts) > 0 && len(d.Parts[0]) > 0 {
		if y := jsonText(d.Parts[0][0]); y != "" {
			if _, err := strconv.Atoi(y); err == nil {
				return y
			}
		}
	}
	if y := yearIn(d.Raw); y != "" {
		return y
	}
	return yearIn(d.Literal)
}

// jsonUsed are the CSL-JSON fields Entry's own fields hold.
var jsonUsed = map[string]bool{"id": true, "type": true, "title": true, "author": true, "editor": true,
	"issued": true, "container-title": true, "publisher": true, "publisher-place": true, "volume": true,
	"issue": true, "page": true, "edition": true, "genre": true, "DOI": true, "URL": true, "note": true}

// jsonDate is a CSL date as year, month, day: from date-parts, else raw
// or literal read as an ISO date (or a year in it).
func jsonDate(raw json.RawMessage) [3]int {
	var d struct {
		Parts   [][]json.RawMessage `json:"date-parts"`
		Raw     string              `json:"raw"`
		Literal string              `json:"literal"`
	}
	if len(raw) == 0 || raw[0] != '{' || json.Unmarshal(raw, &d) != nil {
		return [3]int{}
	}
	var out [3]int
	if len(d.Parts) > 0 {
		for i, p := range d.Parts[0] {
			if i < 3 {
				out[i], _ = strconv.Atoi(jsonText(p))
			}
		}
		if out[0] != 0 {
			return out
		}
	}
	return isoDate(cmpOr(d.Raw, d.Literal))
}

// isoDate reads 1999, 1999-05 or 1999-05-12 (and the year of anything
// else with one).
func isoDate(s string) [3]int {
	var out [3]int
	parts := strings.Split(strings.TrimSpace(s), "-")
	if len(parts) <= 3 && len(parts[0]) == 4 {
		for i, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil {
				return [3]int{atoiOr(yearIn(s))}
			}
			out[i] = n
		}
		return out
	}
	return [3]int{atoiOr(yearIn(s))}
}

func atoiOr(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// yearIn is the first run of four digits in s.
func yearIn(s string) string {
	run := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			run++
			if run == 4 && (i+1 == len(s) || s[i+1] < '0' || s[i+1] > '9') {
				return s[i-3 : i+1]
			}
			continue
		}
		run = 0
	}
	return ""
}

// cleanHTML drops the few tags CSL allows in titles (<i>, <b>, <sup>,
// <span class="nocase">).
func cleanHTML(s string) string {
	if !strings.Contains(s, "<") {
		return s
	}
	var b strings.Builder
	for {
		i := strings.IndexByte(s, '<')
		if i < 0 {
			break
		}
		j := strings.IndexByte(s[i:], '>')
		if j < 0 {
			break
		}
		tag := strings.ToLower(strings.TrimLeft(s[i+1:i+j], "/"))
		known := false
		for _, t := range []string{"i", "b", "sup", "sub", "span", "sc"} {
			if tag == t || strings.HasPrefix(tag, t+" ") {
				known = true
			}
		}
		b.WriteString(s[:i])
		if !known {
			b.WriteString(s[i : i+j+1])
		}
		s = s[i+j+1:]
	}
	b.WriteString(s)
	return b.String()
}
