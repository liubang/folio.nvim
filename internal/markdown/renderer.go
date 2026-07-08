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
	"regexp"
	"slices"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extAST "github.com/yuin/goldmark/extension/ast"
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
	// slices.BinarySearch returns the insertion point for offset, i.e. the
	// number of newlines strictly before it — equivalently, offset's
	// 0-based line number.
	pos, _ := slices.BinarySearch(idx.offsets, offset)
	return pos + 1
}

// NewRenderer returns a Goldmark-compatible Markdown → HTML converter that
// injects data-source-line attributes on every block node.
func NewRenderer() *Renderer {
	return &Renderer{
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM, extension.Footnote),
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

// slugTrackerKey is the attribute key used to store the per-document slug
// de-duplication tracker so that repeated heading texts (e.g. two sections
// both titled "Overview") get distinct, stable anchor IDs.
const slugTrackerKey = "folio.slugTracker"

// slugTracker de-duplicates heading anchor slugs within a single document,
// mirroring GitHub's behavior of suffixing repeats with -1, -2, etc.
type slugTracker struct {
	seen map[string]int
}

// next returns a unique slug derived from text, appending a numeric suffix
// if the base slug has already been used in this document.
func (t *slugTracker) next(text string) string {
	base := slugify(text)
	if base == "" {
		base = "section"
	}
	n := t.seen[base]
	t.seen[base] = n + 1
	if n == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, n)
}

// slugNonWordRe matches runs of characters that are not letters, numbers,
// spaces, or hyphens — GitHub strips these when generating heading anchors.
var slugNonWordRe = regexp.MustCompile(`[^\p{L}\p{N}\s-]`)

// slugify converts heading text into a GitHub-style anchor slug: lowercase,
// punctuation stripped, whitespace collapsed to single hyphens.
func slugify(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	s = slugNonWordRe.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), "-")
	return s
}

// SourceLineRenderer wraps the default HTML renderer to add data-source-line
// attributes to block-level nodes.
//
// It is stateless by design: goldmark's NodeRenderer is registered once per
// Renderer and may be reused across documents, so per-document state (the
// lineIndex and slugTracker) cannot live in struct fields. Instead it is
// stashed on the Document node's attributes by renderDocument and retrieved
// via docAttr below.
type SourceLineRenderer struct{}

// rootDocument walks up the tree to the owning Document node.
func rootDocument(node ast.Node) ast.Node {
	for node.Parent() != nil {
		node = node.Parent()
	}
	return node
}

// docAttr retrieves a typed, per-document value previously stored (by
// renderDocument) on the Document node's attributes, identified by key.
// If the attribute is missing or has an unexpected type, fallback() supplies
// a zero-value substitute so callers never have to nil-check the result.
func docAttr[T any](node ast.Node, key string, fallback func() T) T {
	if v, ok := rootDocument(node).AttributeString(key); ok {
		if t, ok2 := v.(T); ok2 {
			return t
		}
	}
	return fallback()
}

// getLineIndex retrieves the pre-computed lineIndex for node's document.
// Falls back to an empty index if not found.
func (r *SourceLineRenderer) getLineIndex(node ast.Node) *lineIndex {
	return docAttr(node, lineIndexKey, func() *lineIndex { return &lineIndex{} })
}

// getSlugTracker retrieves the per-document slugTracker used to assign
// unique heading anchor IDs.
func (r *SourceLineRenderer) getSlugTracker(node ast.Node) *slugTracker {
	return docAttr(node, slugTrackerKey, func() *slugTracker { return &slugTracker{seen: map[string]int{}} })
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
	reg.Register(extAST.KindTable, r.renderTable)
	reg.Register(extAST.KindFootnote, r.renderFootnote)
	reg.Register(extAST.KindFootnoteList, r.renderFootnoteList)
}

// sourceLineAttr computes the 1-based source line number of the block node
// and returns a data-source-line="N" attribute string. It uses the
// pre-computed lineIndex stored in the node's owner document to achieve
// O(log n) lookup per call. Returns an empty string if the node carries no
// source position (e.g. a synthetic container with no line-bearing descendant).
func (r *SourceLineRenderer) sourceLineAttr(node ast.Node) string {
	line := r.nodeStartLine(node, r.getLineIndex(node))
	if line <= 0 {
		return ""
	}
	return fmt.Sprintf(` data-source-line="%d"`, line)
}

