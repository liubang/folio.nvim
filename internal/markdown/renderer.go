package markdown

import (
	"bytes"
	"fmt"
	"html"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	ghtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// Renderer produces HTML with data-source-line attributes injected into block-level
// elements for scroll-sync between Neovim and the browser.
type Renderer struct {
	md goldmark.Markdown
}

// NewRenderer returns a Goldmark-compatible Markdown → HTML converter that
// injects data-source-line attributes on every block node.
func NewRenderer() *Renderer {
	return &Renderer{
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(
				ghtml.WithUnsafe(),
				renderer.WithNodeRenderers(
					util.Prioritized(&SourceLineRenderer{}, 100),
				),
			),
		),
	}
}

// SourceLineRenderer wraps the default HTML renderer to add data-source-line
// attributes to block-level nodes.
type SourceLineRenderer struct{}

// RegisterFuncs implements renderer.Renderer.
func (r *SourceLineRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindDocument, r.renderDocument)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)
}

// sourceLineAttr computes the 1-based source line number from the block node's
// first line segment and returns a data-source-line="N" attribute string.
func (r *SourceLineRenderer) sourceLineAttr(source []byte, node ast.Node) string {
	lines := node.Lines()
	if lines == nil || lines.Len() == 0 {
		return ""
	}
	seg := lines.At(0)
	line := 1
	for i := 0; i < seg.Start; i++ {
		if source[i] == '\n' {
			line++
		}
	}
	return fmt.Sprintf(` data-source-line="%d"`, line)
}

func (r *SourceLineRenderer) renderDocument(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("<article class=\"folio-document\">\n")
		return ast.WalkContinue, nil
	}
	w.WriteString("</article>\n")
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		fmt.Fprintf(w, "<h%d%s>", n.Level, r.sourceLineAttr(source, node))
	} else {
		fmt.Fprintf(w, "</h%d>\n", n.Level)
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("<p")
		w.WriteString(r.sourceLineAttr(source, node))
		w.WriteString(">")
	} else {
		w.WriteString("</p>\n")
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("<pre")
		w.WriteString(r.sourceLineAttr(source, node))
		w.WriteString("><code>")
		r.writeCodeLines(w, source, node)
		w.WriteString("</code></pre>\n")
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if entering {
		w.WriteString("<pre")
		w.WriteString(r.sourceLineAttr(source, node))
		w.WriteString("><code")
		lang := n.Language(source)
		if lang != nil {
			fmt.Fprintf(w, " class=\"language-%s\" data-lang=\"%s\"", lang, lang)
		}
		w.WriteByte('>')
		r.writeCodeLines(w, source, n)
		w.WriteString("</code></pre>\n")
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.List)
	tag := "ul"
	if n.IsOrdered() {
		tag = "ol"
	}
	if entering {
		fmt.Fprintf(w, "<%s%s>\n", tag, r.sourceLineAttr(source, node))
	} else {
		fmt.Fprintf(w, "</%s>\n", tag)
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		fmt.Fprintf(w, "<blockquote%s>\n", r.sourceLineAttr(source, node))
	} else {
		w.WriteString("</blockquote>\n")
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	fmt.Fprintf(w, "<hr%s />\n", r.sourceLineAttr(source, node))
	return ast.WalkContinue, nil
}

// Convert parses the given Markdown source and returns HTML with injected
// data-source-line attributes.
func (r *Renderer) Convert(source []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.md.Convert(source, &buf); err != nil {
		return nil, fmt.Errorf("markdown convert: %w", err)
	}
	return buf.Bytes(), nil
}

// writeCodeLines writes the lines of a code block to w, HTML-escaping each line.
// Goldmark stores lines as segments that already include trailing newlines.
func (r *SourceLineRenderer) writeCodeLines(w util.BufWriter, source []byte, node ast.Node) {
	for i := 0; i < node.Lines().Len(); i++ {
		seg := node.Lines().At(i)
		w.WriteString(html.EscapeString(string(source[seg.Start:seg.Stop])))
	}
}
