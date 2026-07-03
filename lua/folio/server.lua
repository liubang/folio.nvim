-- server.lua — Go sidecar process lifecycle management.

local M = {}

---@class folio.ServerHandle
---@field job_id   integer  vim.fn.jobstart job id
---@field port     integer  TCP port the Go backend is listening on

---@type folio.ServerHandle | nil
local active_server = nil

--- start(callback) launches the Go sidecar binary as a job and reads its
--- announced port from stdout. The callback is invoked with (port, error).
---@param callback fun(port: integer|nil, err: string|nil)
function M.start(callback)
  if active_server and active_server.job_id then
    callback(active_server.port, nil)
    return
  end

  local config = require("folio").config
  local cmd = { config.binary }
  vim.list_extend(cmd, { "-port", tostring(config.port) })

  local port_found = false
  -- Accumulate stdout data to handle partial chunks.
  local stdout_buf = ""

  -- Pre-allocate the server handle so that on_stdout can safely access it.
  -- (Neovim's event loop guarantees on_stdout fires after this Lua call stack
  -- unwinds, but placing the assignment before jobstart makes the intent clear.)
  local handle = { job_id = nil, port = 0 }
  active_server = handle

  local job_id = vim.fn.jobstart(cmd, {
    on_stdout = function(_, data)
      if port_found then
        return
      end
      stdout_buf = stdout_buf .. table.concat(data, "\n")
      local port_str = stdout_buf:match("PORT:(%d+)")
      if port_str then
        port_found = true
        handle.port = tonumber(port_str)
        callback(handle.port, nil)
      end
    end,
    on_stderr = function(_, data)
      for _, line in ipairs(data) do
        if line ~= "" then
          vim.notify("[folio] " .. line, vim.log.levels.ERROR)
        end
      end
    end,
    on_exit = function(_, code)
      if code and code ~= 0 then
        vim.notify("[folio] backend exited with code " .. code, vim.log.levels.ERROR)
      end
      if active_server and active_server.job_id == job_id then
        active_server = nil
      end
    end,
  })

  if job_id <= 0 then
    active_server = nil
    callback(nil, "failed to start folio (is the binary in $PATH?)")
    return
  end

  -- Now that we have the real job_id, store it.
  handle.job_id = job_id

  -- Timeout: if the backend doesn't announce a port within 5 seconds, fail.
  vim.defer_fn(function()
    if active_server and not port_found then
      M.stop()
      callback(nil, "folio did not announce a port within 5 seconds")
    end
  end, 5000)
end

--- stop terminates the Go sidecar process.
function M.stop()
  if not active_server or not active_server.job_id then
    return
  end
  vim.fn.jobstop(active_server.job_id)
  active_server = nil
end

--- port returns the TCP port of the active server, or nil.
---@return integer|nil
function M.port()
  return active_server and active_server.port
end

--- is_running returns true if the sidecar is alive.
---@return boolean
function M.is_running()
  return active_server ~= nil
    and active_server.job_id ~= nil
    and active_server.job_id > 0
end

--- _raw_job_id returns the vim job id for chansend, or nil.
---@return integer|nil
function M._raw_job_id()
  return active_server and active_server.job_id
end

return M
