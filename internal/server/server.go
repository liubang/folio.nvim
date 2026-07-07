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
// Created: 2026/07/02 00:09

package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/liubang/folio.nvim" // embedded frontend assets (package assets)
	"github.com/liubang/folio.nvim/internal/markdown"
	"github.com/liubang/folio.nvim/internal/protocol"

	"github.com/gorilla/websocket"
)

// Server is the HTTP + WebSocket sidecar that bridges Neovim (via stdin) and
// the browser (via WebSocket).
// connEntry wraps a WebSocket connection with a per-connection write mutex
// to satisfy gorilla/websocket's "one concurrent writer" requirement.
type connEntry struct {
	conn *websocket.Conn
	wmu  sync.Mutex
}

type Server struct {
	port    int
	httpSrv *http.Server

	renderer *markdown.Renderer

	mu          sync.RWMutex
	clients     map[int]map[*connEntry]struct{}    // bufnr → set of connections
	lastContent map[int]*protocol.OutgoingMessage  // cached last render per bufnr
	workDirs    map[int]string                     // bufnr → markdown file directory

	shutdownOnce sync.Once
}

// New creates a Server listening on the given port (0 = auto-assign).
func New(port int) (*Server, error) {
	s := &Server{
		renderer:    markdown.NewRenderer(),
		clients:     make(map[int]map[*connEntry]struct{}),
		lastContent: make(map[int]*protocol.OutgoingMessage),
		workDirs:    make(map[int]string),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/preview/", s.handlePreview)
	mux.HandleFunc("/ws/", s.handleWebSocket)
	mux.HandleFunc("/files/", s.handleFile)

	// Serve embedded static assets (index.html, vendored libs) from the
	// go:embed FS so the binary is fully self-contained.
	frontendFS, err := fs.Sub(assets.Files, "frontend")
	if err != nil {
		return nil, fmt.Errorf("embed frontend: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendFS)))

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	s.port = listener.Addr().(*net.TCPAddr).Port

	s.httpSrv = &http.Server{
		Handler: mux,
		// Conservative timeouts for local-only usage.
		ReadHeaderTimeout: 5 * time.Second, // mitigates slowloris-style stalls
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		if err := s.httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("folio: http serve error: %v", err)
		}
	}()

	return s, nil
}

// Port returns the TCP port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // localhost-only
}

// parseBufnr extracts the buffer number from a URL path segment or query string.
// Returns an error if the input is empty or not a valid integer.
func parseBufnr(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty bufnr")
	}
	// strconv.Atoi rejects partial matches like "12abc" (which Sscanf would
	// silently accept), giving stricter validation.
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid bufnr %q: %w", s, err)
	}
	return n, nil
}

// handlePreview serves the single-page HTML that opens a WebSocket connection
// for live preview. The HTML is read from the embedded frontend/ FS.
func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	data, err := assets.Files.ReadFile("frontend/index.html")
	if err != nil {
		http.Error(w, "index.html not found in embedded assets", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// handleFile serves a local file from the markdown file's directory.
// URL format: /files/{bufnr}/path/to/file.png
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/files/"), "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing bufnr", http.StatusBadRequest)
		return
	}
	bufnr, err := parseBufnr(parts[0])
	if err != nil {
		http.Error(w, "invalid bufnr", http.StatusBadRequest)
		return
	}
	relPath := ""
	if len(parts) > 1 {
		relPath = parts[1]
	}
	if relPath == "" {
		// Refuse bare directory requests — otherwise http.ServeFile would
		// enumerate the work directory contents to the browser.
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	s.mu.RLock()
	workDir := s.workDirs[bufnr]
	s.mu.RUnlock()

	if workDir == "" {
		http.Error(w, "no work directory for buffer", http.StatusNotFound)
		return
	}

	// Prevent path traversal: resolve symlinks before checking the prefix
	// so that a symlink pointing outside workDir is correctly rejected.
	cleanWork, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	fullPath := filepath.Clean(filepath.Join(workDir, relPath))
	evalPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !strings.HasPrefix(evalPath, cleanWork+string(os.PathSeparator)) && evalPath != cleanWork {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	http.ServeFile(w, r, fullPath)
}

// handleWebSocket upgrades an HTTP connection and registers the client for
// the given bufnr.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Validate bufnr *before* upgrading: refusing a bad request with a plain
	// 400 is cleaner than upgrading a WebSocket only to close it immediately,
	// and avoids silently bucketing the client onto buffer 1 (which would mix
	// content and scroll events across unrelated buffers).
	bufnr, parseErr := parseBufnr(r.URL.Query().Get("bufnr"))
	if parseErr != nil {
		http.Error(w, "invalid or missing bufnr", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("folio: websocket upgrade error: %v", err)
		return
	}

	entry := &connEntry{conn: conn}

	s.mu.Lock()
	if s.clients[bufnr] == nil {
		s.clients[bufnr] = make(map[*connEntry]struct{})
	}
	s.clients[bufnr][entry] = struct{}{}

	// Snapshot the cached content under the lock, but perform the actual
	// WebSocket write *after* releasing s.mu. Writing while holding s.mu would
	// block every other client (new connections, broadcasts, cursor events)
	// for up to the write deadline if this client is slow to drain.
	var cachedData []byte
	if cached, ok := s.lastContent[bufnr]; ok {
		if data, err := json.Marshal(cached); err == nil {
			cachedData = data
		}
	}
	s.mu.Unlock()

	if len(cachedData) > 0 {
		entry.wmu.Lock()
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, cachedData); err != nil {
			log.Printf("folio: replay write error (bufnr=%d): %v", bufnr, err)
		}
		entry.wmu.Unlock()
	}

	// Read loop — drains any client→server messages (future: click-to-scroll-back).
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.clients[bufnr], entry)
			s.mu.Unlock()
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

// RunStdinLoop blocks, reading JSON messages from stdin (sent by the Neovim
// Lua plugin) and broadcasting rendered HTML to connected browsers.
func (s *Server) RunStdinLoop() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 1 MB initial, 10 MB max

	for scanner.Scan() {
		var msg protocol.IncomingMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			log.Printf("folio: failed to parse stdin message: %v", err)
			continue
		}

		switch msg.Event {
		case protocol.EventContentChanged:
			s.handleContentChanged(&msg)
		case protocol.EventCursorMoved:
			s.handleCursorMoved(&msg)
		case protocol.EventBufferClosed:
			s.handleBufferClosed(&msg)
		default:
			log.Printf("folio: unknown event: %s", msg.Event)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("folio: stdin scanner error: %v", err)
	}
	log.Println("folio: stdin closed, shutting down")
}

