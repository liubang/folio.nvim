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

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/liubang/folio.nvim/internal/server"
)

func main() {
	port := flag.Int("port", 0, "HTTP/WebSocket listen port (0 = auto-assign)")
	flag.Parse()

	srv, err := server.New(*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "folio: failed to start server: %v\n", err)
		os.Exit(1)
	}

	// Announce the assigned port on stdout so the Neovim Lua side can read it.
	// Explicitly flush: stdout may be fully buffered when connected to a pipe.
	fmt.Printf("PORT:%d\n", srv.Port())
	os.Stdout.Sync()

	// Handle OS signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		// Gracefully close the HTTP server and WebSocket clients. We then
		// exit explicitly: RunStdinLoop blocks on stdin (a TTY read when
		// run manually), and closing os.Stdin from another goroutine does
		// not reliably interrupt a blocked read on all platforms, so the
		// process might otherwise hang after Shutdown completes.
		srv.Shutdown()
		os.Exit(0)
	}()

	// Block on stdin — when Neovim exits or the pipe breaks, we self-terminate.
	srv.RunStdinLoop()
	srv.Shutdown() // idempotent (shutdownOnce)
}
