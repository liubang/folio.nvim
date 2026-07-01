-- health.lua — :checkhealth integration for folio.nvim.

local M = {}

function M.check()
  local health = vim.health or require("health")

  local start = health.start or health.report_start
  local ok = health.ok or health.report_ok
  local warn = health.warn or health.report_warn
  local err = health.error or health.report_error

  start("folio.nvim")

  local binary = require("folio").config.binary or "folio"
  if vim.fn.executable(binary) == 1 then
    ok("folio backend found: " .. binary)
  else
    warn("folio backend not in $PATH: " .. binary .. " (run make build)")
  end

  if vim.fn.has("nvim-0.10") == 1 then
    ok("Neovim >= 0.10 satisfied")
  else
    err("Neovim 0.10+ required")
  end
end

return M
