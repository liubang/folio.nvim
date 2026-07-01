-- buffer.lua — Buffer-level event handling: attach, content sync, cursor sync.

local M = {}

-- Cache JSON encoder at module load time.
local encode
do
  local ok, cjson = pcall(require, "cjson")
  if ok then
    encode = cjson.encode
  else
    encode = vim.json.encode
  end
end

---@class folio.BufferState
---@field bufnr          integer
---@field attached       boolean
---@field debounce_timer uv.uv_timer_t | nil
---@field autocmd_ids    integer[]            autocmd IDs for cleanup

---@type table<integer, folio.BufferState>
local buffers = {}

--- open starts the Go sidecar (if needed) and attaches to the current buffer.
function M.open()
  local bufnr = vim.api.nvim_get_current_buf()
  local config = require("folio").config
  if not vim.tbl_contains(config.filetypes, vim.bo[bufnr].filetype) then
    vim.notify("[folio] not a markdown buffer", vim.log.levels.WARN)
    return
  end

  local server = require("folio.server")
  server.start(function(port, err)
    if err then
      vim.notify("[folio] " .. err, vim.log.levels.ERROR)
      return
    end

    local url = "http://127.0.0.1:" .. port .. "/preview/?bufnr=" .. bufnr
    M._open_browser(url)
    M._attach(bufnr)
  end)
end

--- close detaches from the current buffer and stops the sidecar if no buffers remain.
function M.close()
  local bufnr = vim.api.nvim_get_current_buf()
  M._detach(bufnr)

  if vim.tbl_count(buffers) == 0 then
    require("folio.server").stop()
  end
end

--- is_open returns true if the current buffer has an active preview.
---@return boolean
function M.is_open()
  local bufnr = vim.api.nvim_get_current_buf()
  return buffers[bufnr] ~= nil and buffers[bufnr].attached
end

--- _attach registers buffer change and cursor-move callbacks for the given bufnr.
---@param bufnr integer
function M._attach(bufnr)
  if buffers[bufnr] and buffers[bufnr].attached then
    return
  end

  local config = require("folio").config
  local state = { bufnr = bufnr, attached = true, autocmd_ids = {} }
  buffers[bufnr] = state

  -- nvim_buf_attach for efficient incremental content tracking.
  vim.api.nvim_buf_attach(bufnr, false, {
    on_lines = function()
      M._debounce(bufnr, function()
        M._send_content(bufnr)
      end, config.debounce_ms)
    end,
  })

  -- Cursor-move autocmd (one per buffer). Store ID for cleanup in _detach.
  local group = vim.api.nvim_create_augroup("FolioBuf" .. bufnr, { clear = true })
  local id = vim.api.nvim_create_autocmd({ "CursorMoved", "CursorMovedI" }, {
    group = group,
    buffer = bufnr,
    callback = function()
      M._debounce(bufnr, function()
        M._send_cursor(bufnr)
      end, 50)
    end,
  })
  table.insert(state.autocmd_ids, id)

  -- Send initial content immediately.
  M._send_content(bufnr)
end

--- _detach removes buffer listeners and cleans up resources.
---@param bufnr integer
function M._detach(bufnr)
  local state = buffers[bufnr]
  if not state then
    return
  end

  -- Cancel pending timer.
  if state.debounce_timer then
    state.debounce_timer:stop()
    state.debounce_timer:close()
    state.debounce_timer = nil
  end

  -- Remove cursor-move autocmds.
  for _, id in ipairs(state.autocmd_ids) do
    pcall(vim.api.nvim_del_autocmd, id)
  end

  buffers[bufnr] = nil
end

--- _debounce wraps a callback with a per-buffer debounce timer.
--- Creates the timer once and uses timer:again() to reset it.
---@param bufnr   integer
---@param callback fun()
---@param delay_ms integer
function M._debounce(bufnr, callback, delay_ms)
  local state = buffers[bufnr]
  if not state then
    return
  end

  if state.debounce_timer then
    -- Reuse existing timer — no syscall for create/close.
    state.debounce_timer:again(delay_ms, 0, function()
      vim.schedule(callback)
    end)
  else
    state.debounce_timer = vim.uv.new_timer()
    state.debounce_timer:start(delay_ms, 0, function()
      vim.schedule(callback)
    end)
  end
end

--- _send_content reads the full buffer content and sends it to the Go sidecar.
---@param bufnr integer
function M._send_content(bufnr)
  if not vim.api.nvim_buf_is_valid(bufnr) then
    M._detach(bufnr)
    return
  end

  local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  local content = table.concat(lines, "\n")

  -- Resolve the file's directory for relative-path assets (images, etc.).
  local filepath = vim.api.nvim_buf_get_name(bufnr)
  local work_dir = ""
  if filepath and filepath ~= "" then
    work_dir = vim.fn.fnamemodify(filepath, ":h")
  end

  -- Get cursor position from the window actually showing this buffer.
  local cursor_line = 1
  local winid = vim.fn.bufwinid(bufnr)
  if winid and winid ~= -1 then
    cursor_line = vim.api.nvim_win_get_cursor(winid)[1]
  end

  local msg = encode({
    event = "content_changed",
    bufnr = bufnr,
    content = content,
    cursor_line = cursor_line,
    work_dir = work_dir,
  })

  M._send(msg)
end

--- _send_cursor sends cursor position to the Go sidecar.
---@param bufnr integer
function M._send_cursor(bufnr)
  local winid = vim.fn.bufwinid(bufnr)
  if not winid or winid == -1 then
    return
  end
  local cursor = vim.api.nvim_win_get_cursor(winid)
  local msg = encode({
    event = "cursor_moved",
    bufnr = bufnr,
    cursor_line = cursor[1],
  })
  M._send(msg)
end

--- _send writes a JSON message to the Go sidecar's stdin.
---@param msg string
function M._send(msg)
  local server = require("folio.server")
  local job_id = server._raw_job_id and server._raw_job_id()
  if not job_id then
    vim.notify("[folio] chansend skipped: no job_id", vim.log.levels.DEBUG)
    return
  end
  local ok = vim.fn.chansend(job_id, msg .. "\n")
  if ok == 0 then
    vim.notify("[folio] chansend FAILED for job_id=" .. job_id, vim.log.levels.ERROR)
  end
end

--- _open_browser opens the given URL in the system browser.
---@param url string
function M._open_browser(url)
  if vim.fn.has("mac") == 1 then
    vim.fn.jobstart({ "open", url }, { detach = true })
  elseif vim.fn.has("win32") == 1 then
    vim.fn.jobstart({ "cmd", "/c", "start", url }, { detach = true })
  else
    vim.fn.jobstart({ "xdg-open", url }, { detach = true })
  end
end

return M
