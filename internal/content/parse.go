package content

import (
	"bytes"
	"errors"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"go.yaml.in/yaml/v3"
)

// markdown renders CommonMark in safe mode: raw HTML is omitted.
var markdown = goldmark.New()

var errNoFrontmatter = errors.New("missing frontmatter: the file must start with a line \"---\" and the metadata must end with another \"---\" line")

// splitFrontmatter normalises line endings and a BOM, then returns the YAML between the
// opening and closing "---" lines and the Markdown body after them.
func splitFrontmatter(raw []byte) (meta, body []byte, err error) {
	s := strings.TrimPrefix(string(raw), "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	rest, ok := strings.CutPrefix(s, "---\n")
	if !ok {
		return nil, nil, errNoFrontmatter
	}
	if strings.HasPrefix(rest, "---\n") || rest == "---" {
		return nil, []byte(strings.TrimPrefix(rest, "---")), nil
	}
	m, b, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		m, ok = strings.CutSuffix(rest, "\n---")
		if !ok {
			return nil, nil, errNoFrontmatter
		}
	}
	return []byte(m), []byte(b), nil
}

// decodeStrict decodes YAML into v, rejecting unknown keys.
func decodeStrict(data []byte, v any) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(v)
}

// render returns the Markdown body as HTML and as plain text (for excerpts and search).
func render(body []byte) (template.HTML, string, error) {
	doc := markdown.Parser().Parse(gmtext.NewReader(body))
	var html bytes.Buffer
	if err := markdown.Renderer().Render(&html, body, doc); err != nil {
		return "", "", err
	}
	var plain strings.Builder
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Text:
			plain.Write(n.Segment.Value(body))
			if n.SoftLineBreak() || n.HardLineBreak() {
				plain.WriteByte(' ')
			}
		case *ast.RawHTML, *ast.HTMLBlock:
			return ast.WalkSkipChildren, nil
		default:
			if n.Type() == ast.TypeBlock {
				plain.WriteByte(' ')
			}
		}
		return ast.WalkContinue, nil
	})
	return template.HTML(html.String()), strings.Join(strings.Fields(plain.String()), " "), nil
}
