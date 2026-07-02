-- health.lua — :checkhealth integration for folio.nvim.

local M = {}

function M.check()
  vim.health.start("folio.nvim")

  local binary = require("folio").config.binary or "folio"
  if vim.fn.executable(binary) == 1 then
    vim.health.ok("folio backend found: " .. binary)
  else
    vim.health.warn("folio backend not in $PATH: " .. binary .. " (run make build)")
  end

  if vim.fn.has("nvim-0.10") == 1 then
    vim.health.ok("Neovim >= 0.10 satisfied")
  else
    vim.health.error("Neovim 0.10+ required")
  end
end

return M
