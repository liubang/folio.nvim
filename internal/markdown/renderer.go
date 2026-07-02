// Copyright (c) 2026 The Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Authors: liubang (it.liubang@gmail.com)

package markdown

import (
	"bytes"
	"fmt"
	"html"
	"sort"

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

// lineIndex is a pre-computed index of newline byte offsets in the source
// document, enabling O(log n) line-number lookups instead of O(n) scans.
type lineIndex struct {
	// offsets[i] is the byte offset where line i+2 begins (after the i-th newline).
	// Line 1 starts at offset 0.
	offsets []int
}

// buildLineIndex scans the source once and records the byte offset of every newline.
func buildLineIndex(source []byte) *lineIndex {
	idx := &lineIndex{}
	for i, b := range source {
		if b == '\n' {
			idx.offsets = append(idx.offsets, i)
		}
	}
	return idx
}

// lineAt returns the 1-based line number for the given byte offset.
func (idx *lineIndex) lineAt(offset int) int {
	// sort.SearchInts returns the number of newlines before `offset`.
	return sort.SearchInts(idx.offsets, offset) + 1
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

// lineIndexKey is the attribute key used to store the pre-computed line index
// on the Document node so child renderers can retrieve it.
const lineIndexKey = "folio.lineIndex"

// SourceLineRenderer wraps the default HTML renderer to add data-source-line
// attributes to block-level nodes.
type SourceLineRenderer struct{}

// getLineIndex walks up to the Document node and retrieves the pre-computed
// lineIndex. Falls back to an empty index if not found.
func (r *SourceLineRenderer) getLineIndex(node ast.Node) *lineIndex {
	doc := node
	for doc.Parent() != nil {
		doc = doc.Parent()
	}
	if v, ok := doc.AttributeString(lineIndexKey); ok {
		if idx, ok2 := v.(*lineIndex); ok2 {
			return idx
		}
	}
	return &lineIndex{}
}

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
// It uses the pre-computed lineIndex stored in the node's owner document to
// achieve O(log n) lookup per call.
func (r *SourceLineRenderer) sourceLineAttr(source []byte, node ast.Node, idx *lineIndex) string {
	lines := node.Lines()
	if lines == nil || lines.Len() == 0 {
		return ""
	}
	seg := lines.At(0)
	line := idx.lineAt(seg.Start)
	return fmt.Sprintf(` data-source-line="%d"`, line)
}

func (r *SourceLineRenderer) renderDocument(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Build the line index once for the entire document and store it
		// so that all block renderers can use O(log n) lookups.
		idx := buildLineIndex(source)
		node.SetAttributeString(lineIndexKey, idx)
		w.WriteString("<article class=\"folio-document\">\n")
		return ast.WalkContinue, nil
	}
	w.WriteString("</article>\n")
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		fmt.Fprintf(w, "<h%d%s>", n.Level, r.sourceLineAttr(source, node, r.getLineIndex(node)))
	} else {
		fmt.Fprintf(w, "</h%d>\n", n.Level)
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("<p")
		w.WriteString(r.sourceLineAttr(source, node, r.getLineIndex(node)))
		w.WriteString(">")
	} else {
		w.WriteString("</p>\n")
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("<pre")
		w.WriteString(r.sourceLineAttr(source, node, r.getLineIndex(node)))
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
		w.WriteString(r.sourceLineAttr(source, node, r.getLineIndex(node)))
		w.WriteString("><code")
		lang := n.Language(source)
		if lang != nil {
			escapedLang := html.EscapeString(string(lang))
			fmt.Fprintf(w, " class=\"language-%s\" data-lang=\"%s\"", escapedLang, escapedLang)
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
		fmt.Fprintf(w, "<%s%s>\n", tag, r.sourceLineAttr(source, node, r.getLineIndex(node)))
	} else {
		fmt.Fprintf(w, "</%s>\n", tag)
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		fmt.Fprintf(w, "<blockquote%s>\n", r.sourceLineAttr(source, node, r.getLineIndex(node)))
	} else {
		w.WriteString("</blockquote>\n")
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	fmt.Fprintf(w, "<hr%s />\n", r.sourceLineAttr(source, node, r.getLineIndex(node)))
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
