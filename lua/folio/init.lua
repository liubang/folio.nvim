-- folio.nvim — Browser-level Markdown live preview with Go sidecar.

local M = {}

--- Locate the plugin root directory via Neovim's runtime path.
--- Returns the directory that contains lua/folio/init.lua.
---@return string
local function find_plugin_dir()
  local markers = vim.api.nvim_get_runtime_file("lua/folio/init.lua", false)
  if markers[1] then
    -- lua/folio/init.lua  →  strip 3 path components to get plugin root
    return vim.fn.fnamemodify(markers[1], ":h:h:h")
  end
  return "."
end

---@class folio.Config
---@field port         integer  TCP port for the Go sidecar (0 = auto, default: 0)
---@field debounce_ms  integer  Debounce interval in ms for content sync (default: 150)
---@field binary       string   Path or name of the Go backend binary (default: "<plugin_dir>/build/folio")
---@field auto_start   boolean  Automatically open preview for markdown files (default: false)
---@field filetypes    string[] Filetypes to treat as markdown (default: {"markdown"})

---@type folio.Config
M.config = {
  port = 0,
  debounce_ms = 150,
  binary = "folio",
  auto_start = false,
  filetypes = { "markdown" },
}

--- setup(config?) — called by lazy.nvim or the user. Merges user config with defaults.
---@param opts? folio.Config
function M.setup(opts)
  -- If the user did not specify a binary path, resolve it from the plugin
  -- directory so it works out-of-the-box after `make build`.
  if not opts or not opts.binary then
    M.config.binary = find_plugin_dir() .. "/build/folio"
  end
  M.config = vim.tbl_deep_extend("force", M.config, opts or {})

  vim.api.nvim_create_user_command("FolioPreview", function()
    require("folio.buffer").open()
  end, { desc = "Open Markdown preview in browser" })

  vim.api.nvim_create_user_command("FolioClose", function()
    require("folio.buffer").close()
  end, { desc = "Close Markdown preview for the current buffer" })

  vim.api.nvim_create_user_command("FolioCloseAll", function()
    require("folio.buffer").close_all()
  end, { desc = "Close all Markdown previews" })

  if M.config.auto_start then
    local group = vim.api.nvim_create_augroup("FolioAuto", { clear = true })
    vim.api.nvim_create_autocmd({ "BufEnter", "BufWinEnter" }, {
      group = group,
      callback = function()
        if vim.tbl_contains(M.config.filetypes, vim.bo.filetype) then
          require("folio.buffer").open()
        end
      end,
    })
  end
end

--- is_open returns true if a preview is active for the current buffer.
function M.is_open()
  return require("folio.buffer").is_open()
end

--- open starts the preview for the current buffer.
function M.open()
  require("folio.buffer").open()
end

--- close stops the preview for the current buffer.
function M.close()
  require("folio.buffer").close()
end

--- close_all stops the preview for all buffers.
function M.close_all()
  require("folio.buffer").close_all()
end

return M
