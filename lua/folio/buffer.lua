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
---@field bufnr            integer
---@field attached         boolean
---@field content_timer    uv.uv_timer_t | nil   debounce timer for content updates
---@field cursor_timer     uv.uv_timer_t | nil   debounce timer for cursor updates
---@field autocmd_ids      integer[]              autocmd IDs for cleanup

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

    if not port or port == 0 then
      vim.notify("[folio] invalid port", vim.log.levels.ERROR)
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

--- close_all detaches all active buffers and stops the sidecar.
function M.close_all()
  for bufnr, _ in pairs(buffers) do
    M._detach(bufnr)
  end
  require("folio.server").stop()
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
  -- Return true from callback to auto-detach when buffer is no longer tracked.
  vim.api.nvim_buf_attach(bufnr, false, {
    on_lines = function()
      if not buffers[bufnr] then
        return true -- detach this callback
      end
      M._debounce(bufnr, "content_timer", function()
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
      M._debounce(bufnr, "cursor_timer", function()
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

  -- Cancel any pending debounce timers.
  M._close_timer(state, "content_timer")
  M._close_timer(state, "cursor_timer")

  -- Remove the augroup (also removes all autocmds in it).
  pcall(vim.api.nvim_del_augroup_by_name, "FolioBuf" .. bufnr)

  -- Notify the sidecar to release cached state for this buffer.
  M._send_buffer_closed(bufnr)

  buffers[bufnr] = nil
end

--- _close_timer stops and closes the uv timer stored under state[field], if any.
---@param state table   a folio.BufferState
---@param field string  field name holding the uv_timer_t
function M._close_timer(state, field)
  local timer = state[field]
  if not timer then
    return
  end
  timer:stop()
  if not timer:is_closing() then
    timer:close()
  end
  state[field] = nil
end

--- _debounce (re)starts a timer stored under buffers[bufnr][timer_field],
--- invoking callback (wrapped in vim.schedule) after delay_ms of inactivity.
--- Used for both content-change and cursor-move debouncing; the two are kept
--- as independent timers (different fields) so a burst of cursor movement
--- doesn't delay a pending content sync, and vice versa.
---@param bufnr       integer
---@param timer_field string  field name in the buffer state table
---@param callback    fun()
---@param delay_ms    integer
function M._debounce(bufnr, timer_field, callback, delay_ms)
  local state = buffers[bufnr]
  if not state then
    return
  end

  if not state[timer_field] then
    state[timer_field] = vim.uv.new_timer()
  else
    state[timer_field]:stop()
  end
  state[timer_field]:start(delay_ms, 0, function()
    vim.schedule(callback)
  end)
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

  -- Resolve the file's directory for relative-path assets (images, etc.),
  -- and its base name for the browser tab title.
  local filepath = vim.api.nvim_buf_get_name(bufnr)
  local work_dir = ""
  local filename = ""
  if filepath and filepath ~= "" then
    work_dir = vim.fn.fnamemodify(filepath, ":h")
    filename = vim.fn.fnamemodify(filepath, ":t")
  end

  -- Get cursor position from the window actually showing this buffer.
  local cursor_line = 1
  local winid = vim.fn.bufwinid(bufnr)
  if winid and winid ~= -1 then
    cursor_line = vim.api.nvim_win_get_cursor(winid)[1]
  end

  local ok, msg = pcall(encode, {
    event = "content_changed",
    bufnr = bufnr,
    content = content,
    cursor_line = cursor_line,
    work_dir = work_dir,
    filename = filename,
  })
  if not ok then
    vim.notify("[folio] JSON encode error: " .. tostring(msg), vim.log.levels.ERROR)
    return
  end

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
  local ok, msg = pcall(encode, {
    event = "cursor_moved",
    bufnr = bufnr,
    cursor_line = cursor[1],
  })
  if not ok then
    vim.notify("[folio] JSON encode error: " .. tostring(msg), vim.log.levels.ERROR)
    return
  end
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

--- _send_buffer_closed notifies the Go sidecar to release state for a buffer.
---@param bufnr integer
function M._send_buffer_closed(bufnr)
  local ok, msg = pcall(encode, {
    event = "buffer_closed",
    bufnr = bufnr,
  })
  if not ok then
    return
  end
  M._send(msg)
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
