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
