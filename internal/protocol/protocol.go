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
// Created: 2026/07/02 00:08

package protocol

// IncomingMessage is the JSON envelope received from Neovim over stdin.
type IncomingMessage struct {
	Event      string `json:"event"`
	Bufnr      int    `json:"bufnr"`
	Content    string `json:"content,omitempty"`
	CursorLine int    `json:"cursor_line"`
	WorkDir    string `json:"work_dir,omitempty"` // directory of the markdown file for resolving relative paths
}

// Event types sent by Neovim.
const (
	EventContentChanged = "content_changed"
	EventCursorMoved    = "cursor_moved"
)

// OutgoingMessage is the JSON envelope broadcast to browsers over WebSocket.
type OutgoingMessage struct {
	Type         string `json:"type"`
	Bufnr        int    `json:"bufnr"`
	HTML         string `json:"html,omitempty"`
	ScrollToLine int    `json:"scroll_to_line,omitempty"`
}

// Outgoing message types.
const (
	TypeRender = "render"
	TypeScroll = "scroll"
)
