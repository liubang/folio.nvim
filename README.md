# folio.nvim

A high-performance, browser-level Markdown live preview plugin for Neovim.

Powered by a **Go sidecar** that parses Markdown and pushes rendered HTML to your
browser over WebSocket. The Lua plugin stays lean — no regex, no parsing, no
CPU-heavy work on the Neovim main thread.

## Features

- **Native browser rendering** — full CSS layout, smooth scrolling, dark mode
- **Scroll sync** — cursor movement in Neovim scrolls the browser preview
  accurately via `data-source-line` anchor interpolation
- **Near-zero main-thread cost** — Markdown parsing and HTML generation run in a
  Go sidecar process, off the Neovim event loop
- **Multi-buffer** — each Neovim buffer gets its own browser tab
- **Graceful shutdown** — the Go process exits when Neovim closes the stdin pipe;
  no zombie processes
- **Auto light/dark theme** — follows your OS `prefers-color-scheme` setting

## Requirements

- Neovim ≥ 0.10
- Go ≥ 1.22 (only for compiling the backend; pre-built binaries provided in
  releases)
- A modern browser (Chrome, Firefox, Safari, Edge)

## Installation

### lazy.nvim

```lua
{
  "liubang/folio.nvim",
  cmd = { "FolioPreview", "FolioClose" },
  build = "make build",
  config = function()
    require("folio").setup({
      -- port = 0,           -- auto-assign TCP port (default)
      -- debounce_ms = 150,  -- debounce interval for content sync
      -- auto_start = false, -- set true to auto-open preview for .md files
      -- filetypes = { "markdown" },
    })
  end,
  keys = {
    { "<Leader>mp", "<Cmd>FolioPreview<CR>", desc = "Markdown Preview" },
  },
}
```

## Commands

| Command         | Description                                |
| --------------- | ------------------------------------------ |
| `:FolioPreview` | Open the Markdown preview in a browser tab |
| `:FolioClose`   | Close the preview for the current buffer   |

## How It Works

```
┌──────────────┐   stdin (JSON)   ┌──────────────────┐   WebSocket   ┌───────────────┐
│  Neovim Lua  │ ───────────────> │   Go Sidecar     │ ────────────> │   Browser     │
│  (events)    │                  │  (goldmark + ws) │               │  (HTML + JS)  │
└──────────────┘                  └──────────────────┘               └───────────────┘
```

1. The Lua plugin listens to buffer changes (`nvim_buf_attach`) and cursor
   movements.
2. Events are serialized as JSON and written to the Go sidecar's stdin.
3. The Go sidecar parses the Markdown with a custom Goldmark renderer that
   injects `data-source-line` attributes into block-level HTML elements.
4. The rendered HTML is broadcast to connected browsers over WebSocket.
5. The browser's JavaScript highlights the current line and scrolls smoothly.

## Development

```bash
# Clone and enter the repo
git clone https://github.com/liubang/folio.nvim.git
cd folio.nvim

# Build the Go backend
make build

# Run locally for testing
make run

# Cross-compile for all platforms
make build-all
```

For local development with Neovim, point lazy.nvim at your checkout:

```lua
{ "liubang/folio.nvim", dir = "~/workspace/liubang/folio.nvim" }
```

## License

Apache 2.0 — see [LICENSE](./LICENSE).
