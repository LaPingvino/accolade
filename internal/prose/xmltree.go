package prose

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// node is an element of an XML document, read whole: DOCX and ODT parts
// are small enough, and a tree is easier to walk than a token stream.
type node struct {
	Name     xml.Name
	Attr     []xml.Attr
	Children []*node
	Text     string // character data, for a text node (Name.Local "")
}

// parseXML reads an XML document into a tree; its root is returned.
func parseXML(r io.Reader) (*node, error) {
	d := xml.NewDecoder(r)
	d.Strict = false
	root := &node{}
	stack := []*node{root}
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		top := stack[len(stack)-1]
		switch t := t.(type) {
		case xml.StartElement:
			n := &node{Name: t.Name, Attr: append([]xml.Attr(nil), t.Attr...)}
			top.Children = append(top.Children, n)
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			top.Children = append(top.Children, &node{Text: string(t)})
		}
	}
	for _, c := range root.Children {
		if c.Name.Local != "" {
			return c, nil
		}
	}
	return nil, errors.New("no XML in it")
}

// attr is the value of the attribute with that local name ("" if none).
func (n *node) attr(local string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// hasAttr: n has an attribute with that local name.
func (n *node) hasAttr(local string) bool {
	if n == nil {
		return false
	}
	for _, a := range n.Attr {
		if a.Name.Local == local {
			return true
		}
	}
	return false
}

// child is the first child element with that local name, nil if none.
func (n *node) child(local string) *node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Name.Local == local {
			return c
		}
	}
	return nil
}

// find is the first element with that local name below n, depth first.
func (n *node) find(local string) *node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Name.Local == local {
			return c
		}
		if f := c.find(local); f != nil {
			return f
		}
	}
	return nil
}

// all are the elements with that local name below n, not below another.
func (n *node) all(local string) []*node {
	var out []*node
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Name.Local == local {
			out = append(out, c)
		} else {
			out = append(out, c.all(local)...)
		}
	}
	return out
}

// text is all the character data below n.
func (n *node) text() string {
	if n == nil {
		return ""
	}
	if n.Name.Local == "" {
		return n.Text
	}
	var b strings.Builder
	for _, c := range n.Children {
		b.WriteString(c.text())
	}
	return b.String()
}

// zipPart reads a part of a zipped document; nil (no error) if it is not
// there.
func zipPart(z *zip.Reader, name string) (*node, error) {
	for _, f := range z.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			n, err := parseXML(io.LimitReader(rc, 256<<20))
			if err != nil {
				return nil, fmt.Errorf("%s: %v", name, err)
			}
			return n, nil
		}
	}
	return nil, nil
}

// xmlEsc escapes text for XML.
func xmlEsc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// zipWriter writes a zipped document's parts.
type zipWriter struct {
	z   *zip.Writer
	err error
}

func (w *zipWriter) part(name, body string, stored bool) {
	if w.err != nil {
		return
	}
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
	if stored {
		h.Method = zip.Store // ODT's mimetype: first, not compressed
	}
	f, err := w.z.CreateHeader(h)
	if err == nil {
		_, err = io.WriteString(f, body)
	}
	w.err = err
}