// nodeStartLine returns the 1-based source line of a block node. Container
// nodes such as GFM tables carry no line segments of their own (the segments
// live on their children), so we fall back to the first descendant that has a
// line segment. This keeps scroll-sync working when the cursor is inside a
// table or any other container that does not own its lines.
//
// Only block nodes are descended into: calling Lines() on an inline node
// panics inside goldmark ("can not call with inline nodes"), so we stop at
// inline children rather than recursing through them.
func (r *SourceLineRenderer) nodeStartLine(node ast.Node, idx *lineIndex) int {
	if node.Type() == ast.TypeInline {
		return 0
	}
	if lines := node.Lines(); lines != nil && lines.Len() > 0 {
		return idx.lineAt(lines.At(0).Start)
	}
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		if line := r.nodeStartLine(c, idx); line > 0 {
			return line
		}
	}
	return 0
}

func (r *SourceLineRenderer) renderDocument(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Build the line index once for the entire document and store it
		// so that all block renderers can use O(log n) lookups.
		idx := buildLineIndex(source)
		node.SetAttributeString(lineIndexKey, idx)
		// Fresh slug tracker per document so heading anchor IDs (and their
		// de-duplication counters) don't leak across renders.
		node.SetAttributeString(slugTrackerKey, &slugTracker{seen: map[string]int{}})
		w.WriteString("<article class=\"folio-document\">\n")
		return ast.WalkContinue, nil
	}
	w.WriteString("</article>\n")
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		id := r.getSlugTracker(node).next(string(n.Text(source)))
		fmt.Fprintf(w, "<h%d id=\"%s\"%s>", n.Level, html.EscapeString(id), r.sourceLineAttr(node))
	} else {
		fmt.Fprintf(w, "</h%d>\n", n.Level)
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("<p")
		w.WriteString(r.sourceLineAttr(node))
		w.WriteString(">")
	} else {
		w.WriteString("</p>\n")
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("<pre")
		w.WriteString(r.sourceLineAttr(node))
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
		w.WriteString(r.sourceLineAttr(node))
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
		fmt.Fprintf(w, "<%s%s>\n", tag, r.sourceLineAttr(node))
	} else {
		fmt.Fprintf(w, "</%s>\n", tag)
	}
	return ast.WalkContinue, nil
}

func (r *SourceLineRenderer) renderBlockquote(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return r.renderSimpleBlock(w, "blockquote", node, entering)
}

func (r *SourceLineRenderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	fmt.Fprintf(w, "<hr%s />\n", r.sourceLineAttr(node))
	return ast.WalkContinue, nil
}

// renderTable wraps the GFM table node with a data-source-line attribute so
// cursor scroll-sync keeps working when the cursor is inside a table. The
// inner thead/tbody/tr/td nodes are still rendered by goldmark's default
// HTML renderer (we only override the Table kind itself).
func (r *SourceLineRenderer) renderTable(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return r.renderSimpleBlock(w, "table", node, entering)
}

// renderSimpleBlock writes `<tag data-source-line="N">` on entry and
// `</tag>` on exit. It factors out the handful of block renderers (e.g.
// blockquote, table) whose HTML shell differs only by tag name.
func (r *SourceLineRenderer) renderSimpleBlock(w util.BufWriter, tag string, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		fmt.Fprintf(w, "<%s%s>\n", tag, r.sourceLineAttr(node))
	} else {
		fmt.Fprintf(w, "</%s>\n", tag)
	}
	return ast.WalkContinue, nil
}

// renderFootnote overrides extension.Footnote's default <li id="fn:N">...</li>
// renderer to additionally inject a data-source-line attribute, so scroll-sync
// also works when the cursor is on a footnote definition at the bottom of the
// document. goldmark's renderer.NodeRenderer registry only allows one handler
// per node kind, so we fully re-implement the (small) <li> shell here rather
// than delegating to the extension's renderer for the parts we don't change.
func (r *SourceLineRenderer) renderFootnote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*extAST.Footnote)
	if entering {
		fmt.Fprintf(w, "<li id=\"fn:%d\"%s>\n", n.Index, r.sourceLineAttr(node))
	} else {
		w.WriteString("</li>\n")
	}
	return ast.WalkContinue, nil
}

// renderFootnoteList wraps the <div class="footnotes"> container that holds
// all footnote definitions, so the whole footnotes section participates in
// scroll-sync (anchored to the first footnote's source line).
func (r *SourceLineRenderer) renderFootnoteList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		fmt.Fprintf(w, "<div class=\"footnotes\" role=\"doc-endnotes\"%s>\n<hr>\n<ol>\n", r.sourceLineAttr(node))
	} else {
		w.WriteString("</ol>\n</div>\n")
	}
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
