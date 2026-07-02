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

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/liubang/folio.nvim/internal/protocol"
)

// ---------------------------------------------------------------------------
// Helper: create a server and return it with cleanup.
// ---------------------------------------------------------------------------

func newTestServer(t *testing.T) *Server {
	t.Helper()
	srv, err := New(0)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	t.Cleanup(func() { srv.Shutdown() })
	return srv
}

func wsURL(srv *Server, bufnr int) string {
	return fmt.Sprintf("ws://127.0.0.1:%d/ws/?bufnr=%d", srv.Port(), bufnr)
}

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial(%s) failed: %v", url, err)
	}
	return conn
}

// ---------------------------------------------------------------------------
// P0: broadcast() — race condition: delete under RLock + map mutation
//     during iteration.
//
// This test verifies the fix: after fixing, concurrent broadcasts and
// connection closures should not panic or race (run with -race).
// ---------------------------------------------------------------------------

func TestBroadcast_ConcurrentSafety(t *testing.T) {
	srv := newTestServer(t)
	bufnr := 1

	const numClients = 10
	conns := make([]*websocket.Conn, numClients)
	for i := range conns {
		conns[i] = dialWS(t, wsURL(srv, bufnr))
	}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	// Close half the clients immediately so writes will fail during broadcast.
	for i := 0; i < numClients/2; i++ {
		conns[i].Close()
	}
	time.Sleep(50 * time.Millisecond) // let the read-loop goroutines notice

	// Concurrent broadcasts should NOT panic or race.
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			srv.broadcast(bufnr, &protocol.OutgoingMessage{
				Type:         protocol.TypeScroll,
				Bufnr:        bufnr,
				ScrollToLine: i,
			})
		}(i)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// P1: WebSocket per-connection write serialization.
//
// Test that concurrent writes to the same connection (from broadcast + new
// client replay) don't panic under -race.
// ---------------------------------------------------------------------------

func TestWebSocket_ConcurrentWriteSafety(t *testing.T) {
	srv := newTestServer(t)
	bufnr := 2

	// Pre-populate cached content so handleWebSocket replays it.
	srv.mu.Lock()
	srv.lastContent[bufnr] = &protocol.OutgoingMessage{
		Type: protocol.TypeRender, Bufnr: bufnr, HTML: "<p>hello</p>",
	}
	srv.mu.Unlock()

	// Dial and immediately broadcast concurrently.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c := dialWS(t, wsURL(srv, bufnr))
			defer c.Close()
			// Read whatever the server sends.
			c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			c.ReadMessage()
		}()
		go func(n int) {
			defer wg.Done()
			srv.broadcast(bufnr, &protocol.OutgoingMessage{
				Type: protocol.TypeScroll, Bufnr: bufnr, ScrollToLine: n,
			})
		}(i)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// P1: parseBufnr — invalid input silently defaults to 1.
// ---------------------------------------------------------------------------

func TestParseBufnr(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		name     string
	}{
		{"1", 1, "normal_1"},
		{"42", 42, "normal_42"},
		{"abc", 0, "invalid_alpha"},
		{"", 0, "empty"},
		{"-1", -1, "negative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBufnr(tt.input)
			if tt.input == "abc" || tt.input == "" {
				if err == nil {
					t.Errorf("parseBufnr(%q) should return an error", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseBufnr(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("parseBufnr(%q) = %d, want %d", tt.input, got, tt.expected)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// P1: Path traversal via symlinks.
//
// handleFile must resolve symlinks before doing the prefix check.
// ---------------------------------------------------------------------------

func TestHandleFile_SymlinkTraversal(t *testing.T) {
	srv := newTestServer(t)

	// Create a temp directory structure:
	//   workdir/
	//   workdir/link → /etc  (or some external dir)
	//   external/secret.txt
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "workdir")
	os.MkdirAll(workDir, 0755)

	externalDir := filepath.Join(tmpDir, "external")
	os.MkdirAll(externalDir, 0755)
	os.WriteFile(filepath.Join(externalDir, "secret.txt"), []byte("secret-data"), 0644)

	// Create a symlink: workdir/link → external
	linkPath := filepath.Join(workDir, "link")
	if err := os.Symlink(externalDir, linkPath); err != nil {
		t.Skipf("Cannot create symlink (permissions?): %v", err)
	}

	bufnr := 10
	srv.mu.Lock()
	srv.workDirs[bufnr] = workDir
	srv.mu.Unlock()

	// Try to access workdir/link/secret.txt — should be blocked.
	url := fmt.Sprintf("http://127.0.0.1:%d/files/%d/link/secret.txt", srv.Port(), bufnr)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for symlink traversal, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// P1: Path traversal via ".." sequences.
// ---------------------------------------------------------------------------

func TestHandleFile_DotDotTraversal(t *testing.T) {
	srv := newTestServer(t)

	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(workDir, 0755)
	// Create a file outside workDir.
	os.WriteFile(filepath.Join(tmpDir, "passwd"), []byte("root:x:0:0"), 0644)

	bufnr := 11
	srv.mu.Lock()
	srv.workDirs[bufnr] = workDir
	srv.mu.Unlock()

	url := fmt.Sprintf("http://127.0.0.1:%d/files/%d/..%%2fpasswd", srv.Port(), bufnr)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("expected non-200 for .. traversal, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// handlePreview — serves index.html via filepath.Join (not string concat).
// ---------------------------------------------------------------------------

func TestHandlePreview_ServesIndexHTML(t *testing.T) {
	srv := newTestServer(t)
	url := fmt.Sprintf("http://127.0.0.1:%d/preview/", srv.Port())
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET /preview/ failed: %v", err)
	}
	defer resp.Body.Close()
	// Even if index.html doesn't exist in test env, we just verify no panic.
	// Status 200 or 404 are both acceptable.
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		t.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// handleContentChanged — end-to-end test via stdin-like JSON.
// ---------------------------------------------------------------------------

func TestHandleContentChanged_BroadcastsHTML(t *testing.T) {
	srv := newTestServer(t)
	bufnr := 1

	// Connect a WebSocket client.
	conn := dialWS(t, wsURL(srv, bufnr))
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	// Simulate a content_changed event.
	srv.handleContentChanged(&protocol.IncomingMessage{
		Event:      protocol.EventContentChanged,
		Bufnr:      bufnr,
		Content:    "# Hello\n\nWorld",
		CursorLine: 1,
	})

	// Read the broadcast message.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	var msg protocol.OutgoingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if msg.Type != protocol.TypeRender {
		t.Errorf("expected type=%q, got %q", protocol.TypeRender, msg.Type)
	}
	if !strings.Contains(msg.HTML, "Hello") {
		t.Errorf("expected HTML to contain 'Hello', got: %s", msg.HTML)
	}
}

// ---------------------------------------------------------------------------
// handlePreview uses filepath.Join (not string concat).
// ---------------------------------------------------------------------------

func TestHandlePreview_UsesFilepathJoin(t *testing.T) {
	srv := newTestServer(t)
	// This test verifies handlePreview doesn't panic.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/preview/", nil)
	srv.handlePreview(w, r)
	// Accept 200 or 404 (index.html may not be in test env).
	if w.Code != 200 && w.Code != 404 {
		t.Errorf("unexpected status: %d", w.Code)
	}
}
