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
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// P2: renderThematicBreak outputs <hr> twice (entering + leaving).
//
// After fix, there should be exactly one <hr> per "---".
// ---------------------------------------------------------------------------

func TestThematicBreak_NoDuplicate(t *testing.T) {
	r := NewRenderer()
	src := "paragraph one\n\n---\n\nparagraph two\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	count := strings.Count(html, "<hr")
	if count != 1 {
		t.Errorf("expected 1 <hr>, got %d in:\n%s", count, html)
	}
}

// ---------------------------------------------------------------------------
// P1: XSS via unsanitized language attribute in fenced code blocks.
//
// The language field must be HTML-escaped.
// ---------------------------------------------------------------------------

func TestFencedCodeBlock_XSSLanguage(t *testing.T) {
	r := NewRenderer()
	// Craft a malicious language string.
	src := "```\"><script>alert(1)</script>\ncode\n```\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)

	// The output must NOT contain unescaped <script>.
	if strings.Contains(html, "<script>") {
		t.Errorf("XSS: unescaped <script> found in output:\n%s", html)
	}
	// Should contain the escaped form.
	if !strings.Contains(html, "&lt;script&gt;") && !strings.Contains(html, "&#34;") {
		t.Errorf("expected HTML-escaped language attribute, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// Normal fenced code block rendering — language should appear correctly.
// ---------------------------------------------------------------------------

func TestFencedCodeBlock_Normal(t *testing.T) {
	r := NewRenderer()
	src := "```go\nfmt.Println(\"hello\")\n```\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	if !strings.Contains(html, `class="language-go"`) {
		t.Errorf("expected language-go class, got:\n%s", html)
	}
	if !strings.Contains(html, `data-lang="go"`) {
		t.Errorf("expected data-lang=go, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// P2: sourceLineAttr — verify correctness (line numbers).
//
// This also implicitly tests the O(n) → O(log n) optimization: the result
// should be the same regardless of the algorithm.
// ---------------------------------------------------------------------------

func TestSourceLine_Correctness(t *testing.T) {
	r := NewRenderer()
	// Line 1: "# Title"
	// Line 2: ""
	// Line 3: "paragraph"
	// Line 4: ""
	// Line 5: "---"
	src := "# Title\n\nparagraph\n\n---\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	// The heading should have data-source-line="1".
	if !strings.Contains(html, `data-source-line="1"`) {
		t.Errorf("expected heading at line 1, got:\n%s", html)
	}
	// The paragraph should have data-source-line="3".
	if !strings.Contains(html, `data-source-line="3"`) {
		t.Errorf("expected paragraph at line 3, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// sourceLineAttr performance: large document should not be O(n*m).
// ---------------------------------------------------------------------------

func BenchmarkConvert_LargeDoc(b *testing.B) {
	r := NewRenderer()
	// Build a document with 5000 paragraphs.
	var sb strings.Builder
	for i := 0; i < 5000; i++ {
		sb.WriteString("This is paragraph number ")
		sb.WriteString(strings.Repeat("word ", 20))
		sb.WriteString("\n\n")
	}
	src := []byte(sb.String())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Convert(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---------------------------------------------------------------------------
// Basic heading rendering.
// ---------------------------------------------------------------------------

func TestHeading_Rendering(t *testing.T) {
	r := NewRenderer()
	src := "# Hello\n## World\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	if !strings.Contains(html, "<h1") {
		t.Errorf("expected <h1>, got:\n%s", html)
	}
	if !strings.Contains(html, "<h2") {
		t.Errorf("expected <h2>, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// Code block rendering.
// ---------------------------------------------------------------------------

func TestCodeBlock_Rendering(t *testing.T) {
	r := NewRenderer()
	src := "    indented code\n    second line\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	if !strings.Contains(html, "<pre") {
		t.Errorf("expected <pre>, got:\n%s", html)
	}
	if !strings.Contains(html, "<code>") {
		t.Errorf("expected <code>, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// Blockquote rendering.
// ---------------------------------------------------------------------------

func TestBlockquote_Rendering(t *testing.T) {
	r := NewRenderer()
	src := "> quoted text\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	if !strings.Contains(html, "<blockquote") {
		t.Errorf("expected <blockquote>, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// List rendering.
// ---------------------------------------------------------------------------

func TestList_Rendering(t *testing.T) {
	r := NewRenderer()
	src := "- item one\n- item two\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	if !strings.Contains(html, "<ul") {
		t.Errorf("expected <ul>, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// GFM tables must carry a data-source-line attribute so scroll-sync works
// when the cursor is inside a table.
// ---------------------------------------------------------------------------

func TestTable_SourceLine(t *testing.T) {
	r := NewRenderer()
	src := "| A | B |\n|---|---|\n| 1 | 2 |\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	if !strings.Contains(html, "<table") {
		t.Errorf("expected <table>, got:\n%s", html)
	}
	if !strings.Contains(html, `data-source-line="1"`) {
		t.Errorf("expected table to have data-source-line=1, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// Regression: nodeStartLine must not recurse into inline nodes, whose Lines()
// panics inside goldmark. A table whose cells contain inline markup (links,
// emphasis) exercises this path.
// ---------------------------------------------------------------------------

func TestTable_WithInlineMarkup_NoPanic(t *testing.T) {
	r := NewRenderer()
	src := "| A | B |\n|---|---|\n| **bold** | [link](http://x) |\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if !strings.Contains(string(out), "<table") {
		t.Fatalf("expected <table> in output")
	}
}

// ---------------------------------------------------------------------------
// Headings get a GitHub-style slug id so the frontend TOC can deep-link to
// them via #anchor.
// ---------------------------------------------------------------------------

func TestHeading_SlugID(t *testing.T) {
	r := NewRenderer()
	src := "# Hello World\n## Getting Started!\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	if !strings.Contains(html, `id="hello-world"`) {
		t.Errorf("expected id=hello-world, got:\n%s", html)
	}
	if !strings.Contains(html, `id="getting-started"`) {
		t.Errorf("expected id=getting-started (punctuation stripped), got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// Duplicate heading text must get de-duplicated slugs, mirroring GitHub's
// -1, -2 suffixing behavior.
// ---------------------------------------------------------------------------

func TestHeading_SlugID_Deduplication(t *testing.T) {
	r := NewRenderer()
	src := "# Overview\n\ntext\n\n# Overview\n\nmore text\n\n# Overview\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	for _, id := range []string{`id="overview"`, `id="overview-1"`, `id="overview-2"`} {
		if !strings.Contains(html, id) {
			t.Errorf("expected %s, got:\n%s", id, html)
		}
	}
}

// ---------------------------------------------------------------------------
// Heading slug ids must be HTML-escaped / safe even with inline markup or
// unusual characters in the heading text (defense in depth alongside the
// language-attribute XSS test above).
// ---------------------------------------------------------------------------

func TestHeading_SlugID_SafeWithInlineMarkup(t *testing.T) {
	r := NewRenderer()
	src := "# **Bold** & <weird>\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	if strings.Contains(html, `id="**bold**`) {
		t.Errorf("slug must not retain markdown syntax, got:\n%s", html)
	}
	// The heading id attribute value itself must not break out of the
	// attribute (no unescaped quotes/angle brackets).
	if strings.Contains(html, `id="`) {
		start := strings.Index(html, `id="`) + len(`id="`)
		end := strings.Index(html[start:], `"`)
		if end == -1 {
			t.Fatalf("malformed id attribute in:\n%s", html)
		}
		idVal := html[start : start+end]
		if strings.ContainsAny(idVal, `<>"`) {
			t.Errorf("id attribute value contains unsafe characters: %q", idVal)
		}
	}
}

// ---------------------------------------------------------------------------
// Footnotes (extension.Footnote, PHP Markdown Extra syntax): `text[^1]` plus
// a `[^1]: definition` block. goldmark renders the reference as
// `<sup id="fnref:N"><a href="#fn:N" class="footnote-ref">N</a></sup>` and
// collects definitions into `<div class="footnotes"><ol><li id="fn:N">...`.
// ---------------------------------------------------------------------------

func TestFootnote_BasicRendering(t *testing.T) {
	r := NewRenderer()
	src := "Here is a claim[^1].\n\n[^1]: The supporting evidence.\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)

	// Reference marker in the body.
	if !strings.Contains(html, `id="fnref:1"`) {
		t.Errorf("expected fnref:1 marker, got:\n%s", html)
	}
	if !strings.Contains(html, `href="#fn:1"`) {
		t.Errorf("expected link to #fn:1, got:\n%s", html)
	}
	if !strings.Contains(html, `class="footnote-ref"`) {
		t.Errorf("expected footnote-ref class, got:\n%s", html)
	}

	// Footnote definition list at the end of the document.
	if !strings.Contains(html, `class="footnotes"`) {
		t.Errorf("expected footnotes container, got:\n%s", html)
	}
	if !strings.Contains(html, `id="fn:1"`) {
		t.Errorf("expected fn:1 definition, got:\n%s", html)
	}
	if !strings.Contains(html, "The supporting evidence.") {
		t.Errorf("expected footnote body text, got:\n%s", html)
	}

	// Backlink from the definition back to the reference.
	if !strings.Contains(html, `href="#fnref:1"`) {
		t.Errorf("expected backlink to #fnref:1, got:\n%s", html)
	}
	if !strings.Contains(html, `class="footnote-backref"`) {
		t.Errorf("expected footnote-backref class, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// The footnote definition (<li>) and its container (<div class="footnotes">)
// must carry data-source-line so scroll-sync keeps working when the cursor
// is at the bottom of the document on a footnote definition line.
// ---------------------------------------------------------------------------

func TestFootnote_SourceLine(t *testing.T) {
	r := NewRenderer()
	// Line 1: claim, line 2: blank, line 3: footnote definition.
	src := "Claim one[^a].\n\n[^a]: Definition text.\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)

	if !strings.Contains(html, `class="footnotes" role="doc-endnotes" data-source-line="3"`) {
		t.Errorf("expected footnotes container with data-source-line=3, got:\n%s", html)
	}
	if !strings.Contains(html, `id="fn:1" data-source-line="3"`) {
		t.Errorf("expected fn:1 li with data-source-line=3, got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// A footnote referenced multiple times gets one <sup> per reference (with
// distinct fnref ids) but a single shared definition, with one backlink per
// reference.
// ---------------------------------------------------------------------------

func TestFootnote_MultipleReferences(t *testing.T) {
	r := NewRenderer()
	src := "First[^dup] and second[^dup] mention.\n\n[^dup]: Shared note.\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)

	if !strings.Contains(html, `id="fnref:1"`) {
		t.Errorf("expected first reference fnref:1, got:\n%s", html)
	}
	if !strings.Contains(html, `id="fnref1:1"`) {
		t.Errorf("expected second reference fnref1:1, got:\n%s", html)
	}
	if strings.Count(html, `id="fn:1"`) != 1 {
		t.Errorf("expected exactly one shared definition fn:1, got:\n%s", html)
	}
	if strings.Count(html, `class="footnote-backref"`) != 2 {
		t.Errorf("expected two backlinks (one per reference), got:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// A `[^x]: ...` definition with no corresponding `[^x]` reference in the body
// must not be rendered at all (matches goldmark/PHP Markdown Extra behavior).
// ---------------------------------------------------------------------------

func TestFootnote_UndefinedReferenceOmitted(t *testing.T) {
	r := NewRenderer()
	src := "No references here.\n\n[^orphan]: Never used.\n"
	out, err := r.Convert([]byte(src))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	html := string(out)
	if strings.Contains(html, "footnotes") || strings.Contains(html, "Never used") {
		t.Errorf("orphan footnote definition should be dropped, got:\n%s", html)
	}
}