func (s *Server) handleContentChanged(msg *protocol.IncomingMessage) {
	// Store work directory for serving relative-path assets (images, etc.).
	if msg.WorkDir != "" {
		s.mu.Lock()
		s.workDirs[msg.Bufnr] = msg.WorkDir
		s.mu.Unlock()
	}

	html, err := s.renderer.Convert([]byte(msg.Content))
	if err != nil {
		log.Printf("folio: render error: %v", err)
		return
	}

	out := &protocol.OutgoingMessage{
		Type:         protocol.TypeRender,
		Bufnr:        msg.Bufnr,
		HTML:         string(html),
		ScrollToLine: msg.CursorLine,
		Filename:     msg.Filename,
	}
	// Cache so that late-connecting clients get content immediately.
	s.mu.Lock()
	s.lastContent[msg.Bufnr] = out
	s.mu.Unlock()
	s.broadcast(msg.Bufnr, out)
}

func (s *Server) handleCursorMoved(msg *protocol.IncomingMessage) {
	out := protocol.OutgoingMessage{
		Type:         protocol.TypeScroll,
		Bufnr:        msg.Bufnr,
		ScrollToLine: msg.CursorLine,
	}
	s.broadcast(msg.Bufnr, &out)
}

// handleBufferClosed releases all server-side state for a buffer that Neovim
// has detached from: closes any WebSocket clients for the buffer and drops the
// cached render and the work-directory mapping. Without this, long editing
// sessions would accumulate stale entries in lastContent/workDirs indefinitely.
func (s *Server) handleBufferClosed(msg *protocol.IncomingMessage) {
	s.mu.Lock()
	if entries, ok := s.clients[msg.Bufnr]; ok {
		for entry := range entries {
			entry.wmu.Lock()
			entry.conn.Close()
			entry.wmu.Unlock()
		}
		delete(s.clients, msg.Bufnr)
	}
	delete(s.lastContent, msg.Bufnr)
	delete(s.workDirs, msg.Bufnr)
	s.mu.Unlock()
}

func (s *Server) broadcast(bufnr int, msg *protocol.OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("folio: marshal error: %v", err)
		return
	}

	// Collect the current set of connections under RLock.
	s.mu.RLock()
	entries := make([]*connEntry, 0, len(s.clients[bufnr]))
	for entry := range s.clients[bufnr] {
		entries = append(entries, entry)
	}
	s.mu.RUnlock()

	// Write to each connection using its per-connection mutex.
	// Collect failed connections for removal.
	var failed []*connEntry
	for _, entry := range entries {
		entry.wmu.Lock()
		entry.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := entry.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("folio: write error to client (bufnr=%d): %v", bufnr, err)
			entry.conn.Close()
			failed = append(failed, entry)
		}
		entry.wmu.Unlock()
	}

	// Remove failed connections under a full write lock.
	if len(failed) > 0 {
		s.mu.Lock()
		for _, entry := range failed {
			delete(s.clients[bufnr], entry)
		}
		s.mu.Unlock()
	}
}

// Shutdown gracefully stops the HTTP server and closes all WebSocket connections.
// Safe to call multiple times — all calls after the first are no-ops.
func (s *Server) Shutdown() {
	s.shutdownOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s.mu.Lock()
		for _, entries := range s.clients {
			for entry := range entries {
				// Take the per-connection write lock: a broadcast goroutine may
				// currently be mid-write (it releases s.mu before acquiring wmu).
				// gorilla/websocket forbids concurrent writers on the same
				// connection, so we must serialize here too.
				entry.wmu.Lock()
				entry.conn.SetWriteDeadline(time.Now().Add(time.Second))
				entry.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"))
				entry.conn.Close()
				entry.wmu.Unlock()
			}
		}
		s.clients = make(map[int]map[*connEntry]struct{})
		s.mu.Unlock()

		if err := s.httpSrv.Shutdown(ctx); err != nil {
			log.Printf("folio: http shutdown error: %v", err)
		}
		log.Println("folio: shutdown complete")
	})
}
