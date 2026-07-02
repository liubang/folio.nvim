# folio.nvim

A high-performance, browser-level Markdown live preview plugin for Neovim.

Powered by a **Go sidecar** that parses Markdown and pushes rendered HTML to your
browser over WebSocket. The Lua plugin stays lean — no regex, no parsing, no
CPU-heavy work on the Neovim main thread. Styling is closely aligned with
**GitHub's Markdown rendering** for a familiar, polished look.

## Features

- **GitHub-style rendering** — headings, code blocks, tables, blockquotes,
  task lists, and admonitions all match GitHub's look and feel
- **Syntax highlighting** — code blocks are highlighted with
  [highlight.js](https://highlightjs.org/) (light + dark themes)
- **Math support** — inline and display math via
  [KaTeX](https://katex.org/)
- **Diagram support** — [Mermaid](https://mermaid.js.org/) diagrams rendered
  client-side
- **Admonitions** — GitHub-flavored `> [!NOTE]`, `> [!TIP]`, `> [!WARNING]`,
  `> [!IMPORTANT]`, `> [!CAUTION]` blockquotes with colored icons
- **Image lightbox** — click images to view full-size in an overlay
- **Code copy buttons** — hover over any code block to copy its contents
- **Scroll sync** — cursor movement in Neovim highlights and scrolls the
  browser preview via `data-source-line` anchor interpolation
- **Live reload** — buffer changes are pushed instantly (debounced) to the
  browser; no manual refresh needed
- **Dark mode** — auto-detects your OS `prefers-color-scheme` setting
- **Multi-buffer** — each Neovim buffer gets its own browser tab
- **Off-main-thread** — Markdown parsing and HTML generation run in a Go
  sidecar process; Neovim stays responsive
- **Graceful shutdown** — the Go process exits when Neovim closes stdin; no
  zombie processes

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
      -- port = 0,           -- TCP port (0 = auto-assign)
      -- debounce_ms = 150,  -- content sync debounce in ms
      -- auto_start = false, -- auto-open preview for markdown buffers
      -- filetypes = { "markdown" },
    })
  end,
  keys = {
    { "<Leader>mp", "<Cmd>FolioPreview<CR>", desc = "Markdown Preview" },
  },
}
```

### rocks.nvim

```lua
{
  "liubang/folio.nvim",
  config = function()
    require("folio").setup()
  end,
}
```

## Commands

| Command           | Description                                |
| ----------------- | ------------------------------------------ |
| `:FolioPreview`   | Open the Markdown preview in a browser tab |
| `:FolioClose`     | Close the preview for the current buffer   |

## API

```lua
-- Programmatic control
require("folio").open()       -- start preview for current buffer
require("folio").close()      -- stop preview for current buffer
require("folio").is_open()    -- returns true if preview is active
```

## Configuration

```lua
require("folio").setup({
  -- TCP port for the Go sidecar.  0 = auto-assign from the OS.
  port = 0,

  -- Debounce interval (milliseconds) for content synchronization.
  -- Lower values make preview updates more responsive but increase CPU usage.
  debounce_ms = 150,

  -- Path to the compiled Go binary.
  -- Defaults to <plugin-dir>/build/folio.
  binary = nil,

  -- Automatically open the preview when entering a markdown buffer.
  auto_start = false,

  -- Filetypes that folio treats as markdown.
  filetypes = { "markdown" },
})
```

## How It Works

```
┌──────────────┐   stdin (JSON)   ┌──────────────────┐   WebSocket   ┌───────────────┐
│  Neovim Lua  │ ───────────────> │   Go Sidecar     │ ────────────> │   Browser     │
│  (events)    │                  │  (goldmark + ws) │               │  (HTML + JS)  │
└──────────────┘                  └──────────────────┘               └───────────────┘
```

1. The Lua plugin listens to buffer changes (`nvim_buf_attach`) and cursor
   movements (`CursorMoved` / `CursorMovedI`).
2. Events are serialized as JSON and written to the Go sidecar's stdin.
3. The Go sidecar parses Markdown with a custom
   [Goldmark](https://github.com/yuin/goldmark) renderer that injects
   `data-source-line` attributes into every block-level HTML element.
4. Rendered HTML is broadcast to connected browsers over WebSocket.
5. The browser's JavaScript highlights the cursor line and scrolls the
   preview to keep it in sync with Neovim.

## Development

```bash
# Clone and enter the repo
git clone https://github.com/liubang/folio.nvim.git
cd folio.nvim

# Build the Go backend
make build

# Start the sidecar manually (useful for debugging)
make run

# Cross-compile for all platforms
make build-all

# Run tests
go test ./...
```

For local development with Neovim, point lazy.nvim at your checkout:

```lua
{ "liubang/folio.nvim", dir = "~/workspace/liubang/folio.nvim" }
```

## License

Apache 2.0 — see [LICENSE](./LICENSE).
