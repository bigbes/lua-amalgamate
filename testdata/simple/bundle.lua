-- Amalgamated by amalg
-- Entry: main

package.preload["main"] = function(...)
  local mod = require("module")
  
  print("Hello from main!")
  print("Module says: " .. mod.greet())
end

package.preload["module"] = function(...)
  local M = {}
  
  function M.greet()
      return "Hello from module!"
  end
  
  return M
end

require("main")
