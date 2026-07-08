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
	"log"
	"maps"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/liubang/folio.nvim/internal/protocol"
)

// connEntry wraps a WebSocket connection with a per-connection write mutex
// to satisfy gorilla/websocket's "one concurrent writer" requirement.
type connEntry struct {
	conn *websocket.Conn
	wmu  sync.Mutex
}

func newConnEntry(conn *websocket.Conn) *connEntry {
	return &connEntry{conn: conn}
}

// send writes a text message to the connection, serialized against any
// concurrent writer via wmu and bounded by wsWriteTimeout.
func (c *connEntry) send(data []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// close sends a WebSocket close frame (best-effort) and closes the
// underlying connection, serialized against any in-flight send via wmu.
func (c *connEntry) close(reason string) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(shutdownWriteDeadline))
	c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, reason))
	c.conn.Close()
}

// bufferHub owns all per-buffer server-side state: the set of connected
// WebSocket clients, the last rendered payload (replayed to late-joining
// clients), and the buffer's working directory (for resolving relative
// asset paths). Extracting this from Server keeps the HTTP/lifecycle
// concerns in server.go separate from the concurrent bookkeeping here.
type bufferHub struct {
	mu          sync.RWMutex
	clients     map[int]map[*connEntry]struct{}   // bufnr → set of connections
	lastContent map[int]*protocol.OutgoingMessage // cached last render per bufnr
	workDirs    map[int]string                    // bufnr → markdown file directory
}

func newBufferHub() *bufferHub {
	return &bufferHub{
		clients:     make(map[int]map[*connEntry]struct{}),
		lastContent: make(map[int]*protocol.OutgoingMessage),
		workDirs:    make(map[int]string),
	}
}

// register adds entry to bufnr's client set and returns the JSON-encoded
// cached render for bufnr, if any, so the caller can replay it to the new
// client. The write itself is left to the caller so that it happens outside
// the hub's lock.
func (h *bufferHub) register(bufnr int, entry *connEntry) []byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[bufnr] == nil {
		h.clients[bufnr] = make(map[*connEntry]struct{})
	}
	h.clients[bufnr][entry] = struct{}{}

	cached, ok := h.lastContent[bufnr]
	if !ok {
		return nil
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return nil
	}
	return data
}

// unregister removes entry from bufnr's client set.
func (h *bufferHub) unregister(bufnr int, entry *connEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients[bufnr], entry)
}

// setWorkDir records the markdown file's directory for bufnr.
func (h *bufferHub) setWorkDir(bufnr int, dir string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.workDirs[bufnr] = dir
}

// workDir returns the recorded working directory for bufnr, or "" if none.
func (h *bufferHub) workDir(bufnr int) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.workDirs[bufnr]
}

// setLastContent caches the most recent render for bufnr so late-connecting
// clients can be brought up to date immediately.
func (h *bufferHub) setLastContent(bufnr int, msg *protocol.OutgoingMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastContent[bufnr] = msg
}

// broadcast sends data to every connection registered for bufnr, dropping
// (and unregistering) any connection whose write fails.
func (h *bufferHub) broadcast(bufnr int, data []byte) {
	// Collect the current set of connections under RLock so the slow path
	// (actual network writes) doesn't hold the hub lock.
	h.mu.RLock()
	entries := make([]*connEntry, 0, len(h.clients[bufnr]))
	for entry := range h.clients[bufnr] {
		entries = append(entries, entry)
	}
	h.mu.RUnlock()

	var failed []*connEntry
	for _, entry := range entries {
		if err := entry.send(data); err != nil {
			log.Printf("folio: write error to client (bufnr=%d): %v", bufnr, err)
			entry.conn.Close()
			failed = append(failed, entry)
		}
	}

	if len(failed) > 0 {
		h.mu.Lock()
		for _, entry := range failed {
			delete(h.clients[bufnr], entry)
		}
		h.mu.Unlock()
	}
}

// release drops all server-side state for bufnr: it closes every connected
// client and removes the cached render and working directory. Called when
// Neovim reports that a buffer has been closed, so long editing sessions
// don't accumulate stale entries indefinitely.
func (h *bufferHub) release(bufnr int) {
	h.mu.Lock()
	entries := h.clients[bufnr]
	delete(h.clients, bufnr)
	delete(h.lastContent, bufnr)
	delete(h.workDirs, bufnr)
	h.mu.Unlock()

	for entry := range entries {
		entry.close("buffer closed")
	}
}

// closeAll closes every connection across every buffer and clears all
// state. Used during server shutdown.
func (h *bufferHub) closeAll() {
	h.mu.Lock()
	allClients := maps.Clone(h.clients)
	h.clients = make(map[int]map[*connEntry]struct{})
	h.mu.Unlock()

	for _, entries := range allClients {
		for entry := range entries {
			entry.close("server shutting down")
		}
	}
}
